// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package mycnf

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mapFS builds the parser's three hooks over an in-memory set of files, so a
// test can describe a whole include chain as a literal.
func mapFS(files map[string]string) (FileReader, DirLister) {
	reader := func(path string) (string, error) {
		content, ok := files[filepath.Clean(path)]
		if !ok {
			return "", errors.New("no such file: " + path)
		}
		return content, nil
	}
	dirLister := func(dir string) ([]string, error) {
		dir = filepath.Clean(dir)
		var out []string
		for path := range files {
			if filepath.Dir(path) == dir {
				out = append(out, path)
			}
		}
		if out == nil {
			return nil, errors.New("no such directory: " + dir)
		}
		return out, nil
	}
	return reader, dirLister
}

func parseString(t *testing.T, content string) *Conf {
	t.Helper()
	reader, dirLister := mapFS(map[string]string{"/etc/my.cnf": content})
	conf, err := Parse("/etc/my.cnf", reader, dirLister)
	require.NoError(t, err)
	return conf
}

func TestParseGroupHeaders(t *testing.T) {
	conf := parseString(t, `
[mysqld]
port=3306
[ client ]
socket=/tmp/x.sock
[mysqld-8.0]
port=3307
[mysqld]
bind_address=127.0.0.1
`)
	// Group names are recorded in first-seen order, deduplicated.
	assert.Equal(t, []string{"mysqld", "client", "mysqld-8.0"}, conf.SectionNames())

	// A reopened group accumulates rather than replacing, and a
	// version-suffixed group stays a group of its own.
	sections := conf.Sections()
	require.Len(t, sections, 3)
	assert.Equal(t, "mysqld", sections[0].Name)
	assert.Len(t, sections[0].Options, 2, "port and bind_address, across both [mysqld] blocks")
	assert.Equal(t, "mysqld-8.0", sections[2].Name)
	assert.Len(t, sections[2].Options, 1)
}

// A group reopened in a later file must win over a version-suffixed group that
// appeared earlier. This is the reason Conf stores a flat option list in read
// order instead of grouping as it parses: grouping first would sort the
// reopened [mysqld] back to its original position and let [mysqld-8.0] win.
func TestParseReopenedGroupRespectsReadOrder(t *testing.T) {
	reader, dirLister := mapFS(map[string]string{
		"/etc/my.cnf":   "[mysqld]\nport=3306\n[mysqld-8.0]\nport=3307\n!include /etc/late.cnf\n",
		"/etc/late.cnf": "[mysqld]\nport=3308\n",
	})
	conf, err := Parse("/etc/my.cnf", reader, dirLister)
	require.NoError(t, err)

	merged := Merge(conf, "mysqld")
	assert.Equal(t, "3308", merged["port"], "the last file read must win")
}

func TestParseOptionForms(t *testing.T) {
	conf := parseString(t, `
[mysqld]
port=3306
socket = /var/run/mysqld/mysqld.sock
skip-name-resolve
empty_value=
loose-some_unknown_option=1
--datadir=/var/lib/mysql
`)
	opts := map[string]Option{}
	for _, o := range conf.Options {
		opts[o.Name] = o
	}

	assert.Equal(t, "3306", opts["port"].Value)
	assert.Equal(t, "/var/run/mysqld/mysqld.sock", opts["socket"].Value)

	// A bare option is in effect even though it carries no value.
	assert.True(t, opts["skip_name_resolve"].Bare)
	assert.Equal(t, "", opts["skip_name_resolve"].Value)

	// An explicit empty assignment is NOT bare. MySQL distinguishes the two.
	assert.False(t, opts["empty_value"].Bare)

	assert.True(t, opts["some_unknown_option"].Loose)
	assert.Equal(t, "1", opts["some_unknown_option"].Value)

	// A leading "--" is tolerated even though option files don't use it.
	assert.Equal(t, "/var/lib/mysql", opts["datadir"].Value)
}

// MySQL treats "-" and "_" as interchangeable in option names, so the two
// spellings are one option and must collapse to a single key. A parser that
// keeps them apart reports a value that is not in effect.
func TestParseSeparatorAliasing(t *testing.T) {
	conf := parseString(t, "[mysqld]\nbind-address=10.0.0.1\nbind_address=127.0.0.1\n")

	merged := Merge(conf, "mysqld")
	assert.Len(t, merged, 1, "both spellings must collapse to one key")
	assert.Equal(t, "127.0.0.1", merged["bind_address"], "last write wins across spellings")
}

func TestParseCaseInsensitiveNames(t *testing.T) {
	conf := parseString(t, "[mysqld]\nSecure-File-Priv=/var/lib/mysql-files\n")
	assert.Equal(t, "/var/lib/mysql-files", Merge(conf, "mysqld")["secure_file_priv"])
}

func TestParseQuotingAndEscapes(t *testing.T) {
	for _, tc := range []struct {
		name, line, want string
	}{
		{"double quoted", `k="a b"`, "a b"},
		{"single quoted", `k='a b'`, "a b"},
		{"hash inside quotes is literal", `k="pass#word"`, "pass#word"},
		{"semicolon inside value is literal", `k=a;b`, "a;b"},
		{"inline comment stripped", `k=value  # a note`, "value"},
		{"unquoted keeps inner spaces trimmed", `k=  spaced  `, "spaced"},
		{"escape space", `k=a\sb`, "a b"},
		{"escape tab", `k=a\tb`, "a\tb"},
		{"escape newline", `k=a\nb`, "a\nb"},
		{"escape backslash", `k=a\\b`, `a\b`},
		{"unknown escape kept literal", `k=a\qb`, `a\qb`},
		{"escaped quote inside quotes", `k="a\"b"`, `a"b`},
		{"unterminated quote takes rest", `k="abc`, "abc"},
		{"empty value", `k=`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conf := parseString(t, "[mysqld]\n"+tc.line+"\n")
			require.Len(t, conf.Options, 1)
			assert.Equal(t, tc.want, conf.Options[0].Value)
		})
	}
}

func TestParseComments(t *testing.T) {
	conf := parseString(t, `
# a hash comment
; a semicolon comment
[mysqld]
   # an indented comment
port=3306
`)
	require.Len(t, conf.Options, 1)
	assert.Equal(t, "port", conf.Options[0].Name)
}

// MySQL rejects options that appear before any group header, so attributing
// them to some arbitrary group would report settings that are not in effect.
func TestParseOptionsBeforeAnyGroupAreDropped(t *testing.T) {
	conf := parseString(t, "orphan=1\n[mysqld]\nport=3306\n")
	require.Len(t, conf.Options, 1)
	assert.Equal(t, "port", conf.Options[0].Name)
}

func TestParseCRLF(t *testing.T) {
	conf := parseString(t, "[mysqld]\r\nport=3306\r\nskip-name-resolve\r\n")
	merged := Merge(conf, "mysqld")
	assert.Equal(t, "3306", merged["port"])
	assert.Contains(t, Flags(conf, "mysqld"), "skip_name_resolve")
}

func TestParseEmptyAndCommentOnlyFiles(t *testing.T) {
	assert.Empty(t, parseString(t, "").Options)
	assert.Empty(t, parseString(t, "# nothing here\n\n; nor here\n").Options)
}

func TestParseLineAndFileProvenance(t *testing.T) {
	conf := parseString(t, "[mysqld]\n\nport=3306\n")
	require.Len(t, conf.Options, 1)
	assert.Equal(t, 3, conf.Options[0].Line)
	assert.Equal(t, "/etc/my.cnf", conf.Options[0].File)
	assert.Equal(t, "mysqld", conf.Options[0].Section)
}

func TestParseIncludeRelativeAndAbsolute(t *testing.T) {
	reader, dirLister := mapFS(map[string]string{
		"/etc/my.cnf":        "[mysqld]\n!include extra.cnf\n!include /etc/other/abs.cnf\n",
		"/etc/extra.cnf":     "[mysqld]\nport=3307\n",
		"/etc/other/abs.cnf": "[mysqld]\ndatadir=/data\n",
	})
	conf, err := Parse("/etc/my.cnf", reader, dirLister)
	require.NoError(t, err)

	merged := Merge(conf, "mysqld")
	assert.Equal(t, "3307", merged["port"])
	assert.Equal(t, "/data", merged["datadir"])
	assert.Len(t, conf.Files, 3)
}

func TestParseIncludeDirOnlyReadsCnfAndIni(t *testing.T) {
	reader, dirLister := mapFS(map[string]string{
		"/etc/my.cnf":                "[mysqld]\n!includedir /etc/conf.d\n",
		"/etc/conf.d/a.cnf":          "[mysqld]\na=1\n",
		"/etc/conf.d/b.ini":          "[mysqld]\nb=1\n",
		"/etc/conf.d/c.preset":       "[mysqld]\nc=1\n",
		"/etc/conf.d/d.cnf.disabled": "[mysqld]\nd=1\n",
		"/etc/conf.d/README":         "not a config\n",
	})
	conf, err := Parse("/etc/my.cnf", reader, dirLister)
	require.NoError(t, err)

	merged := Merge(conf, "mysqld")
	assert.Equal(t, "1", merged["a"])
	assert.Equal(t, "1", merged["b"])
	assert.NotContains(t, merged, "c", ".preset must be skipped")
	assert.NotContains(t, merged, "d", "a suffix after .cnf must be skipped")
}

// !includedir has no defined read order in MySQL. Sorting makes a scan
// reproducible, and it also decides which of two fragments setting the same
// option wins, so the order has to be asserted rather than assumed.
func TestParseIncludeDirIsSorted(t *testing.T) {
	reader, dirLister := mapFS(map[string]string{
		"/etc/my.cnf":          "[mysqld]\n!includedir /etc/conf.d\n",
		"/etc/conf.d/50-z.cnf": "[mysqld]\nport=50\n",
		"/etc/conf.d/99-a.cnf": "[mysqld]\nport=99\n",
		"/etc/conf.d/10-m.cnf": "[mysqld]\nport=10\n",
	})
	conf, err := Parse("/etc/my.cnf", reader, dirLister)
	require.NoError(t, err)
	assert.Equal(t, "99", Merge(conf, "mysqld")["port"], "highest-sorting fragment wins")
}

func TestParseIncludeDirRecorded(t *testing.T) {
	conf := parseString(t, "[mysqld]\n!includedir /etc/mysql/mariadb.conf.d/\n")
	assert.Equal(t, []string{"/etc/mysql/mariadb.conf.d"}, conf.Includes)
}

func TestParseIncludeCycleTerminates(t *testing.T) {
	reader, dirLister := mapFS(map[string]string{
		"/etc/my.cnf": "[mysqld]\na=1\n!include /etc/b.cnf\n",
		"/etc/b.cnf":  "[mysqld]\nb=1\n!include /etc/my.cnf\n",
	})
	conf, err := Parse("/etc/my.cnf", reader, dirLister)
	require.NoError(t, err)
	assert.Len(t, conf.Files, 2)
}

// Equivalent spellings of one path must collapse, or a cycle that walks
// through ".." never terminates.
func TestParseIncludeCycleThroughParentSegments(t *testing.T) {
	reader, dirLister := mapFS(map[string]string{
		"/etc/my.cnf":       "[mysqld]\n!include /etc/conf.d/../my.cnf\n!include /etc/conf.d/x.cnf\n",
		"/etc/conf.d/x.cnf": "[mysqld]\nx=1\n",
	})
	conf, err := Parse("/etc/my.cnf", reader, dirLister)
	require.NoError(t, err)
	assert.Equal(t, "1", Merge(conf, "mysqld")["x"])
}

// A dangling include is a normal state on a host where an optional package was
// removed. It must not blind the caller to the rest of the configuration.
func TestParseMissingIncludeIsSkipped(t *testing.T) {
	reader, dirLister := mapFS(map[string]string{
		"/etc/my.cnf": "[mysqld]\nport=3306\n!include /etc/gone.cnf\n!includedir /etc/gone.d\n",
	})
	conf, err := Parse("/etc/my.cnf", reader, dirLister)
	require.NoError(t, err)
	assert.Equal(t, "3306", Merge(conf, "mysqld")["port"])
}

func TestParseMissingRootIsAnError(t *testing.T) {
	reader, dirLister := mapFS(map[string]string{})
	_, err := Parse("/etc/my.cnf", reader, dirLister)
	assert.Error(t, err)
}

// plugin_load_add accumulates rather than overwriting. Distributions ship one
// fragment per pluggable component, so a last-write-wins merge would report
// one loaded plugin where several are in effect.
func TestParseCumulativePluginLoadAdd(t *testing.T) {
	reader, dirLister := mapFS(map[string]string{
		"[root]":                          "",
		"/etc/my.cnf":                     "[mysqld]\n!includedir /etc/conf.d\n",
		"/etc/conf.d/provider_bzip2.cnf":  "[server]\nplugin_load_add=provider_bzip2\n",
		"/etc/conf.d/provider_lz4.cnf":    "[server]\nplugin_load_add=provider_lz4\n",
		"/etc/conf.d/provider_lzma.cnf":   "[server]\nplugin_load_add=provider_lzma\n",
		"/etc/conf.d/provider_lzo.cnf":    "[server]\nplugin_load_add=provider_lzo\n",
		"/etc/conf.d/provider_snappy.cnf": "[server]\nplugin_load_add=provider_snappy\n",
	})
	conf, err := Parse("/etc/my.cnf", reader, dirLister)
	require.NoError(t, err)

	merged := Merge(conf, "mysqld", "server")
	assert.ElementsMatch(t,
		[]string{"provider_bzip2", "provider_lz4", "provider_lzma", "provider_lzo", "provider_snappy"},
		SplitList(merged["plugin_load_add"]),
		"every plugin_load_add occurrence must survive the merge")
}

// plugin_load, without the _add suffix, deliberately does replace.
func TestParsePluginLoadIsNotCumulative(t *testing.T) {
	conf := parseString(t, "[mysqld]\nplugin_load=a\nplugin_load=b\n")
	assert.Equal(t, "b", Merge(conf, "mysqld")["plugin_load"])
}

func TestFlagsAndLooseOptions(t *testing.T) {
	conf := parseString(t, `
[mysqld]
skip-name-resolve
skip_networking
loose-galera_option=1
port=3306
`)
	assert.ElementsMatch(t, []string{"skip_name_resolve", "skip_networking"}, Flags(conf, "mysqld"))
	assert.Equal(t, []string{"galera_option"}, LooseOptions(conf, "mysqld"))
}

// The skip/disable/enable prefixes must be left alone. skip_name_resolve and
// skip_networking are documented options in their own right, so rewriting them
// into an assignment on a shorter name would invent options that do not exist.
func TestNormalizeNameLeavesSkipPrefixAlone(t *testing.T) {
	for _, in := range []string{"skip-name-resolve", "skip_name_resolve"} {
		name, loose := NormalizeName(in)
		assert.Equal(t, "skip_name_resolve", name)
		assert.False(t, loose)
	}
}

func TestNormalizeNameStripsLoose(t *testing.T) {
	name, loose := NormalizeName("loose-innodb_buffer_pool_size")
	assert.Equal(t, "innodb_buffer_pool_size", name)
	assert.True(t, loose)

	// "loose" alone is not a prefix on an empty name.
	name, loose = NormalizeName("loose_")
	assert.Equal(t, "loose_", name)
	assert.False(t, loose)
}

func TestMatchesGroupVersionSuffix(t *testing.T) {
	for _, tc := range []struct {
		section, group string
		want           bool
	}{
		{"mysqld", "mysqld", true},
		{"mysqld-8.0", "mysqld", true},
		{"mysqld-8", "mysqld", true},
		{"mariadb-11.4", "mariadb", true},
		{"mariadb-10.11", "mariadb", true},

		// Prefix traps. The server never reads these as server scope, and
		// folding them in would silently widen every audit.
		{"mysqld_safe", "mysqld", false},
		{"mysqldump", "mysqld", false},
		{"mysqld-safe", "mysqld", false},
		{"mariadb-client", "mariadb", false},
		{"mariadb-dump", "mariadb", false},
		{"mariadb-admin", "mariadb", false},
		{"client-mariadb", "mariadb", false},

		{"client", "mysqld", false},
		{"mysqld", "server", false},
	} {
		t.Run(tc.section+"/"+tc.group, func(t *testing.T) {
			assert.Equal(t, tc.want, MatchesGroup(tc.section, tc.group))
		})
	}
}

func TestMergeVersionSuffixedGroupWins(t *testing.T) {
	conf := parseString(t, "[mysqld]\nport=3306\n[mysqld-8.0]\nport=3307\n")
	assert.Equal(t, "3307", Merge(conf, "mysqld")["port"])
}

func TestMergeExcludesUnnamedGroups(t *testing.T) {
	conf := parseString(t, "[mysqld]\nport=3306\n[mysqldump]\nquick\n[client]\nsocket=/tmp/s\n")
	merged := Merge(conf, "mysqld", "server")
	assert.Equal(t, map[string]string{"port": "3306"}, merged)
}

func TestIsTruthy(t *testing.T) {
	for _, v := range []string{"ON", "on", "1", "true", "TRUE", "yes", "Yes"} {
		assert.True(t, IsTruthy(v, false), v)
	}
	for _, v := range []string{"OFF", "off", "0", "false", "no", "", "maybe"} {
		assert.False(t, IsTruthy(v, false), v)
	}
	// A bare option is enabled regardless of its (empty) value.
	assert.True(t, IsTruthy("", true))
}

func TestSplitList(t *testing.T) {
	assert.Equal(t, []string{"TLSv1.2", "TLSv1.3"}, SplitList("TLSv1.2,TLSv1.3"))
	assert.Equal(t, []string{"TLSv1.2", "TLSv1.3"}, SplitList("TLSv1.2, TLSv1.3"))
	assert.Equal(t, []string{"a", "b"}, SplitList(`"a", 'b'`))
	assert.Nil(t, SplitList(""))
	assert.Nil(t, SplitList("  "))
	assert.Nil(t, SplitList(",,"))
}

// ---------------------------------------------------------------------------
// Real installations
// ---------------------------------------------------------------------------

// treeFS serves a captured installation tree from testdata. Absolute target
// paths are rewritten under the tree root, so the fixtures hold exactly the
// paths a real host uses and the include chains resolve unmodified.
func treeFS(t *testing.T, tree string) (FileReader, DirLister, FileProbe) {
	t.Helper()
	root := filepath.Join("testdata", tree)
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("fixture tree %q missing: %v", tree, err)
	}
	resolve := func(path string) string { return filepath.Join(root, filepath.Clean(path)) }

	reader := func(path string) (string, error) {
		b, err := os.ReadFile(resolve(path))
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	dirLister := func(dir string) ([]string, error) {
		entries, err := os.ReadDir(resolve(dir))
		if err != nil {
			return nil, err
		}
		var out []string
		for _, e := range entries {
			// Directories are skipped, matching how the server reads a
			// fragment directory. MariaDB parks a directory named
			// "99-enable-encryption.cnf.preset" inside its fragment
			// directory, so this is exercised by real fixtures.
			if e.IsDir() {
				continue
			}
			out = append(out, filepath.Join(filepath.Clean(dir), e.Name()))
		}
		return out, nil
	}
	probe := func(path string) (bool, bool) {
		fi, err := os.Stat(resolve(path))
		if err != nil {
			return false, false
		}
		return true, fi.IsDir()
	}
	return reader, dirLister, probe
}

// rootPath is the option file each captured installation actually starts from.
var fixtureRoots = map[string]string{
	"mysql80":            "/etc/my.cnf",
	"mysql84":            "/etc/my.cnf",
	"percona80":          "/etc/my.cnf",
	"almalinux9-mariadb": "/etc/my.cnf",
	"mariadb1011":        "/etc/mysql/my.cnf",
	"mariadb114":         "/etc/mysql/my.cnf",
	"deb13-mariadb":      "/etc/mysql/my.cnf",
	"ubuntu2404-mysql":   "/etc/mysql/my.cnf",
}

func parseTree(t *testing.T, tree string) (*Conf, FileProbe) {
	t.Helper()
	reader, dirLister, probe := treeFS(t, tree)
	conf, err := Parse(fixtureRoots[tree], reader, dirLister)
	require.NoError(t, err)
	return conf, probe
}

// The product a host runs cannot be inferred from a path or a binary name.
// Every one of these layouts is a real capture from the corresponding
// distribution or official image.
func TestDetectFlavorAcrossRealInstallations(t *testing.T) {
	for _, tc := range []struct {
		tree, want, why string
	}{
		{"mysql80", FlavorMySQL, "Oracle image: /etc/my.cnf with [mysqld]"},
		{"mysql84", FlavorMySQL, "Oracle image: /etc/my.cnf with [mysqld]"},
		{"percona80", FlavorMySQL, "Percona ships the MySQL layout"},
		{"ubuntu2404-mysql", FlavorMySQL, "apt mysql-server: !includedir mysql.conf.d"},
		{"mariadb1011", FlavorMariaDB, "official image: [mariadb] group"},
		{"mariadb114", FlavorMariaDB, "official image: [mariadbd] group, no [mysqld] at all"},
		{"deb13-mariadb", FlavorMariaDB, "apt default-mysql-server is MariaDB"},
		{"almalinux9-mariadb", FlavorMariaDB, "RPM layout is byte-identical to MySQL at the root file"},
	} {
		t.Run(tc.tree, func(t *testing.T) {
			conf, probe := parseTree(t, tc.tree)
			assert.Equal(t, tc.want, DetectFlavor(conf, probe), tc.why)
		})
	}
}

// On Debian and Ubuntu both products reach their root config through the same
// /etc/mysql/my.cnf link, so nothing about the path distinguishes them. This
// is the case that makes a path-based or binary-name-based gate wrong.
func TestDetectFlavorDebianAndUbuntuShareTheSameRootPath(t *testing.T) {
	mariadb, mariadbProbe := parseTree(t, "deb13-mariadb")
	mysql, mysqlProbe := parseTree(t, "ubuntu2404-mysql")

	assert.Equal(t, "/etc/mysql/my.cnf", fixtureRoots["deb13-mariadb"])
	assert.Equal(t, "/etc/mysql/my.cnf", fixtureRoots["ubuntu2404-mysql"])
	assert.Equal(t, FlavorMariaDB, DetectFlavor(mariadb, mariadbProbe))
	assert.Equal(t, FlavorMySQL, DetectFlavor(mysql, mysqlProbe))
}

// The RPM layout is the case that forces detection to run after include
// expansion: both products ship an /etc/my.cnf holding only a [client-server]
// header and !includedir /etc/my.cnf.d, and only the fragments name the product.
func TestDetectFlavorRequiresIncludeExpansion(t *testing.T) {
	reader, dirLister, probe := treeFS(t, "almalinux9-mariadb")

	// Without expansion the root file alone is not enough to tell.
	rootOnly, err := Parse("/etc/my.cnf", reader, nil)
	require.NoError(t, err)
	assert.NotEqual(t, FlavorMariaDB, DetectFlavor(rootOnly, nil),
		"the root file alone cannot identify the product")

	// With expansion the [mariadb] and [galera] groups settle it.
	expanded, err := Parse("/etc/my.cnf", reader, dirLister)
	require.NoError(t, err)
	assert.Equal(t, FlavorMariaDB, DetectFlavor(expanded, probe))
}

// MariaDB 11.4 configures the server under [mariadbd] and ships no [mysqld]
// group at all, so a server-scope view built only from [mysqld] and [server]
// comes back empty.
func TestServerScopeOnModernMariaDB(t *testing.T) {
	conf, _ := parseTree(t, "deb13-mariadb")
	assert.NotContains(t, conf.SectionNames(), "mysqld")

	mysqlOnly := Merge(conf, ServerGroups(FlavorMySQL)...)
	assert.NotContains(t, mysqlOnly, "bind_address",
		"MySQL's group set must not find MariaDB's server options")

	mariadb := Merge(conf, ServerGroups(FlavorMariaDB)...)
	assert.Equal(t, "127.0.0.1", mariadb["bind_address"])
	assert.Equal(t, "/run/mysqld/mysqld.pid", mariadb["pid_file"])
	assert.Equal(t, "/run/mysqld/mysqld.sock", mariadb["socket"],
		"[client-server] is server scope on MariaDB")
}

// The five provider_*.cnf fragments Debian's MariaDB packages install are the
// real case cumulative merging exists for.
func TestRealPluginLoadAddAccumulates(t *testing.T) {
	conf, _ := parseTree(t, "deb13-mariadb")
	plugins := SplitList(Merge(conf, ServerGroups(FlavorMariaDB)...)["plugin_load_add"])
	assert.ElementsMatch(t,
		[]string{"provider_bzip2", "provider_lz4", "provider_lzma", "provider_lzo", "provider_snappy"},
		plugins)
}

func TestRealMysqlServerScope(t *testing.T) {
	conf, _ := parseTree(t, "mysql80")
	merged := Merge(conf, ServerGroups(FlavorMySQL)...)
	assert.Equal(t, "/var/lib/mysql", merged["datadir"])
	assert.Equal(t, "/var/lib/mysql-files", merged["secure_file_priv"])
	assert.Equal(t, "mysql", merged["user"])
	assert.Contains(t, Flags(conf, ServerGroups(FlavorMySQL)...), "skip_name_resolve")

	client := Merge(conf, ClientGroups(FlavorMySQL)...)
	assert.Equal(t, "/var/run/mysqld/mysqld.sock", client["socket"])
}

func TestRealUbuntuMysqlServerScope(t *testing.T) {
	conf, _ := parseTree(t, "ubuntu2404-mysql")
	merged := Merge(conf, ServerGroups(FlavorMySQL)...)
	assert.Equal(t, "mysql", merged["user"])
	assert.Equal(t, "127.0.0.1", merged["bind_address"])
	assert.Equal(t, "/var/log/mysql/error.log", merged["log_error"])
}

// [mysqld_safe] sits in its own fragment on Debian's MariaDB, which is exactly
// where a prefix-matching group check would leak it into server scope.
func TestRealMysqldSafeStaysOutOfServerScope(t *testing.T) {
	conf, _ := parseTree(t, "deb13-mariadb")
	require.Contains(t, conf.SectionNames(), "mysqld_safe")

	server := Merge(conf, ServerGroups(FlavorMariaDB)...)
	safe := Merge(conf, "mysqld_safe")
	require.NotEmpty(t, safe, "the fixture must actually set something under [mysqld_safe]")

	for k := range safe {
		if _, alsoServer := server[k]; alsoServer {
			// Only flag options the server group set doesn't legitimately own.
			assert.NotEqual(t, safe[k], server[k],
				"option %q leaked from [mysqld_safe] into server scope", k)
		}
	}
}

// MariaDB's client tool groups ([mariadb-client], [mariadb-dump], ...) look
// like version-suffixed [mariadb] groups but are not, and must stay out of
// server scope.
func TestRealMariadbClientGroupsStayOutOfServerScope(t *testing.T) {
	conf, _ := parseTree(t, "mariadb114")
	require.Contains(t, conf.SectionNames(), "mariadb-client")
	assert.False(t, MatchesGroup("mariadb-client", "mariadb"))
}

func TestRealFixtureFilesAreAllRead(t *testing.T) {
	conf, _ := parseTree(t, "mariadb1011")
	// mariadb.cnf plus the five .cnf fragments; the .preset directory and the
	// .preset file inside it must not be read.
	assert.Len(t, conf.Files, 6)
	for _, f := range conf.Files {
		assert.NotContains(t, f, ".preset")
	}
}

func TestParseVersion(t *testing.T) {
	for _, tc := range []struct {
		name, banner, wantVersion, wantFlavor string
	}{
		{
			"mysql 8.0 oracle image",
			"/usr/sbin/mysqld  Ver 8.0.46 for Linux on aarch64 (MySQL Community Server - GPL)",
			"8.0.46", FlavorMySQL,
		},
		{
			"mysql 8.4 oracle image",
			"/usr/sbin/mysqld  Ver 8.4.11 for Linux on aarch64 (MySQL Community Server - GPL)",
			"8.4.11", FlavorMySQL,
		},
		{
			"mysql from ubuntu apt",
			"/usr/sbin/mysqld  Ver 8.0.46-0ubuntu0.24.04.3 for Linux on aarch64 ((Ubuntu))",
			"8.0.46", FlavorMySQL,
		},
		{
			"percona server",
			"/usr/sbin/mysqld  Ver 8.0.46-37 for Linux on aarch64 (Percona Server (GPL), Release 37, Revision 39e2b60e)",
			"8.0.46", FlavorPercona,
		},
		{
			// MariaDB installs a binary named mysqld, so the name proves nothing.
			"mariadb reporting through a mysqld name",
			"mysqld  Ver 10.11.18-MariaDB-ubu2204 for debian-linux-gnu on aarch64 (mariadb.org binary distribution)",
			"10.11.18", FlavorMariaDB,
		},
		{
			"mariadbd",
			"mariadbd  Ver 11.4.12-MariaDB-ubu2404 for debian-linux-gnu on aarch64 (mariadb.org binary distribution)",
			"11.4.12", FlavorMariaDB,
		},
		{
			"mariadb from debian apt",
			"mysqld  Ver 11.8.6-MariaDB-0+deb13u1 from Debian for debian-linux-gnu on aarch64",
			"11.8.6", FlavorMariaDB,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			version, flavor := ParseVersion(tc.banner)
			assert.Equal(t, tc.wantVersion, version)
			assert.Equal(t, tc.wantFlavor, flavor)
		})
	}
}

func TestParseVersionNoMatch(t *testing.T) {
	version, flavor := ParseVersion("command not found")
	assert.Empty(t, version)
	assert.Empty(t, flavor)
}

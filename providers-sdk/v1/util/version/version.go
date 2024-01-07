// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go.mondoo.com/cnquery/v9/utils/stringx"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	mastermind "github.com/Masterminds/semver"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/v9/cli/components"
	"go.mondoo.com/cnquery/v9/logger"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"golang.org/x/mod/modfile"
)

var rootCmd = &cobra.Command{
	Short: "cnquery versioning tool",
	Long: `
cnquery versioning tool allows us to update the version of one or more providers.

The tool will automatically detect the current version of the provider and
suggest a new version. It will also create a commit with the new version and
push it to a new branch.

  $ version update providers/*/ --increment=patch --commit

The tool will also check if the provider go dependencies have changed since the 
last version and will suggest to update them as well. To just clean up the go.mod
and go.sum files, run:

  $ version mod-tidy providers/*/ 

To update all provider go dependencies to the latest patch version, run:

  $ version mod-update providers/*/ --patch 

To update all provider go dependencies to the latest version, run:

  $ version mod-update providers/*/ --latest
`,
}

var updateCmd = &cobra.Command{
	Use:   "update [PROVIDERS]",
	Short: "try to update the version of the provider",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		updateVersions(args)
	},
}

var checkCmd = &cobra.Command{
	Use:   "check [PROVIDERS]",
	Short: "checks if providers need updates",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for i := range args {
			checkUpdate(args[i])
		}
	},
}

var modTidyCmd = &cobra.Command{
	Use:   "mod-tidy [PROVIDERS]",
	Short: "run 'go mod tidy' for all provided providers",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for i := range args {
			goModTidy(args[i])
		}
	},
}

var modUpdateCmd = &cobra.Command{
	Use:   "mod-update [PROVIDERS]",
	Short: "update all go dependencies for all provided providers",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		updateStrategy := UpdateStrategyNone

		if latestPatchVersion {
			updateStrategy = UpdateStrategyPatch
		} else if latestVersion {
			updateStrategy = UpdateStrategyLatest
		}

		ignorePkgs, _ := cmd.Flags().GetStringSlice("ignore-packages")

		for i := range args {
			checkGoModUpdate(args[i], updateStrategy, ignorePkgs)
		}
	},
}

type UpdateStrategy int

const (
	// UpdateStrategyNone indicates that version should not be updated
	UpdateStrategyNone UpdateStrategy = iota
	// UpdateStrategyLatest indicates that version should be updated to the latest
	UpdateStrategyLatest
	// UpdateStrategyPatch indicates that version should be updated to the latest patch
	UpdateStrategyPatch
)

func checkGoModUpdate(providerPath string, updateStrategy UpdateStrategy, ignorePkgs []string) {
	log.Info().Msgf("Updating dependencies for %s...", providerPath)

	// Define the path to your project's go.mod file
	goModPath := filepath.Join(providerPath, "go.mod")

	// Read the content of the go.mod file
	modContent, err := os.ReadFile(goModPath)
	if err != nil {
		log.Info().Msgf("Error reading go.mod file: %v", err)
		return
	}

	// Parse the go.mod file
	modFile, err := modfile.Parse("go.mod", modContent, nil)
	if err != nil {
		log.Info().Msgf("Error parsing go.mod file: %v", err)
		return
	}

	// Iterate through the require statements and update dependencies
	for _, require := range modFile.Require {
		// Skip indirect dependencies
		if require.Indirect {
			continue
		}

		var modPath string
		switch updateStrategy {
		case UpdateStrategyLatest:
			modPath = require.Mod.Path + "@latest"
		case UpdateStrategyPatch:
			modPath = require.Mod.Path + "@patch" // see https://github.com/golang/go/issues/26812
		default:
			modPath = require.Mod.Path + "@" + require.Mod.Version
		}

		if require.Syntax.Comments.Before != nil {
			for i := range require.Syntax.Comments.Before {
				comment := require.Syntax.Comments.Before[i].Token
				if strings.HasPrefix(comment, "// pin") {
					version := strings.TrimSpace(strings.TrimPrefix(comment, "// pin"))
					log.Info().Msgf("Found pin comment for %s: %s", require.Mod.Path, version)
					modPath = require.Mod.Path + "@" + version
				}
			}
		}

		cmd := exec.Command("go", "get", "-u", modPath)

		// Redirect standard output and standard error to the console
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Set the working directory for the command
		cmd.Dir = providerPath

		if stringx.Contains(ignorePkgs, require.Mod.Path) {
			log.Info().Msgf("Ignoring %s", require.Mod.Path)
			continue
		}

		log.Info().Msgf("Updating %s to the latest version...", require.Mod.Path)

		// Run the `go get` command to update the dependency
		err := cmd.Run()
		if err != nil {
			log.Info().Msgf("Error updating %s: %v", require.Mod.Path, err)
		}
	}

	// Re-read the content of the go.mod file after updating
	modContent, err = os.ReadFile(goModPath)
	if err != nil {
		fmt.Printf("Error reading go.mod file: %v\n", err)
		return
	}

	// Parse the go.mod file again with the updated content
	modFile, err = modfile.Parse("go.mod", modContent, nil)
	if err != nil {
		fmt.Printf("Error parsing go.mod file: %v\n", err)
		return
	}

	// Write the updated go.mod file
	updatedModContent, err := modFile.Format()
	if err != nil {
		log.Info().Msgf("Error formatting go.mod file: %v", err)
		return
	}

	err = os.WriteFile(goModPath, updatedModContent, 0o644)
	if err != nil {
		log.Info().Msgf("Error writing updated go.mod file: %v", err)
		return
	}

	log.Info().Msgf("All dependencies updated.")

	// Run 'go mod tidy' to clean up the go.mod and go.sum files
	goModTidy(providerPath)

	log.Info().Msgf("All dependencies updated and cleaned up successfully.")
}

func goModTidy(providerPath string) {
	log.Info().Msgf("Running 'go mod tidy' for %s...", providerPath)

	// Run 'go mod tidy' to clean up the go.mod and go.sum files
	tidyCmd := exec.Command("go", "mod", "tidy")

	// Redirect standard output and standard error
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr

	// Set the working directory for the command
	tidyCmd.Dir = providerPath

	err := tidyCmd.Run()
	if err != nil {
		log.Error().Msgf("Error running 'go mod tidy': %v", err)
		return
	}
}

var defaultsCmd = &cobra.Command{
	Use:   "defaults [PROVIDERS]",
	Short: "generates the content for the defaults list of providers",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		defaults := parseDefaults(args)
		fmt.Println(defaults)
	},
}

func checkUpdate(providerPath string) {
	conf, err := getConfig(providerPath)
	if err != nil {
		log.Error().Err(err).Str("path", providerPath).Msg("failed to process version")
		return
	}

	changes := countChangesSince(conf, providerPath)
	logChanges(changes, conf)
}

func logChanges(changes int, conf *providerConf) {
	if changes == 0 {
		log.Info().Str("version", conf.version).Str("provider", conf.name).Msg("no changes")
	} else if fastMode {
		log.Info().Str("version", conf.version).Str("provider", conf.name).Msg("provider changed")
	} else {
		log.Info().Int("changes", changes).Str("version", conf.version).Str("provider", conf.name).Msg("provider changed")
	}
}

var (
	reVersion = regexp.MustCompile(`Version:\s*"([^"]+)"`)
	reName    = regexp.MustCompile(`Name:\s*"([^"]+)",`)
)

const (
	titlePrefix = "🎉 "
)

type providerConf struct {
	path    string
	content string
	version string
	name    string
}

func (conf *providerConf) title() string {
	return conf.name + "-" + conf.version
}

func (conf *providerConf) commitTitle() string {
	return titlePrefix + conf.title()
}

type updateConfs []*providerConf

func (confs updateConfs) titles() []string {
	titles := make([]string, len(confs))
	for i := range confs {
		titles[i] = confs[i].title()
	}
	return titles
}

func (confs updateConfs) commitTitle() string {
	return "🎉 " + strings.Join(confs.titles(), ", ")
}

func (confs updateConfs) branchName() string {
	if len(confs) <= 5 {
		return "version/" + strings.Join(confs.titles(), "+")
	}

	now := time.Now()
	return "versions/" + strconv.Itoa(len(confs)) + "-provider-updates-" + now.Format(time.DateOnly)
}

func getVersion(content string) string {
	m := reVersion.FindStringSubmatch(content)
	if len(m) == 0 {
		return ""
	}
	return m[1]
}

func getConfig(providerPath string) (*providerConf, error) {
	var conf providerConf

	conf.path = filepath.Join(providerPath, "config/config.go")
	raw, err := os.ReadFile(conf.path)
	if err != nil {
		return nil, errors.New("failed to read provider config file")
	}
	conf.content = string(raw)

	// Note: name and version must come first in the config, since
	// we only regex-match, instead of reading the structure properly
	m := reName.FindStringSubmatch(conf.content)
	if len(m) == 0 {
		return nil, errors.New("no provider name found in config")
	}
	conf.name = m[1]

	conf.version = getVersion(conf.content)
	if conf.version == "" {
		return nil, errors.New("no provider version found in config")
	}
	return &conf, nil
}

func updateVersions(providerPaths []string) {
	updated := []*providerConf{}

	for _, path := range providerPaths {
		conf, err := tryUpdate(path)
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("failed to process version")
			continue
		}
		if conf == nil {
			log.Info().Str("path", path).Msg("nothing to update")
			continue
		}
		updated = append(updated, conf)
	}

	if doCommit {
		if err := commitChanges(updated); err != nil {
			log.Error().Err(err).Msg("failed to commit changes")
		}
	}
}

func tryUpdate(providerPath string) (*providerConf, error) {
	conf, err := getConfig(providerPath)
	if err != nil {
		return nil, err
	}

	changes := countChangesSince(conf, providerPath)
	logChanges(changes, conf)

	if changes == 0 {
		return nil, nil
	}

	version, err := bumpVersion(conf.version)
	if err != nil || version == "" {
		return nil, err
	}

	res := reVersion.ReplaceAllStringFunc(conf.content, func(v string) string {
		return "Version: \"" + version + "\""
	})

	raw, err := format.Source([]byte(res))
	if err != nil {
		return nil, err
	}

	// no switching config to the new version => gets new commitTitle + branchName!
	log.Info().Str("provider", conf.name).Str("version", version).Str("previous", conf.version).Msg("set new version")
	conf.version = version

	if err = os.WriteFile(conf.path, raw, 0o644); err != nil {
		log.Fatal().Err(err).Str("path", conf.path).Msg("failed to write file")
	}
	log.Info().Str("path", conf.path).Msg("updated config")

	if !doCommit {
		log.Info().Msg("git add " + conf.path + " && git commit -m \"" + conf.commitTitle() + "\"")
	}

	return conf, nil
}

func bumpVersion(version string) (string, error) {
	v, err := mastermind.NewVersion(version)
	if err != nil {
		return "", errors.New("version '" + version + "' is not a semver")
	}

	patch := v.IncPatch()
	minor := v.IncMinor()
	major := v.IncMajor()

	if increment == "patch" {
		return (&patch).String(), nil
	}
	if increment == "minor" {
		return (&minor).String(), nil
	}
	if increment == "major" {
		return (&major).String(), nil
	}
	if increment != "" {
		return "", errors.New("do not understand --increment=" + increment + ", either pick patch, minor, or major")
	}

	versions := []string{
		v.String() + " - no change, keep developing",
		(&patch).String(),
		(&minor).String(),
	}

	selection := -1
	model := components.NewListModel("Select version", versions, func(s int) {
		selection = s
	})
	_, err = tea.NewProgram(model, tea.WithInputTTY()).Run()
	if err != nil {
		return "", err
	}

	if selection == -1 || selection == 0 {
		return "", nil
	}

	return versions[selection], nil
}

func commitChanges(confs updateConfs) error {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return errors.New("failed to open git: " + err.Error())
	}

	headRef, err := repo.Head()
	if err != nil {
		return errors.New("failed to get git head: " + err.Error())
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return errors.New("failed to get git tree: " + err.Error())
	}

	branchName := confs.branchName()
	branchRef := plumbing.NewBranchReferenceName(branchName)

	// Note: The branch may be local and thus won't be found in repo.Branch(branchName)
	// This is consufing and I couldn't find any further docs on this behavior,
	// but we have to work around it.
	if _, err := repo.Reference(branchRef, true); err == nil {
		err = repo.Storer.RemoveReference(branchRef)
		if err != nil {
			return errors.New("failed to git delete branch " + branchName + ": " + err.Error())
		}
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Hash:   headRef.Hash(),
		Branch: branchRef,
		Create: true,
		Keep:   true,
	})
	if err != nil {
		return errors.New("failed to git checkout+create " + branchName + ": " + err.Error())
	}

	fmt.Print("Adding providers to commit ")
	for i := range confs {
		_, err = worktree.Add(confs[i].path)
		if err != nil {
			return errors.New("failed to git add: " + err.Error())
		}
		fmt.Print(".")
	}
	fmt.Println(" done")

	body := "\n\nThis release was created by cnquery's provider versioning bot.\n\n" +
		"You can find me under: `providers-sdk/v1/util/version`.\n"

	commit, err := worktree.Commit(confs.commitTitle()+body, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Mondoo",
			Email: "hello@mondoo.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return errors.New("failed to commit: " + err.Error())
	}

	_, err = repo.CommitObject(commit)
	if err != nil {
		return errors.New("commit is not in repo: " + err.Error())
	}

	// Getting the GPG key is a hassle, so we use CLI for now...
	err = exec.Command("git", "commit", "--amend", "--no-edit", "-S").Run()
	if err != nil {
		return err
	}

	log.Info().Msg("committed changes for " + strings.Join(confs.titles(), ", "))
	log.Info().Msg("running: git push -u origin " + branchName)

	// Not sure why the auth method doesn't work... so we exec here
	err = exec.Command("git", "push", "-u", "origin", branchName).Run()
	if err != nil {
		return err
	}

	log.Info().Msg("updates pushed successfully, open: \n\t" +
		"https://github.com/mondoohq/cnquery/compare/" + branchName + "?expand=1")
	return nil
}

func titleOf(msg string) string {
	i := strings.Index(msg, "\n")
	if i != -1 {
		return msg[0:i]
	}
	return msg
}

func countChangesSince(conf *providerConf, repoPath string) int {
	repo, err := git.PlainOpen(".")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open git repo")
	}
	iter, err := repo.Log(&git.LogOptions{
		PathFilter: func(p string) bool {
			return strings.HasPrefix(p, repoPath)
		},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to iterate git history")
	}

	if !fastMode {
		fmt.Print("crawling git history...")
	}

	var found *object.Commit
	var count int
	for c, err := iter.Next(); err == nil; c, err = iter.Next() {
		if !fastMode {
			fmt.Print(".")
		}

		if strings.HasPrefix(c.Message, titlePrefix) && strings.Contains(titleOf(c.Message), " "+conf.title()) {
			found = c
			break
		}

		count++
		if fastMode {
			return count
		}
	}
	if !fastMode {
		fmt.Println()
	}

	if found == nil {
		log.Warn().Msg("looks like there is no previous version in your commit history => we assume this is the first version commit")
	}
	return count
}

func parseDefaults(paths []string) string {
	confs := []*plugin.Provider{}
	for _, path := range paths {
		name := filepath.Base(path)
		data, err := os.ReadFile(filepath.Join(path, "dist", name+".json"))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to read config json")
		}
		var v plugin.Provider
		if err = json.Unmarshal(data, &v); err != nil {
			log.Fatal().Err(err).Msg("failed to parse config json")
		}
		confs = append(confs, &v)
	}

	var res strings.Builder
	for i := range confs {
		conf := confs[i]
		var connectors strings.Builder
		for j := range conf.Connectors {
			conn := conf.Connectors[j]
			connectors.WriteString(fmt.Sprintf(`
				{
					Name:  %#v,
					Short: %#v,
				},`, conn.Name, conn.Short))
		}

		res.WriteString(fmt.Sprintf(`
	"%s": {
		Provider: &plugin.Provider{
			Name: "%s",
			ConnectionTypes: %#v,
			Connectors: []plugin.Connector{%s
			},
		},
	},`, conf.Name, conf.Name, conf.ConnectionTypes, connectors.String()))
	}

	return res.String()
}

var (
	fastMode           bool
	doCommit           bool
	increment          string
	latestVersion      bool
	latestPatchVersion bool
)

func init() {
	rootCmd.PersistentFlags().BoolVar(&fastMode, "fast", false, "perform fast checking of git repo (not counting changes)")
	rootCmd.PersistentFlags().BoolVar(&doCommit, "commit", false, "commit the change to git if there is a version bump")
	rootCmd.PersistentFlags().StringVar(&increment, "increment", "", "automatically bump either patch, minor, or major version")

	modUpdateCmd.PersistentFlags().BoolVar(&latestVersion, "latest", false, "update versions to latest")
	modUpdateCmd.PersistentFlags().BoolVar(&latestPatchVersion, "patch", false, "update versions to latest patch")
	modUpdateCmd.PersistentFlags().StringSlice("ignore-packages", []string{}, "ignore go package(s) from update")
	rootCmd.AddCommand(updateCmd, checkCmd, modUpdateCmd, modTidyCmd, defaultsCmd)
}

func main() {
	logger.CliCompactLogger(logger.LogOutputWriter)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	"go.mondoo.com/cnquery/cli/components"
	"go.mondoo.com/cnquery/logger"
)

var rootCmd = &cobra.Command{}

var updateCmd = &cobra.Command{
	Use:   "update [PROVIDERS]",
	Short: "try to update the version of the provider",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for i := range args {
			updateVersion(args[i])
		}
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

func checkUpdate(providerPath string) {
	conf, err := getConfig(providerPath)
	if err != nil {
		log.Error().Err(err).Str("path", providerPath).Msg("failed to process version")
		return
	}

	commitTitle := conf.name + "-" + conf.version
	changes := countChangesSince(commitTitle, providerPath, conf.path)
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

type providerConf struct {
	path    string
	content string
	version string
	name    string
}

func (p providerConf) commitTitle() string {
	return "🎉 " + p.name + "-" + p.version
}

func (p providerConf) branchName() string {
	return "version/" + p.name + "-" + p.version
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

	m = reVersion.FindStringSubmatch(conf.content)
	if len(m) == 0 {
		return nil, errors.New("no provider version found in config")
	}

	conf.version = m[1]
	return &conf, nil
}

func updateVersion(providerPath string) {
	conf, err := getConfig(providerPath)
	if err != nil {
		log.Error().Err(err).Str("path", providerPath).Msg("failed to process version")
		return
	}

	didUpdate, err := tryUpdate(providerPath, conf)
	if err != nil {
		log.Fatal().Err(err).Str("path", providerPath).Msg("failed to process version")
	}
	if !didUpdate {
		log.Info().Msg("nothing to do, bye")
		return
	}
}

func tryUpdate(repoPath string, conf *providerConf) (bool, error) {
	changes := countChangesSince(conf.commitTitle(), repoPath, conf.path)
	logChanges(changes, conf)

	if changes == 0 {
		return false, nil
	}

	version, err := bumpVersion(conf.version)
	if err != nil || version == "" {
		return false, err
	}

	res := reVersion.ReplaceAllStringFunc(conf.content, func(v string) string {
		return "Version: \"" + version + "\""
	})

	// no switching config to the new version => gets new commitTitle + branchName!
	log.Info().Str("provider", conf.name).Str("version", version).Str("previous", conf.version).Msg("set new version")
	conf.version = version

	if err = os.WriteFile(conf.path, []byte(res), 0o644); err != nil {
		log.Fatal().Err(err).Str("path", conf.path).Msg("failed to write file")
	}
	log.Info().Str("path", conf.path).Msg("updated config")

	if doCommit {
		if err = commitChanges(conf); err != nil {
			log.Error().Err(err).Msg("failed to commit changes")
		}
	} else {
		log.Info().Msg("git add " + conf.path + " && git commit -m \"" + conf.commitTitle() + "\"")
	}

	return true, nil
}

func bumpVersion(version string) (string, error) {
	v, err := mastermind.NewVersion(version)
	if err != nil {
		return "", errors.New("version '" + version + "' is not a semver")
	}

	patch := v.IncPatch()
	minor := v.IncMinor()
	// TODO: check if the major version of the repo has changed and bump it

	if increment == "patch" {
		return (&patch).String(), nil
	}
	if increment == "minor" {
		return (&patch).String(), nil
	}
	if increment != "" {
		return "", errors.New("do not understand --increment=" + increment + ", either pick patch or minor")
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

func commitChanges(conf *providerConf) error {
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

	branchName := conf.branchName()
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

	_, err = worktree.Add(conf.path)
	if err != nil {
		return errors.New("failed to git add: " + err.Error())
	}

	commit, err := worktree.Commit(conf.commitTitle(), &git.CommitOptions{
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

	log.Info().Msg("comitted changes for " + conf.name + " " + conf.version)
	log.Info().Msg("run: git push -u origin " + branchName)
	return nil
}

func countChangesSince(commitTitle string, repoPath string, confPath string) int {
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

		if strings.HasPrefix(c.Message, commitTitle) {
			found = c
			break
		}

		count++
		if fastMode {
			return count
		}
	}
	fmt.Println()

	if found == nil {
		log.Warn().Msg("looks like there is no previous version in your commit history => we assume this is the first version commit")
	}
	return count
}

var (
	fastMode  bool
	doCommit  bool
	increment string
)

func init() {
	rootCmd.PersistentFlags().BoolVar(&fastMode, "fast", false, "perform fast checking of git repo (not counting changes)")
	rootCmd.PersistentFlags().BoolVar(&doCommit, "commit", false, "commit the change to git if there is a version bump")
	rootCmd.PersistentFlags().StringVar(&increment, "increment", "", "automatically bump either patch or minor version")

	rootCmd.AddCommand(updateCmd, checkCmd)
}

func main() {
	logger.CliCompactLogger(logger.LogOutputWriter)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

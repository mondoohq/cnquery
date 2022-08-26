package sshd

import (
	"bufio"
	"fmt"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/afero"

	"go.mondoo.com/cnquery/motor/providers/os"
)

var (
	// includeStatement is a regexp for checking whether a given sshd configuratoin line
	// is an 'Include' statement
	includeStatement = regexp.MustCompile(`^Include\s+(.*)$`)
	// includeStatementHasGlob is a regext for checking whether the contents of an 'Include'
	// statement have a wildcard/glob (ie. a literal '*')
	includeStatementHasGlob = regexp.MustCompile(`.*\*.*`)
)

// GetAllSshdIncludedFiles will return the list of dependent files referenced in the sshd
// configuration file's 'Include' statements starting from the provided filePath parameter as
// the beginning of the sshd configuration.
func GetAllSshdIncludedFiles(filePath string, osProvider os.OperatingSystemProvider) ([]string, error) {
	allFiles, _, err := readSshdConfig(filePath, osProvider)
	return allFiles, err
}

// GetSshdUnifiedContent will return the unified sshd configuration content starting
// from the provided filePath parameter as the beginning of the sshd configuration.
func GetSshdUnifiedContent(filePath string, osProvider os.OperatingSystemProvider) (string, error) {
	_, content, err := readSshdConfig(filePath, osProvider)
	return content, err
}

// When an Include lists a relative path, it is interpreted as relative to /etc/ssh/
const relativePathPrefix = "/etc/ssh/"

func getBaseDirectory(filePath string) string {
	baseDirectoryPath := filepath.Dir(filePath)
	// insert the /etc/ssh path prefix if a relative path is specified
	if baseDirectoryPath == "." {
		baseDirectoryPath = relativePathPrefix
	}
	if !strings.HasPrefix(baseDirectoryPath, "/") {
		baseDirectoryPath = relativePathPrefix + baseDirectoryPath
	}

	return baseDirectoryPath
}

func getFullPath(filePath string) string {
	dir := getBaseDirectory(filePath)
	fileName := filepath.Base(filePath)
	return filepath.Join(dir, fileName)
}

// readSshdConfig will traverse the provided path to an sshd config file and return
// the list of all depended files encountered while recursively traversing the
// sshd 'Include' statements, and the unified sshd configuration where all the
// sshd 'Include' statments have been replaced with the referenced file's content
// in place of the 'Include'.
func readSshdConfig(filePath string, osProvider os.OperatingSystemProvider) ([]string, string, error) {
	allFiles := []string{}
	var allContent strings.Builder

	baseDirectoryPath := getBaseDirectory(filePath)

	// First check if the Include path has a wildcard/glob
	m := includeStatementHasGlob.FindStringSubmatch(filePath)
	if m != nil {
		glob := filepath.Base(filePath)

		// List all the files in lexical order and check whether any match the glob
		afs := &afero.Afero{Fs: osProvider.FS()}

		wErr := afs.Walk(baseDirectoryPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// don't recurse down further directories (as that matches sshd behavior)
			if info.IsDir() {
				return nil
			}
			match, err := filepath.Match(glob, info.Name())
			if err != nil {
				return err
			}
			if !match {
				return nil
			}

			fullFilepath := filepath.Join(baseDirectoryPath, info.Name())

			// Now search through that file for any more Include statements
			files, content, err := readSshdConfig(fullFilepath, osProvider)
			if err != nil {
				return err
			}
			allFiles = append(allFiles, files...)
			if _, err := allContent.WriteString(content); err != nil {
				return err
			}
			return nil
		})
		if wErr != nil {
			return nil, "", fmt.Errorf("error while walking through sshd config directory: %s", wErr)
		}

		return allFiles, allContent.String(), nil
	}

	// Now see if we're dealing with a directory
	fullFilePath := getFullPath(filePath)
	f, err := osProvider.FS().Open(fullFilePath)
	if err != nil {
		return nil, "", err
	}

	fileInfo, err := f.Stat()
	if err != nil {
		return nil, "", err
	}
	if fileInfo.IsDir() {
		// Again list all files in lexical order
		afs := &afero.Afero{Fs: osProvider.FS()}

		wErr := afs.Walk(fullFilePath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}
			allFiles = append(allFiles, path)

			// Now check this very file for any 'Include' statements
			files, content, err := readSshdConfig(path, osProvider)
			if err != nil {
				return err
			}
			allFiles = append(allFiles, files...)
			if _, err := allContent.WriteString(content); err != nil {
				return err
			}

			return nil
		})
		if wErr != nil {
			return nil, "", fmt.Errorf("error while walking through sshd config directory: %s", wErr)
		}

		return allFiles, allContent.String(), nil
	}

	// If here, we must be dealing with neither a wildcard nor directory
	// so just consume the file's contents
	allFiles = append(allFiles, fullFilePath)

	rawFile, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, "", err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(rawFile)))
	for scanner.Scan() {
		line := scanner.Text()
		m := includeStatement.FindStringSubmatch(line)
		if m != nil {
			includeList := strings.Split(m[1], " ") // TODO: what about files with actual spaces in their names?
			for _, file := range includeList {
				files, content, err := readSshdConfig(file, osProvider)
				if err != nil {
					return nil, "", err
				}
				allFiles = append(allFiles, files...)
				if _, err := allContent.WriteString(content); err != nil {
					return nil, "", err
				}
			}
			continue
		}

		if _, err := allContent.WriteString(line + "\n"); err != nil {
			return nil, "", err
		}
	}
	return allFiles, allContent.String(), nil
}

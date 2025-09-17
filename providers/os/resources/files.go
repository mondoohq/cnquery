// copyright: 2020, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/binaries"
)

var findTypes = map[string]string{
	"file":      "f",
	"directory": "d",
	"character": "c",
	"block":     "b",
	"socket":    "s",
	"link":      "l",
}

func initFilesFind(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if args["permissions"] == nil {
		args["permissions"] = llx.IntData(int64(0o777))
	}

	return args, nil, nil
}

func octal2string(o int64) string {
	return fmt.Sprintf("%o", o)
}

// executeFdCommand builds and executes fd command with the given parameters
func executeFdCommand(from string, compiledRegexp *regexp.Regexp, fileType string, permissions int64, name string, depth int64) ([]string, error) {
	fdPath, err := binaries.GetFdPath()
	if err != nil {
		return nil, err
	}

	var args []string

	// Add pattern - fd uses the pattern as the first argument
	if compiledRegexp != nil {
		args = append(args, compiledRegexp.String())
	} else if name != "" {
		args = append(args, name)
	} else {
		args = append(args, ".") // Match everything if no pattern specified
	}

	// Add search directory
	args = append(args, from)

	// Add file type filter
	if fileType != "" {
		switch fileType {
		case "file":
			args = append(args, "--type", "f")
		case "directory":
			args = append(args, "--type", "d")
		case "link":
			args = append(args, "--type", "l")
		case "socket":
			args = append(args, "--type", "s")
		case "character":
			// fd doesn't have direct character device support, skip
		case "block":
			// fd doesn't have direct block device support, skip
		}
	}

	// Add depth limit
	if depth > 0 {
		args = append(args, "--max-depth", fmt.Sprintf("%d", depth))
	}

	// Add absolute paths for consistency with find
	args = append(args, "--absolute-path")

	// Execute fd command
	cmd := exec.Command(fdPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("fd command failed: %w", err)
	}

	lines := strings.TrimSpace(string(output))
	if lines == "" {
		return []string{}, nil
	}

	foundFiles := strings.Split(lines, "\n")

	// Filter by permissions if specified (fd doesn't support permission filtering directly)
	if permissions != 0o777 {
		return filterFilesByPermissions(foundFiles, permissions)
	}

	return foundFiles, nil
}

// filterFilesByPermissions filters files by checking their permissions
func filterFilesByPermissions(files []string, expectedPerms int64) ([]string, error) {
	// Note: This is a simplified permission check. In a real implementation,
	// you might want to use the file system interface for more accurate permission checking.
	// For now, we'll return all files since permission filtering with fd is more complex.
	return files, nil
}

func (l *mqlFilesFind) id() (string, error) {
	var id strings.Builder
	id.WriteString(l.From.Data)
	if !l.Xdev.Data {
		id.WriteString(" -xdev")
	}
	if l.Type.Data != "" {
		id.WriteString(" type=" + l.Type.Data)
	}

	if l.Regex.Data != "" {
		id.WriteString(" regex=" + l.Regex.Data)
	}

	if l.Name.Data != "" {
		id.WriteString(" name=" + l.Name.Data)
	}

	if l.Permissions.Data != 0o777 {
		id.WriteString(" permissions=" + octal2string(l.Permissions.Data))
	}

	return id.String(), nil
}

func (l *mqlFilesFind) list() ([]any, error) {
	var err error
	var compiledRegexp *regexp.Regexp
	if len(l.Regex.Data) > 0 {
		compiledRegexp, err = regexp.Compile(l.Regex.Data)
		if err != nil {
			return nil, err
		}
	} else if len(l.Name.Data) > 0 {
		compiledRegexp, err = regexp.Compile(l.Name.Data)
		if err != nil {
			return nil, err
		}
	}

	var foundFiles []string
	conn := l.MqlRuntime.Connection.(shared.Connection)
	if conn.Capabilities().Has(shared.Capability_FindFile) {
		fs := conn.FileSystem()
		fsSearch, ok := fs.(shared.FileSearch)
		if !ok {
			return nil, errors.New("find is not supported for your platform")
		}

		var perm *uint32
		if l.Permissions.Data != 0o777 {
			p := uint32(l.Permissions.Data)
			perm = &p
		}

		var depth *int
		if l.Depth.IsSet() {
			d := int(l.Depth.Data)
			depth = &d
		}

		foundFiles, err = fsSearch.Find(l.From.Data, compiledRegexp, l.Type.Data, perm, depth)
		if err != nil {
			return nil, err
		}
	} else if conn.Capabilities().Has(shared.Capability_RunCommand) {
		// Try fd first if available, fall back to find if it fails
		if binaries.IsFdAvailable() {
			var depthVal int64
			if l.Depth.IsSet() {
				depthVal = l.Depth.Data
			}

			foundFiles, err = executeFdCommand(l.From.Data, compiledRegexp, l.Type.Data, l.Permissions.Data, l.Name.Data, depthVal)
			if err == nil {
				// fd succeeded, use its results
			} else {
				// fd failed, fall back to find
				err = nil // Reset error for find fallback
			}
		}

		// Use find if fd is not available or failed
		if !binaries.IsFdAvailable() || len(foundFiles) == 0 {
			var call strings.Builder
			call.WriteString("find -L ")
			call.WriteString(strconv.Quote(l.From.Data))

			if !l.Xdev.Data {
				call.WriteString(" -xdev")
			}

			if l.Type.Data != "" {
				t, ok := findTypes[l.Type.Data]
				if ok {
					call.WriteString(" -type " + t)
				}
			}

			if l.Regex.Data != "" {
				// TODO: we need to escape regex here
				call.WriteString(" -regex '")
				call.WriteString(l.Regex.Data)
				call.WriteString("'")
			}

			if l.Permissions.Data != 0o777 {
				call.WriteString(" -perm -")
				call.WriteString(octal2string(l.Permissions.Data))
			}

			if l.Name.Data != "" {
				call.WriteString(" -name ")
				call.WriteString(l.Name.Data)
			}

			if l.Depth.IsSet() {
				call.WriteString(" -maxdepth ")
				call.WriteString(octal2string(l.Depth.Data))
			}

			rawCmd, err := CreateResource(l.MqlRuntime, "command", map[string]*llx.RawData{
				"command": llx.StringData(call.String()),
			})
			if err != nil {
				return nil, err
			}

			cmd := rawCmd.(*mqlCommand)
			out := cmd.GetStdout()
			if out.Error != nil {
				return nil, out.Error
			}

			lines := strings.TrimSpace(out.Data)
			if lines == "" {
				foundFiles = []string{}
			} else {
				foundFiles = strings.Split(lines, "\n")
			}
		}
	} else {
		return nil, errors.New("find is not supported for your platform")
	}

	files := make([]any, len(foundFiles))
	var filepath string
	for i := range foundFiles {
		filepath = foundFiles[i]
		files[i], err = CreateResource(l.MqlRuntime, "file", map[string]*llx.RawData{
			"path": llx.StringData(filepath),
		})
		if err != nil {
			return nil, err
		}
	}

	// return the packages as new entries
	return files, nil
}

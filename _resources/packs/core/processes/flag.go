package processes

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kballard/go-shellquote"
)

// Flagset is derived from Go's internal flagset, licensed MIT
type FlagSet struct {
	name   string
	parsed bool
	actual map[string]string
	args   []string // arguments after flags
}

func (f *FlagSet) ParseCommand(cmd string) error {
	if cmd == "" {
		return errors.New("no command provided")
	}
	words, err := shellquote.Split(cmd)
	if err != nil {
		return err
	}
	args := words[1:]

	// NOTE: it is impossible to do flag parsing correct without having the context of the binary
	// - `--name=x` is pretty clear where name is the key and x is the value
	//  `--name x` here name is the key, but x could be the value or an arg, if name is a boolean flag x will not be the value
	//
	// Therefore we work with the assumption that if `--arg` is followed by a flag without `-` prefix, we count this as a value
	preparedArgs := []string{}
	n := len(args)
	for i := 0; i < n; i++ {
		key := args[i]
		if strings.HasPrefix(key, "-") {
			if i+1 < n && !strings.HasPrefix(args[i+1], "-") {
				preparedArgs = append(preparedArgs, key+"="+args[i+1])
				i++
				continue
			}
		}
		preparedArgs = append(preparedArgs, key)
	}

	return f.Parse(preparedArgs)
}

func (f *FlagSet) Parse(args []string) error {
	f.parsed = true
	f.args = args

	if f.actual == nil {
		f.actual = make(map[string]string)
	}

	for {
		seen, err := f.parseOneArg()
		if seen {
			continue
		}
		if err == nil {
			break
		}
		return err
	}
	return nil
}

func (f *FlagSet) parseOneArg() (bool, error) {
	if len(f.args) == 0 {
		return false, nil
	}
	s := f.args[0]

	if len(s) < 2 || s[0] != '-' {
		f.args = f.args[1:]
		f.actual[s] = ""
		return true, nil
	}
	numMinuses := 1
	if s[1] == '-' {
		numMinuses++
		if len(s) == 2 { // "--" terminates the flags
			f.args = f.args[1:]
			return false, nil
		}
	}
	name := s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return false, fmt.Errorf("bad flag syntax: %s", s)
	}

	// it's a flag. does it have an argument?
	f.args = f.args[1:]
	value := ""
	for i := 1; i < len(name); i++ { // equals cannot be first
		if name[i] == '=' {
			value = name[i+1:]
			name = name[0:i]
			break
		}
	}
	name = strings.ToLower(name)
	f.actual[name] = value
	return true, nil
}

func (f *FlagSet) Map() map[string]string {
	return f.actual
}

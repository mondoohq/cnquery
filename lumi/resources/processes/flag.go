package processes

import (
	"fmt"
	"strings"

	"github.com/kballard/go-shellquote"
	"github.com/rs/zerolog/log"
)

// Flagset is derived from Go's internal flagset, licensed MIT
type FlagSet struct {
	name   string
	parsed bool
	actual map[string]string
	args   []string // arguments after flags
}

func (f *FlagSet) ParseCommand(cmd string) error {
	args, err := shellquote.Split(cmd)
	if err != nil {
		return err
	}
	return f.Parse(args[1:])
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

	// TODO: we need to handle shorthand flags
	if len(s) < 2 || s[0] != '-' {
		log.Debug().Msg("nope")
		return false, nil
	}
	numMinuses := 1
	if s[1] == '-' {
		numMinuses++
		if len(s) == 2 { // "--" terminates the flags
			f.args = f.args[1:]
			log.Debug().Msg("nope2")
			return false, nil
		}
	}
	name := s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		log.Debug().Msg("err")
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

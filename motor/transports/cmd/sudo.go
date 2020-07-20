package cmd

import (
	"fmt"
	"strings"
)

type Wrapper interface {
	Build(cmd string) string
}

func NewSudo() *Sudo {
	return &Sudo{
		user:       "",
		executable: "sudo",
		shell:      "",
	}
}

type Sudo struct {
	user       string
	executable string
	shell      string
}

func (sudo *Sudo) Build(cmd string) string {
	var sb strings.Builder

	sb.WriteString(sudo.executable)

	if len(sudo.user) > 0 {
		user := fmt.Sprintf(" -u %s", sudo.user)
		sb.WriteString(user)
	}

	if len(sudo.shell) > 0 {
		cmd = fmt.Sprintf(" %s -c %s", sudo.shell, cmd)
		sb.WriteString(cmd)
	} else {
		sb.WriteString(" ")
		sb.WriteString(cmd)
	}

	return sb.String()
}

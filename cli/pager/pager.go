package pager

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mattn/go-isatty"
)

func Display(content string, pagerCmdArg string) error {
	var pagerCmd string
	if pagerCmdArg != "" {
		pagerCmd = pagerCmdArg
	} else {
		pagerCmd = defaultPagerCommand()
	}

	pa := strings.Split(pagerCmd, " ")
	c := exec.Command(pa[0], pa[1:]...)
	c.Stdin = strings.NewReader(content)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func defaultPagerCommand() string {
	pagerCmd := os.Getenv("PAGER")
	if pagerCmd == "" {
		pagerCmd = "less -R"
	}
	return pagerCmd
}

func Supported(pagerCmdArg string) bool {
	if pagerCmdArg == "" {
		pagerCmdArg = defaultPagerCommand()
	}
	pa := strings.Split(pagerCmdArg, " ")
	p, err := exec.LookPath(pa[0])
	return isatty.IsTerminal(os.Stdout.Fd()) && runtime.GOOS != "windows" && err == nil && p != ""
}

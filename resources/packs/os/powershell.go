package os

import (
	"bytes"
	"io/ioutil"

	"go.mondoo.io/mondoo/motor/providers/os"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/resources/packs/os/powershell"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// TODO: consider sharing more code with command resource
func (c *lumiPowershell) id() (string, error) {
	return c.Script()
}

func (c *lumiPowershell) execute() (*os.Command, error) {
	osProvider, err := osProvider(c.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	var executedCmd *os.Command

	cmd, err := c.Script()
	if err != nil {
		return nil, err
	}

	// encode the powershell command
	encodedCmd := powershell.Encode(cmd)

	data, ok := c.Cache.Load(encodedCmd)
	if ok {
		executedCmd, ok := data.Data.(*os.Command)
		if ok {
			return executedCmd, nil
		}
	}

	executedCmd, err = osProvider.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	c.Cache.Store(encodedCmd, &lumi.CacheEntry{Data: executedCmd})
	return executedCmd, nil
}

func convertToUtf8Encoding(out []byte) (string, error) {
	enc, name, _ := charset.DetermineEncoding(out, "")
	log.Trace().Str("encoding", name).Msg("check powershell results charset")
	r := transform.NewReader(bytes.NewReader(out), enc.NewDecoder())
	utf8out, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(utf8out), nil
}

func (c *lumiPowershell) GetStdout() (string, error) {
	executedCmd, err := c.execute()
	if err != nil {
		return "", err
	}

	out, err := ioutil.ReadAll(executedCmd.Stdout)
	if err != nil {
		return "", err
	}

	return convertToUtf8Encoding(out)
}

func (c *lumiPowershell) GetStderr() (string, error) {
	executedCmd, err := c.execute()
	if err != nil {
		return "", err
	}

	outErr, err := ioutil.ReadAll(executedCmd.Stderr)
	if err != nil {
		return "", err
	}

	return convertToUtf8Encoding(outErr)
}

func (c *lumiPowershell) GetExitcode() (int64, error) {
	executedCmd, err := c.execute()
	if err != nil {
		return 1, err
	}
	return int64(executedCmd.ExitStatus), nil
}

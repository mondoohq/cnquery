package sshd

import (
	"io"
	"io/ioutil"
	"regexp"
)

// TODO: this is demo implementationn and it does not capture
// all edge cases like multiple keys etc
func Params(reader io.Reader) (map[string]string, error) {
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile("(?m:^([[:alpha:]]+)\\s+(.*))")
	m := re.FindAllStringSubmatch(string(content), -1)
	res := make(map[string]string)
	for _, mm := range m {
		res[mm[1]] = mm[2]
	}

	return res, nil
}

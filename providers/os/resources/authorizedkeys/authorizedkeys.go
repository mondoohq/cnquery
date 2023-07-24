package authorizedkeys

import (
	"bufio"
	"encoding/base64"
	"io"
	"strings"

	"golang.org/x/crypto/ssh"
)

// most ssh keys include base64 padding, so lets use it too (not default in Go)
var RawStdEncoding = base64.StdEncoding.WithPadding(base64.StdPadding)

type Entry struct {
	Line    int64
	Key     ssh.PublicKey
	Label   string
	Options []string
}

func (e Entry) Base64Key() string {
	return RawStdEncoding.EncodeToString(e.Key.Marshal())
}

func Parse(r io.Reader) ([]Entry, error) {
	res := []Entry{}
	scanner := bufio.NewScanner(r)

	lineNo := int64(1)
	for scanner.Scan() {
		line := scanner.Text()

		in := strings.TrimSpace(line)
		if len(in) == 0 || in[0] == '#' {
			continue
		}

		key, comment, options, _, err := ssh.ParseAuthorizedKey([]byte(line))
		if err != nil {
			return nil, err
		}

		res = append(res, Entry{
			Line:    lineNo,
			Key:     key,
			Label:   comment,
			Options: options,
		})
		lineNo++
	}
	return res, nil
}

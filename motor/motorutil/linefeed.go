package motorutil

import "io"

var (
	// the '\r' rune
	RByte byte = 13
	// the '\n' rune
	NByte byte = 10
)

type lineFeedReader struct {
	r io.Reader
}

// NewLineFeedReader creates a new io.Reader that replaces /r with /n
// see https://github.com/golang/go/issues/7802
func NewLineFeedReader(r io.Reader) io.Reader {
	return &lineFeedReader{r: r}
}

func (r lineFeedReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	for i, b := range p {
		if b == RByte {
			p[i] = NByte
		}
	}
	return
}

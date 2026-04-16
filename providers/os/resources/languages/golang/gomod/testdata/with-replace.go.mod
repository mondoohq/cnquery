module github.com/example/replaced

go 1.21

require (
	github.com/pkg/errors v0.9.1
	github.com/foo/bar/v2 v2.1.0
	golang.org/x/net v0.15.0 // indirect
)

replace github.com/foo/bar/v2 => github.com/foo/bar/v2 v2.2.0

exclude github.com/old/dep v1.0.0

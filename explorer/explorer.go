package explorer

//go:generate protoc --proto_path=../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. cnquery_explorer.proto

const (
	SERVICE_NAME = "explorer.api.mondoo.com"
)

package proto

//go:generate protoc --proto_path=../../:. --go_out=. --go_opt=paths=source_relative  --go-grpc_out=. --go-grpc_opt=paths=source_relative cnquery.proto

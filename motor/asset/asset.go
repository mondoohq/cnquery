package asset

//go:generate protoc --proto_path=$GOPATH/src:. --proto_path=$GOPATH/pkg/mod/github.com/gogo/protobuf@v1.3.1/gogoproto --falcon_out=. --iam-actions_out=. --gofast_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types:$GOPATH/src asset.proto

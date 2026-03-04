1. Run tests

```

export GOPATH=$HOME/go                            
export PATH=$PATH:$GOPATH/bin
protoc --proto_path=proto --go_out=. proto/*.proto

export test_folder=$PWD/../test_data
go test -v -timeout 0 -count 1 -p 1 -ldflags="-X 'github.com/evgeniums/go-utils/test_utils.SqliteFolder=$test_folder'" ./test/...

```

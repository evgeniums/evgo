export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
protoc --proto_path=proto --go_out=. proto/*.proto


Run from top project module path the following. It will build all executables in cmd folder and place them to ../bin folder

```

export GOPATH=$HOME/go                            
export PATH=$PATH:$GOPATH/bin
protoc --proto_path=proto --go_out=. proto/*.proto

# put your label here
export LABEL=whitelabel

$PWD$/go-utils/scripts/build.sh $LABEL

```
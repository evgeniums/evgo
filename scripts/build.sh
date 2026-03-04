#!/bin/bash

# Configuration
LABEL="${1:=whitelabel}"
VERSION="${2:-0.0.1}"
CONFIG_PACKAGE="${3:-$PWD/internal/build_config}"

OUT_DIR=$PWD/../bin
CMD_DIR=$PWD/cmd

echo "Building all executables from $CMD_DIR for label \"$LABEL\" to output $OUT_DIR with config in $CONFIG_PACKAGE..."

# Create output directory
mkdir -p $OUT_DIR

export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
protoc --proto_path=proto --go_out=. proto/*.proto

export app_label=$LABEL
export app_version=$VERSION
export app_git_revision=$(git rev-parse HEAD)
export app_buld_time=$(date +%Y-%m-%dT%H:%M:%S)
export build_config_package=$(go list $CONFIG_PACKAGE)

echo "app_labe=$app_label"
echo "app_version=$app_version"
echo "app_git_revision=$app_git_revision"
echo "app_buld_time=$app_buld_time"
echo "build_config_package=$build_config_package"

# Loop through each directory in cmd/
for dir in ./cmd/*/; do

    name=$(basename "$dir")    
    output_path="$OUT_DIR/${app_label}-${name}"
    
    echo "Building $output_path..."

    LDFLAGS="-X \"$build_config_package.Revision=$app_git_revision\" -X \"$build_config_package.Time=$app_buld_time\" -X \"$build_config_package.Label=$app_label\" -X \"$build_config_package.Version=$app_version\""

    go build -ldflags="$LDFLAGS" -o "$output_path" "$dir"

done
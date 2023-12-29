#!/bin/sh

project_name="ua3f"
release_version="0.2.1"
target=cmd/ua3f.go

release_dir=./bin
rm -rf $release_dir/*
mkdir -p $release_dir

cd $(dirname $0)

gofmt -w ./

for goarch in "amd64" "arm" "arm64" "mipsle" "mips64" "riscv64" "386"; do
    obj_name=$project_name

    GOOS=linux GOARCH=$goarch go build -trimpath -ldflags="-s -w" $target
    tar -zcf $release_dir/$project_name-$release_version-$goarch.tar.gz $obj_name
    rm -f $obj_name
done

GOOS=linux GOARCH="mipsle" GOMIPS=softfloat go build -trimpath -ldflags="-s -w" $target
tar -zcf $release_dir/$project_name-$release_version-mipsle-softfloat.tar.gz $obj_name
rm -f $obj_name
GOOS=linux GOARCH="mipsle" GOMIPS=hardfloat go build -trimpath -ldflags="-s -w" $target
tar -zcf $release_dir/$project_name-$release_version-mipsle-hardfloat.tar.gz $obj_name
rm -f $obj_name

GOOS=linux GOARCH="arm" GOARM=7 go build -trimpath -ldflags="-s -w" $target
tar -zcf $release_dir/$project_name-$release_version-armv7.tar.gz $obj_name
rm -f $obj_name

cp $release_dir/$project_name-$release_version-arm64.tar.gz $release_dir/$project_name-$release_version-armv8.tar.gz

cd $release_dir
for file in ./*; do
    md5 -r $file >>sha1sum.txt
done

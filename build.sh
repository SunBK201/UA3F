#!/bin/sh

project_name="ua3f"
release_version="0.0.2"

release_dir=./release
rm -rf $release_dir/*
mkdir -p $release_dir

cd $(dirname $0)

gofmt -w ./

for goarch in "amd64" "arm" "arm64" "mipsle" "mips64" "riscv64" "386"; do
    obj_name=$project_name

    GOOS=linux GOARCH=$goarch go build -ldflags="-s -w"
    tar -zcf $release_dir/$project_name-$release_version-$goarch.tar.gz $obj_name
    rm -f $obj_name
done

GOOS=linux GOARCH="mipsle" GOMIPS=softfloat go build -ldflags="-s -w"
tar -zcf $release_dir/$project_name-$release_version-mipsle-softfloat.tar.gz $obj_name
rm -f $obj_name
GOOS=linux GOARCH="mipsle" GOMIPS=hardfloat go build -ldflags="-s -w"
tar -zcf $release_dir/$project_name-$release_version-mipsle-hardfloat.tar.gz $obj_name
rm -f $obj_name

cd $release_dir
for file in ./*; do
    md5 -r $file >>sha1sum.txt
done

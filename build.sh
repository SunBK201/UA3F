#!/bin/sh

project_name="ua3f"
release_version="0.5.0"
target=cmd/ua3f.go
dist=./dist
release_dir=./bin

rm -rf $release_dir/*
rm -rf $dist/*
mkdir -p $release_dir
mkdir -p $dist/bin

cd $(dirname $0)

gofmt -w ./

for goarch in "amd64" "arm" "arm64" "mipsle" "mips64" "riscv64" "386"; do
    obj_name=$project_name-$release_version-$goarch

    GOOS=linux GOARCH=$goarch go build -trimpath -ldflags="-s -w" -o $obj_name $target
    cp $obj_name $dist/bin
    mv $obj_name $project_name
    tar -zcf $release_dir/$project_name-$release_version-$goarch.tar.gz $project_name
    rm -f $project_name
done

# mipsle-softfloat
obj_name=$project_name-$release_version-mipsle-softfloat
GOOS=linux GOARCH="mipsle" GOMIPS=softfloat go build -trimpath -ldflags="-s -w" -o $obj_name $target
cp $obj_name $dist/bin
mv $obj_name $project_name
tar -zcf $release_dir/$project_name-$release_version-mipsle-softfloat.tar.gz $project_name
rm -f $project_name

# mipsle-hardfloat
obj_name=$project_name-$release_version-mipsle-hardfloat
GOOS=linux GOARCH="mipsle" GOMIPS=hardfloat go build -trimpath -ldflags="-s -w" -o $obj_name $target
cp $obj_name $dist/bin
mv $obj_name $project_name
tar -zcf $release_dir/$project_name-$release_version-mipsle-hardfloat.tar.gz $project_name
rm -f $project_name

# armv7
obj_name=$project_name-$release_version-armv7
GOOS=linux GOARCH="arm" GOARM=7 go build -trimpath -ldflags="-s -w" -o $obj_name $target
cp $obj_name $dist/bin
mv $obj_name $project_name
tar -zcf $release_dir/$project_name-$release_version-armv7.tar.gz $project_name
rm -f $project_name

# armv8
cp $release_dir/$project_name-$release_version-arm64.tar.gz $release_dir/$project_name-$release_version-armv8.tar.gz
cp $dist/bin/$project_name-$release_version-arm64 $dist/bin/$project_name-$release_version-armv8

cd $release_dir
for file in ./*; do
    md5 -r $file >>sha1sum.txt
done

cd ..
opkg_template=./opkg
ipkg_build=ipkg-build.sh
for goarch in "amd64" "arm" "arm64" "mipsle" "mips64" "riscv64" "386" "mipsle-softfloat" "mipsle-hardfloat" "armv7" "armv8"; do
    obj_name=$project_name-$release_version-$goarch
    mv $dist/bin/$obj_name $opkg_template/usr/bin/ua3f
    sh $ipkg_build $opkg_template
    mv $project_name"_"$release_version-1_all.ipk $dist/$project_name"_"$release_version-1_$goarch.ipk
done

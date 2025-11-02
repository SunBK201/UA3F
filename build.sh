#!/bin/sh
# ------------------------------
#   ./build.sh amd64 arm64
#   ./build.sh
# ------------------------------

set -e

project_name="ua3f"
release_version="1.1.0"
target=main.go
dist=./dist
release_dir=./bin

ALL_ARCHS="amd64 arm arm64 mipsle mips64 riscv64 386 mipsle-softfloat mipsle-hardfloat armv7 armv8"

if [ $# -gt 0 ]; then
    ARCH_LIST="$@"
else
    ARCH_LIST="$ALL_ARCHS"
fi

rm -rf "$release_dir"/* "$dist"/*
mkdir -p "$release_dir" "$dist/bin"

cd "$(dirname "$0")"
gofmt -w ./
cd "$(dirname "$0")/src"

for goarch in $ARCH_LIST; do
    obj_name=$project_name-$release_version-$goarch
    echo ">>> Building for $goarch ..."

    case "$goarch" in
    mipsle-softfloat)
        GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -trimpath -ldflags="-s -w" -o "$obj_name" "$target"
        ;;
    mipsle-hardfloat)
        GOOS=linux GOARCH=mipsle GOMIPS=hardfloat go build -trimpath -ldflags="-s -w" -o "$obj_name" "$target"
        ;;
    armv7)
        GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -ldflags="-s -w" -o "$obj_name" "$target"
        ;;
    armv8)
        # armv8 alias to arm64
        if [ ! -f "../dist/bin/$project_name-$release_version-arm64" ]; then
            echo ">>> Building arm64 (for armv8 alias)"
            GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o "$project_name-$release_version-arm64" "$target"
            cp "$project_name-$release_version-arm64" ../dist/bin/
        fi
        cp "../dist/bin/$project_name-$release_version-arm64" "$obj_name"
        ;;
    *)
        GOOS=linux GOARCH="$goarch" go build -trimpath -ldflags="-s -w" -o "$obj_name" "$target"
        ;;
    esac

    cp "$obj_name" ../dist/bin/
    mv "$obj_name" "$project_name"
    tar -zcf ../bin/$project_name-$release_version-$goarch.tar.gz "$project_name"
    rm -f "$project_name"
done

cd ../bin
rm -f sha1sum.txt
for file in ./*; do
    md5 -r "$file" >>sha1sum.txt
done

cd ..
opkg_template=./ipkg
ipkg_build=ipkg-build.sh

mkdir -p \
    $opkg_template/usr/bin \
    $opkg_template/usr/lib/lua/luci/controller \
    $opkg_template/usr/lib/lua/luci/model/cbi \
    $opkg_template/usr/lib/lua/luci/view/ua3f \
    $opkg_template/usr/lib/lua/luci/i18n \
    $opkg_template/etc/init.d \
    $opkg_template/etc/config

cp openwrt/files/luci/controller.lua $opkg_template/usr/lib/lua/luci/controller/ua3f.lua
cp openwrt/files/luci/cbi.lua $opkg_template/usr/lib/lua/luci/model/cbi/ua3f.lua
cp openwrt/files/luci/statistics.htm $opkg_template/usr/lib/lua/luci/view/ua3f/statistics.htm
cp openwrt/files/ua3f.init $opkg_template/etc/init.d/ua3f
cp openwrt/files/ua3f.uci $opkg_template/etc/config/ua3f
./po2lmo openwrt/po/zh_cn/ua3f.po $opkg_template/usr/lib/lua/luci/i18n/ua3f.zh-cn.lmo

for goarch in $ARCH_LIST; do
    obj_name=$project_name-$release_version-$goarch
    [ -f "$dist/bin/$obj_name" ] || continue

    mv "$dist/bin/$obj_name" $opkg_template/usr/bin/ua3f
    sh "$ipkg_build" "$opkg_template"
    mv "$project_name"_"$release_version"-1_all.ipk "$dist/$project_name"_"$release_version"-1_"$goarch".ipk
done

echo "✅ Build complete：$ARCH_LIST"

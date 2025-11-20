#!/bin/sh
# --------------------------------
# Usage examples:
#   ./scripts/build.sh                 # default: linux all + win/darwin 64/arm64
#   ./scripts/build.sh linux/amd64 windows/arm64
# --------------------------------

set -e

project_name="ua3f"
release_version="1.8.4"
target=main.go

LINUX_ARCHS="amd64 arm arm64 mipsle mips64 riscv64 386 mipsle-softfloat mipsle-hardfloat armv7 armv8"

# default build target（Linux + Windows + macOS）
DEFAULT_TARGETS=""
for a in $LINUX_ARCHS; do
    DEFAULT_TARGETS="$DEFAULT_TARGETS linux/$a"
done
DEFAULT_TARGETS="$DEFAULT_TARGETS windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 android/arm64"

if [ $# -gt 0 ]; then
    TARGET_LIST="$@"
else
    TARGET_LIST="$DEFAULT_TARGETS"
fi

script_dir="$(cd "$(dirname "$0")" && pwd)"
project_root="$(cd "$script_dir/.." && pwd)"
dist="$project_root/dist"
release_dir="$project_root/bin"

rm -rf "$release_dir"/* "$dist"/*
mkdir -p "$release_dir" "$dist/bin"

cd "$project_root"
gofmt -w ./
cd "$project_root/src"

for target_item in $TARGET_LIST; do
    goos=$(echo "$target_item" | cut -d'/' -f1)
    goarch=$(echo "$target_item" | cut -d'/' -f2)

    obj_name=$project_name-$release_version-${goos}-${goarch}
    echo ">>> Building for $goos/$goarch ..."

    case "$goarch" in
    mipsle-softfloat)
        CGO_ENABLED=0 GOOS=$goos GOARCH=mipsle GOMIPS=softfloat go build -trimpath -ldflags="-s -w" -o "$obj_name" "$target"
        ;;
    mipsle-hardfloat)
        CGO_ENABLED=0 GOOS=$goos GOARCH=mipsle GOMIPS=hardfloat go build -trimpath -ldflags="-s -w" -o "$obj_name" "$target"
        ;;
    armv7)
        CGO_ENABLED=0 GOOS=$goos GOARCH=arm GOARM=7 go build -trimpath -ldflags="-s -w" -o "$obj_name" "$target"
        ;;
    armv8)
        alias_name=$project_name-$release_version-${goos}-arm64
        if [ ! -f "$project_root/dist/bin/$alias_name" ]; then
            echo ">>> Building $goos/arm64 (for armv8 alias)"
            CGO_ENABLED=0 GOOS=$goos GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o "$alias_name" "$target"
            cp "$alias_name" "$project_root/dist/bin/"
        fi
        cp "$project_root/dist/bin/$alias_name" "$obj_name"
        ;;
    *)
        CGO_ENABLED=0 GOOS=$goos GOARCH="$goarch" go build -trimpath -ldflags="-s -w" -o "$obj_name" "$target"
        ;;
    esac

    cp "$obj_name" "$project_root/dist/bin/"

    echo ">>> Packaging for $goos/$goarch ..."
    if [ "$goos" = "windows" ]; then
        mv "$obj_name" "$project_name.exe"
        zip "$project_root/bin/$project_name-$release_version-${goos}-${goarch}.zip" "$project_name.exe"
        rm -f "$project_name.exe"
    elif [ "$goos" = "darwin" ]; then
        mv "$obj_name" "$project_name"
        zip -q "$project_root/bin/$project_name-$release_version-${goos}-${goarch}.zip" "$project_name"
        rm -f "$project_name"
    else
        mv "$obj_name" "$project_name"
        # only package linux/arm64 and linux/amd64
        if [ "$goos" = "linux" ] && [ "$goarch" != "arm64" ] && [ "$goarch" != "amd64" ]; then
            echo ">>> Skipping packaging for linux/$goarch (only arm64 and amd64 are packaged)"
            rm -f "$project_name"
        else
            tar -zcf "$project_root/bin/$project_name-$release_version-${goos}-${goarch}.tar.gz" "$project_name"
            rm -f "$project_name"
        fi
    fi
done

cd ..
opkg_template=./scripts/ipkg
ipkg_build=./scripts/ipkg-build.sh

mkdir -p \
    $opkg_template/usr/bin \
    $opkg_template/usr/lib/lua/luci/controller \
    $opkg_template/usr/lib/lua/luci/model/cbi/ua3f \
    $opkg_template/usr/lib/lua/luci/view/ua3f \
    $opkg_template/usr/lib/lua/luci/i18n \
    $opkg_template/etc/init.d \
    $opkg_template/etc/config

cp openwrt/files/ua3f.init $opkg_template/etc/init.d/ua3f
cp openwrt/files/ua3f.uci $opkg_template/etc/config/ua3f
cp -r openwrt/files/luci/* $opkg_template/usr/lib/lua/luci/
po2lmo openwrt/po/zh_cn/ua3f.po $opkg_template/usr/lib/lua/luci/i18n/ua3f.zh-cn.lmo

# only build ipk for linux targets
for target_item in $TARGET_LIST; do
    goos=$(echo "$target_item" | cut -d'/' -f1)
    goarch=$(echo "$target_item" | cut -d'/' -f2)

    # only package linux
    [ "$goos" = "linux" ] || continue

    obj_name=$project_name-$release_version-${goos}-${goarch}
    [ -f "$dist/bin/$obj_name" ] || continue

    echo ">>> Building IPK for $goos/$goarch ..."

    mv "$dist/bin/$obj_name" $opkg_template/usr/bin/ua3f
    sh "$ipkg_build" "$opkg_template"
    mv "$project_root/$project_name"_"$release_version"-1_all.ipk "$dist/$project_name"_"$release_version"-1_"${goos}_${goarch}".ipk
done

rm -rf "$dist/bin"
mv "$release_dir"/* "$dist/"
rm -rf "$release_dir"

echo "✅ Build complete：$TARGET_LIST"

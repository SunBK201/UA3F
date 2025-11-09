#!/bin/sh
# --------------------------------
# Usage examples:
#   ./build.sh                 # default: linux all + win/darwin 64/arm64
#   ./build.sh linux/amd64 windows/arm64
# --------------------------------

set -e

project_name="ua3f"
release_version="1.6.0"
target=main.go
dist=./dist
release_dir=./bin

# Linux 默认支持的架构
LINUX_ARCHS="amd64 arm arm64 mipsle mips64 riscv64 386 mipsle-softfloat mipsle-hardfloat armv7 armv8"

# 默认构建目标（Linux + Windows + macOS）
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

rm -rf "$release_dir"/* "$dist"/*
mkdir -p "$release_dir" "$dist/bin"

cd "$(dirname "$0")"
gofmt -w ./
cd "$(dirname "$0")/src"

for target_item in $TARGET_LIST; do
    goos=$(echo "$target_item" | cut -d'/' -f1)
    goarch=$(echo "$target_item" | cut -d'/' -f2)

    obj_name=$project_name-$release_version-${goos}-${goarch}
    echo ">>> Building for $goos/$goarch ..."

    case "$goarch" in
    mipsle-softfloat)
        GOOS=$goos GOARCH=mipsle GOMIPS=softfloat go build -trimpath -ldflags="-s -w" -o "$obj_name" "$target"
        ;;
    mipsle-hardfloat)
        GOOS=$goos GOARCH=mipsle GOMIPS=hardfloat go build -trimpath -ldflags="-s -w" -o "$obj_name" "$target"
        ;;
    armv7)
        GOOS=$goos GOARCH=arm GOARM=7 go build -trimpath -ldflags="-s -w" -o "$obj_name" "$target"
        ;;
    armv8)
        alias_name=$project_name-$release_version-${goos}-arm64
        if [ ! -f "../dist/bin/$alias_name" ]; then
            echo ">>> Building $goos/arm64 (for armv8 alias)"
            GOOS=$goos GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o "$alias_name" "$target"
            cp "$alias_name" ../dist/bin/
        fi
        cp "../dist/bin/$alias_name" "$obj_name"
        ;;
    *)
        GOOS=$goos GOARCH="$goarch" go build -trimpath -ldflags="-s -w" -o "$obj_name" "$target"
        ;;
    esac

    cp "$obj_name" ../dist/bin/

    echo ">>> Packaging for $goos/$goarch ..."
    if [ "$goos" = "windows" ]; then
        mv "$obj_name" "$project_name.exe"
        zip ../bin/$project_name-$release_version-"${goos}"-"${goarch}".zip "$project_name.exe"
        rm -f "$project_name.exe"
    elif [ "$goos" = "darwin" ]; then
        mv "$obj_name" "$project_name"
        zip -q ../bin/$project_name-$release_version-"${goos}"-"${goarch}".zip "$project_name"
        rm -f "$project_name"
    else
        mv "$obj_name" "$project_name"
        tar -zcf ../bin/$project_name-$release_version-"${goos}"-"${goarch}".tar.gz "$project_name"
        rm -f "$project_name"
    fi
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
    $opkg_template/usr/lib/lua/luci/model/cbi/ua3f \
    $opkg_template/usr/lib/lua/luci/view/ua3f \
    $opkg_template/usr/lib/lua/luci/i18n \
    $opkg_template/etc/init.d \
    $opkg_template/etc/config

cp openwrt/files/ua3f.init $opkg_template/etc/init.d/ua3f
cp openwrt/files/ua3f.uci $opkg_template/etc/config/ua3f
cp -r openwrt/files/luci/* $opkg_template/usr/lib/lua/luci/
./po2lmo openwrt/po/zh_cn/ua3f.po $opkg_template/usr/lib/lua/luci/i18n/ua3f.zh-cn.lmo

# 仅 Linux 平台生成 ipk 包
for target_item in $TARGET_LIST; do
    goos=$(echo "$target_item" | cut -d'/' -f1)
    goarch=$(echo "$target_item" | cut -d'/' -f2)

    # 只打包 Linux 的
    [ "$goos" = "linux" ] || continue

    obj_name=$project_name-$release_version-${goos}-${goarch}
    [ -f "$dist/bin/$obj_name" ] || continue

    mv "$dist/bin/$obj_name" $opkg_template/usr/bin/ua3f
    sh "$ipkg_build" "$opkg_template"
    mv "$project_name"_"$release_version"-1_all.ipk "$dist/$project_name"_"$release_version"-1_"${goos}_${goarch}".ipk
done

echo "✅ Build complete：$TARGET_LIST"

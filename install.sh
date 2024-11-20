#!/bin/sh

getcpucore() {
    cputype=$(uname -ms | tr ' ' '_' | tr '[A-Z]' '[a-z]')
    [ -n "$(echo $cputype | grep -E "linux.*armv.*")" ] && cpucore="armv5"
    [ -n "$(echo $cputype | grep -E "linux.*armv7.*")" ] && [ -n "$(cat /proc/cpuinfo | grep vfp)" ] && [ ! -d /jffs/clash ] && cpucore="armv7"
    [ -n "$(echo $cputype | grep -E "linux.*aarch64.*|linux.*armv8.*")" ] && cpucore="armv8"
    [ -n "$(echo $cputype | grep -E "linux.*86.*")" ] && cpucore="386"
    [ -n "$(echo $cputype | grep -E "linux.*86_64.*")" ] && cpucore="amd64"
    if [ -n "$(echo $cputype | grep -E "linux.*mips.*")" ]; then
        mipstype=$(echo -n I | hexdump -o 2>/dev/null | awk '{ print substr($2,6,1); exit}')
        [ "$mipstype" = "0" ] && cpucore="mips-softfloat" || cpucore="mipsle-softfloat"
    fi
}

ckcmd() {
    command -v sh >/dev/null 2>&1 && command -v $1 >/dev/null 2>&1 || type $1 >/dev/null 2>&1
}

chmod_clash() {
    if id -u shellclash >/dev/null 2>&1; then
        chmod o+w /etc/clash >/dev/null 2>&1
    fi
    if id -u shellcrash >/dev/null 2>&1; then
        chmod o+w /etc/clash >/dev/null 2>&1
    fi
}

ck_ua3f_log() {
    chmod ugo+w /var/log
    if [ -f "/var/log/ua3f.log" ]; then
        rm "/var/log/ua3f.log"
    fi
    mkdir -p /var/log/ua3f && chmod ugo+w /var/log/ua3f
}

dl_ua3f() {
    wget https://blog.sunbk201.site/cdn/bin/$ua3f_tar
    if [ $? -ne 0 ]; then
        echo "Download UA3F Failed, Please Retry."
        exit 1
    fi

    wget https://blog.sunbk201.site/cdn/ua3f.init
    if [ $? -ne 0 ]; then
        echo "Download ua3f.init Failed, Please Retry."
        exit 1
    fi

    wget https://blog.sunbk201.site/cdn/ua3f.uci
    if [ $? -ne 0 ]; then
        echo "Download ua3f.uci Failed, Please Retry."
        exit 1
    fi

    wget https://blog.sunbk201.site/cdn/cbi.lua
    if [ $? -ne 0 ]; then
        echo "Download cbi.lua Failed, Please Retry."
        exit 1
    fi

    wget https://blog.sunbk201.site/cdn/controller.lua
    if [ $? -ne 0 ]; then
        echo "Download controller.lua Failed, Please Retry."
        exit 1
    fi
}

clean_ua3f() {
    if [ -f "/etc/init.d/ua3f.init" ]; then
        rm "/etc/init.d/ua3f.init"
    fi
    if [ -f "/etc/init.d/ua3f" ]; then
        rm "/etc/init.d/ua3f"
    fi
    if [ -f "/etc/config/ua3f" ]; then
        rm "/etc/config/ua3f"
    fi
    if [ -f "/usr/lib/lua/luci/model/cbi/ua3f.lua" ]; then
        rm "/usr/lib/lua/luci/model/cbi/ua3f.lua"
    fi
    if [ -f "/usr/lib/lua/luci/controller/ua3f.lua" ]; then
        rm "/usr/lib/lua/luci/controller/ua3f.lua"
    fi
}

install_ua3f() {
    tar zxf $ua3f_tar && rm -f $ua3f_tar
    mv ua3f /usr/bin/ua3f && chmod +x /usr/bin/ua3f
    mv ua3f.uci /etc/config/ua3f && chmod +x /etc/config/ua3f
    mv ua3f.init /etc/init.d/ua3f && chmod +x /etc/init.d/ua3f
    mkdir -p /usr/lib/lua/luci/model/cbi
    mkdir -p /usr/lib/lua/luci/controller
    mv cbi.lua /usr/lib/lua/luci/model/cbi/ua3f.lua
    mv controller.lua /usr/lib/lua/luci/controller/ua3f.lua
}

cd /root
getcpucore

version=0.7.1
ua3f_tar=ua3f-$version-$cpucore.tar.gz

chmod_clash

if ! command -v sudo >/dev/null 2>&1; then
    opkg update >/dev/null 2>&1 && opkg install sudo >/dev/null 2>&1
fi

ck_ua3f_log
if [ -f "$ua3f_tar" ]; then
    rm "$ua3f_tar"
fi
dl_ua3f
clean_ua3f
install_ua3f
/etc/init.d/ua3f enable
if [ $? -eq 0 ]; then
    echo "Install UA3F Success."
fi

rm /tmp/luci-modulecache/* >/dev/null 2>&1
rm /tmp/luci-indexcache* >/dev/null 2>&1
service ua3f reload >/dev/null 2>&1

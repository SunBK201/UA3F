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

cd /root
getcpucore

version=0.1.1
ua3f_tar=ua3f-$version-$cpucore.tar.gz

if [ -f "ua3f" ]; then
    rm "ua3f"
    killall ua3f &> /dev/null
fi

if ! command -v sudo &> /dev/null; then
    opkg update &> /dev/null && opkg install sudo &> /dev/null
fi

chmod ugo+w /var/log
if [ -f "/var/log/ua3f.log" ]; then
    rm "/var/log/ua3f.log"
fi

if [ -f "$ua3f_tar" ]; then
    rm "$ua3f_tar"
fi

wget https://blog.sunbk201.site/cdn/bin/$ua3f_tar
if [ $? -ne 0 ]; then
    echo "Download UA3F Failed, Please Retry."
    exit 1
fi
tar zxf $ua3f_tar && rm -f $ua3f_tar
chmod +x ua3f

if [ -f "ua3f.service" ]; then
    rm "ua3f.service"
fi
wget https://blog.sunbk201.site/cdn/ua3f.service
if [ $? -ne 0 ]; then
    echo "Download ua3f.service Failed, Please Retry."
    exit 1
fi
mv ua3f.service /etc/init.d/ && chmod +x /etc/init.d/ua3f.service
/etc/init.d/ua3f.service enable

if [ $? -eq 0 ]; then
    echo "Install UA3F Success."
    echo "Use /etc/init.d/ua3f.service {start|stop|restart} to control UA3F."
fi
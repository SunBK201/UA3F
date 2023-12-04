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

version=0.0.4
ua3f_tar=ua3f-$version-$cpucore.tar.gz

if [ -f "ua3f" ]; then
    rm "ua3f"
fi

if [ -f "$ua3f_tar" ]; then
    rm "$ua3f_tar"
fi

wget https://fastly.jsdelivr.net/gh/SunBK201/UA3F@master/release/$ua3f_tar
if [ $? -ne 0 ]; then
    echo "Download UA3F Failed, Please Retry."
    exit 1
fi
tar zxf $ua3f_tar && rm -f $ua3f_tar
chmod +x ua3f

if [ -f "ua3f.service" ]; then
    rm "ua3f.service"
fi
wget https://fastly.jsdelivr.net/gh/SunBK201/UA3F@master/ua3f.service
if [ $? -ne 0 ]; then
    echo "Download ua3f.service Failed, Please Retry."
    exit 1
fi
mv ua3f.service /etc/init.d/ && chmod +x /etc/init.d/ua3f.service
/etc/init.d/ua3f.service enable

if [ $? -eq 0 ]; then
    echo "Install UA3F Success."
fi
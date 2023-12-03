# UA3F

UA3F 是新一代 HTTP User-Agent 修改方法，对外作为一个 SOCK5 服务，可以部署在路由器等设备等设备进行透明 UA 修改。

## 部署

[Release](https://github.com/SunBK201/UA3F/releases) 页面已经提供常见架构的编译版本，可以根据自己架构下载并解压到路由器等设备上。

## 使用

参数:
- `-p <port>`: 端口号，默认 1080
- `-f <UA>`: 自定义 UA，默认 FFF

### 手动启动
```bash
ua3f -p <port> -f <UA>
```

### 作为后台服务运行
将 `ua3f` 置于 `/root/ua3f`

将 [ua3f.service](ua3f.service) 置于 `/etc/init.d/ua3f.service`

执行下面的命令：
```bash
chmod +x /etc/init.d/ua3f.service
service ua3f.service enable
service ua3f.service start
```

### Clash 的配置建议
请确保 `PROCESS-NAME,ua3f,DIRECT` 置于规则列表顶部。

可以在 `PROCESS-NAME,ua3f,DIRECT` 与 `MATCH,ua3f` 之间按需加入自定义加密代理规则。如果上述 2 条规则之间加入 DIRECT 规则，请确保匹配该规则的流量属于非 HTTP 协议流量

请不要将从 [Release](https://github.com/SunBK201/UA3F/releases) 下载解压得到的 `ua3f` 二进制文件修改名称，
如需修改，则需要在 `PROCESS-NAME,ua3f,DIRECT` 中修改相应的名称。

```yaml
proxies:
  - name: "ua3f"
    type: socks5
    server: 127.0.0.1
    port: 1080
    url: http://connectivitycheck.platform.hicloud.com/generate_204

rules:
  - PROCESS-NAME,ua3f,DIRECT
  - MATCH,ua3f
```
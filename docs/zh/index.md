---
layout: home
hero:
  name: UA3F
  text: 高级网络流量重写代理
  tagline: 在 HTTP、SOCKS5、TPROXY、REDIRECT 与 NFQUEUE 模式下透明重写 HTTP(S) 流量。
  image:
    src: https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-210.png
    alt: UA3F
  actions:
    - theme: brand
      text: 快速开始
      link: /zh/guide/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/SunBK201/UA3F
    - theme: alt
      text: English
      link: /
features:
  - title: 多模式入站
    details: 可作为 HTTP、SOCKS5、TPROXY、REDIRECT 或 NFQUEUE 服务运行，覆盖代理式与透明式部署。
  - title: 灵活规则系统
    details: 支持域名、端口、IP、Header、URL 等匹配条件，并执行重写、删除、拒绝、丢弃和重定向动作。
  - title: HTTPS MitM
    details: 可对指定主机名启用 MitM 解密，在请求或响应方向重写 Header 与 Body。
  - title: OpenWrt 原生支持
    details: 提供 opkg/apk 包、LuCI 页面、Clash 伴生配置与多种透明代理部署方式。
  - title: L3 重写 与 Desync
    details: 支持 TTL、IPID、TCP Timestamp、TCP 初始窗口重写，以及分片乱序与混淆注入。
  - title: API 与可观测
    details: 内置 RESTful API，可查询版本、配置、规则与实时日志，并支持运行时重启。
---

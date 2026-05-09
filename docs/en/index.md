---
layout: home
hero:
  name: UA3F
  text: Advanced Network Traffic Rewriting Proxy
  tagline: Transparently rewrite HTTP(S) traffic in HTTP, SOCKS5, TPROXY, REDIRECT, and NFQUEUE modes.
  image:
    src: https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-210.png
    alt: UA3F
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/SunBK201/UA3F
    - theme: alt
      text: 简体中文
      link: /zh/
features:
  - title: Multiple server modes
    details: Run as HTTP, SOCKS5, TPROXY, REDIRECT, or NFQUEUE to support explicit proxy and transparent gateway deployments.
  - title: Flexible rule engine
    details: Match domains, ports, IP ranges, headers, and URLs, then rewrite, delete, reject, drop, or redirect traffic.
  - title: HTTPS MitM
    details: Enable MitM for selected hostnames to rewrite request and response headers or bodies over HTTPS.
  - title: OpenWrt ready
    details: Ships opkg/apk packages, LuCI integration, Clash coexistence examples, and transparent proxy modes.
  - title: L3 Rewriting and Desync
    details: Rewrite TTL, IPID, TCP Timestamp, and TCP Initial Window, with TCP reordering and injection based Desync support.
  - title: API and observability
    details: Query version, configuration, rules, and logs through the built-in RESTful API.
---

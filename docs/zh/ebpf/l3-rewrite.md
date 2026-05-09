# L3 重写 eBPF 加速

UA3F 可以通过在出口网卡挂载 TC eBPF 程序来加速 L3 重写。

对于 TTL、IPID 重写等功能，UA3F `<= v3.1.0` 基于 netfilter NFQUEUE 将流量从内核态劫持到用户态，并在 L3 视角进行重写修改。该路径涉及上下文切换与数据拷贝，性能开销较高。

UA3F `3.2.0` 引入 eBPF 加速能力，基于 TC egress 将重写逻辑下沉到内核态，大幅降低包处理开销并提高转发效率。目前 UA3F L3 eBPF 加速要求 Linux 内核版本 `>= 5.15`。

> [!IMPORTANT]
> eBPF 对 Linux 内核和处理器架构均有要求。常见可用架构包括 x86-64、x86-32、ARM32、ARM64、MIPS 64、RISC-V；不支持 MIPS 32。

## 配置

```yaml
l3-rewrite:
  ttl: true
  ipid: true
  tcpts: true
  tcpwin: true
  bpf-offload: true
```

开启 `l3-rewrite.bpf-offload` 后，UA3F 会优先尝试使用 TC eBPF 处理已启用的 L3 重写功能。若初始化失败，可关闭该选项回退到 netfilter/NFQUEUE 路径。

## 程序选择

UA3F 会根据启用的 L3 功能选择 TC 程序：

| 功能 | TC 程序 |
| --- | --- |
| IPID | `set_ip_id_zero` |
| TTL | `set_ip_ttl` |
| TCP 初始窗口 | `set_tcp_syn_window` |
| TCP 时间戳 | `clear_tcp_syn_ts` |

## 挂载行为

UA3F 会选择带 IPv4 默认路由、封装类型为 Ethernet 或 PPP 的出口接口，并跳过 loopback、`lo` 和 `br-lan`。它会优先尝试较新内核的 TCX，失败后回退到 classic `cls_bpf`。

如果 TC eBPF 初始化失败，关闭 `l3-rewrite.bpf-offload` 可回退到 netfilter/NFQUEUE 路径。

## 性能测试

UA3F L3 重写 eBPF 性能测试使用 `netperf` 进行。测试开启 L3 重写全部功能选项，并分别测试 `TCP_STREAM`、`TCP_RR`、`TCP_CRR`。

整体优化结果如下：

| 测试类型 | 整体吞吐/QPS | CPU 占用 | 本地时延 |
| --- | --- | --- | --- |
| TCP_STREAM | +4.56% | -82.46% | -83.23% |
| TCP_RR | +9.22% | -77.01% | -78.93% |
| TCP_CRR | +1.19% | -52.73% | -53.75% |

相较于 NFQUEUE，eBPF 在三类测试中使整体吞吐平均提升约 5.0%，同时 CPU 使用率平均下降约 70.7%，本地时延平均降低约 72.0%。

### TCP_STREAM

| 指标 | NFQUEUE | eBPF | 提升幅度 |
| --- | --- | --- | --- |
| Throughput (Mbps) | 409.83 | 428.53 | +4.56% |
| Local CPU (%) | 44.87 | 7.87 | -82.46% |
| Send us/KB | 35.873 | 6.020 | -83.23% |

### TCP_RR

| 指标 | NFQUEUE | eBPF | 提升幅度 |
| --- | --- | --- | --- |
| Transactions/s | 252.03 | 275.26 | +9.22% |
| Local CPU (%) | 2.74 | 0.63 | -77.01% |
| Local us/Tr | 435.562 | 91.781 | -78.93% |

### TCP_CRR

| 指标 | NFQUEUE | eBPF | 提升幅度 |
| --- | --- | --- | --- |
| Transactions/s | 25.20 | 25.50 | +1.19% |
| Local CPU (%) | 0.55 | 0.26 | -52.73% |
| Local us/Tr | 879.750 | 406.912 | -53.75% |

# L3 Rewrite eBPF Acceleration

UA3F can accelerate L3 rewrite by attaching TC eBPF programs to egress interfaces.

For features such as TTL and IPID rewriting, UA3F `<= v3.1.0` used netfilter NFQUEUE to redirect traffic from kernel space to user space and rewrite packets from an L3 perspective. That path requires context switches and packet data copies, so it has higher runtime overhead.

UA3F `3.2.0` introduced eBPF acceleration. It moves rewrite logic into the kernel through TC egress, reducing packet-processing overhead and improving forwarding efficiency. UA3F L3 eBPF acceleration currently requires Linux kernel `>= 5.15`.

> [!IMPORTANT]
> eBPF depends on both Linux kernel support and CPU architecture support. Common supported architectures include x86-64, x86-32, ARM32, ARM64, MIPS 64, and RISC-V. MIPS 32 is not supported.

## Configuration

```yaml
l3-rewrite:
  ttl: true
  ipid: true
  tcpts: true
  tcpwin: true
  bpf-offload: true
```

When `l3-rewrite.bpf-offload` is enabled, UA3F first tries to process enabled L3 rewrite features through TC eBPF. If initialization fails, disable this option to fall back to the netfilter/NFQUEUE path.

## Program selection

UA3F selects TC programs based on enabled L3 features:

| Feature | TC program |
| --- | --- |
| IPID | `set_ip_id_zero` |
| TTL | `set_ip_ttl` |
| TCP Initial Window | `set_tcp_syn_window` |
| TCP Timestamp | `clear_tcp_syn_ts` |

## Attachment behavior

UA3F attaches to eligible IPv4 default-route interfaces that are Ethernet or PPP and skips loopback, `lo`, and `br-lan`. It tries TCX first on newer kernels and falls back to classic `cls_bpf`.

If TC eBPF initialization fails, disable `l3-rewrite.bpf-offload` to use the netfilter/NFQUEUE path.

## Performance testing

UA3F L3 rewrite eBPF performance was tested with `netperf`. The tests enabled all L3 rewrite options and covered `TCP_STREAM`, `TCP_RR`, and `TCP_CRR`.

Overall optimization results:

| Test type | Throughput/QPS | CPU usage | Local latency |
| --- | --- | --- | --- |
| TCP_STREAM | +4.56% | -82.46% | -83.23% |
| TCP_RR | +9.22% | -77.01% | -78.93% |
| TCP_CRR | +1.19% | -52.73% | -53.75% |

Compared with NFQUEUE, eBPF improved average throughput by about 5.0%, reduced average CPU usage by about 70.7%, and reduced average local latency by about 72.0% across the three test types.

### TCP_STREAM

| Metric | NFQUEUE | eBPF | Improvement |
| --- | --- | --- | --- |
| Throughput (Mbps) | 409.83 | 428.53 | +4.56% |
| Local CPU (%) | 44.87 | 7.87 | -82.46% |
| Send us/KB | 35.873 | 6.020 | -83.23% |

### TCP_RR

| Metric | NFQUEUE | eBPF | Improvement |
| --- | --- | --- | --- |
| Transactions/s | 252.03 | 275.26 | +9.22% |
| Local CPU (%) | 2.74 | 0.63 | -77.01% |
| Local us/Tr | 435.562 | 91.781 | -78.93% |

### TCP_CRR

| Metric | NFQUEUE | eBPF | Improvement |
| --- | --- | --- | --- |
| Transactions/s | 25.20 | 25.50 | +1.19% |
| Local CPU (%) | 0.55 | 0.26 | -52.73% |
| Local us/Tr | 879.750 | 406.912 | -53.75% |

// +build ignore

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

struct
{
    __uint(type, BPF_MAP_TYPE_SOCKHASH);
    __uint(max_entries, 131072); // >= 2 * max_connections
    __type(key, __u64); // socket cookie
    __type(value, __u32); // socket fd from userspace
} sockhash SEC(".maps");

struct
{
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 131072);
    __type(key, __u64); // cookie
    __type(value, __u64); // peer cookie
} peer SEC(".maps");

SEC("sk_skb/stream_parser")
int stream_parser(struct __sk_buff* skb)
{
    return skb->len;
}

SEC("sk_skb/stream_verdict")
int stream_verdict(struct __sk_buff* skb)
{
    __u64 c = bpf_get_socket_cookie(skb);
    __u64* p = bpf_map_lookup_elem(&peer, &c);
    if (!p) {
        return SK_PASS;
    }

    __u64 peer_cookie = *p;
    return bpf_sk_redirect_hash(skb, &sockhash, &peer_cookie, 0);
}

char LICENSE[] SEC("license") = "GPL";

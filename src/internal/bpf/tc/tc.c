// +build ignore

#include <linux/bpf.h>
#include <linux/pkt_cls.h>

#include <linux/if_ether.h>
#include <linux/in.h>
#include <linux/ip.h>
#include <linux/tcp.h>

#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>

#define MAX_TCP_OPT_LEN 40
#define IP_TTL_DEFAULT 64

static __always_inline int parse_l2(void* data, void* data_end, __u32* off, __u16* proto)
{
    __u8* cursor = data;

    struct ethhdr* eth = (struct ethhdr*)cursor;
    if ((void*)(eth + 1) > data_end)
        return -1;

    *proto = bpf_ntohs(eth->h_proto);
    *off = sizeof(*eth);

    return 0;
}

static __always_inline int parse_ipv4_tcp(void* data, void* data_end, __u32* off, struct iphdr** ip_out)
{
    __u8* cursor = data;

    struct iphdr* ip = (struct iphdr*)(cursor + *off);
    if ((void*)(ip + 1) > data_end)
        return -1;

    if (ip->version != 4)
        return -1;

    int ip_hlen = ip->ihl * 4;
    if (ip_hlen < (int)sizeof(*ip))
        return -1;

    if ((void*)(cursor + *off + ip_hlen) > data_end)
        return -1;

    if (ip->protocol != IPPROTO_TCP)
        return -1;

    // ignore fragmented packet
    __u16 frag = bpf_ntohs(ip->frag_off);
    if (frag & 0x3FFF) // MF=1 or offset>0
        return -1;

    *ip_out = ip;
    *off += ip_hlen;
    return 0;
}

static __always_inline int parse_tcp_hdr(void* data, void* data_end, __u32 off,
    struct tcphdr** tcp_out,
    int* opt_off_out, int* opt_len_out)
{
    __u8* cursor = data;

    struct tcphdr* tcp = (struct tcphdr*)(cursor + off);
    if ((void*)(tcp + 1) > data_end)
        return -1;

    int tcp_hlen = tcp->doff * 4;
    if (tcp_hlen < (int)sizeof(*tcp))
        return -1;

    if ((void*)(cursor + off + tcp_hlen) > data_end)
        return -1;

    *tcp_out = tcp;
    *opt_off_out = (int)off + (int)sizeof(*tcp);
    *opt_len_out = tcp_hlen - (int)sizeof(*tcp);
    return 0;
}

static __always_inline int is_first_syn(const struct tcphdr* tcp)
{
    // SYN=1, ACK=0
    return tcp->syn && !tcp->ack;
}

static __always_inline void clear_tcp_ts_option(struct __sk_buff* skb, int opt_off, int opt_len, int csum_off)
{
    if (opt_len < 10)
        return;

    int i = 0;

    for (int iter = 0; iter < MAX_TCP_OPT_LEN; iter++) {
        if (i >= opt_len)
            break;

        __u8 kind = 0;
        if (bpf_skb_load_bytes(skb, opt_off + i, &kind, 1) < 0)
            break;

        if (kind == 0) // EOL
            break;

        if (kind == 1) { // NOP
            i++;
            continue;
        }

        if (i + 1 >= opt_len)
            break;

        __u8 len = 0;
        if (bpf_skb_load_bytes(skb, opt_off + i + 1, &len, 1) < 0)
            break;

        if (len < 2)
            break;

        if (i + len > opt_len)
            break;

        // Timestamp: kind=8, len=10, TSval(4)+TSecr(4)
        // Replace entire 10-byte option with NOPs (0x01) to fully remove it
        if (kind == 8 && len == 10) {
            __be16 old_words[5];
            __u8 nops[10];

            if (bpf_skb_load_bytes(skb, opt_off + i, old_words, sizeof(old_words)) < 0)
                break;

            __builtin_memset(nops, 1, sizeof(nops));
            (void)bpf_skb_store_bytes(skb, opt_off + i, nops, sizeof(nops),
                0);

            for (int w = 0; w < 5; w++) {
                __be16 new_word = bpf_htons(0x0101);
                (void)bpf_l4_csum_replace(skb, csum_off, old_words[w], new_word,
                    sizeof(new_word));
            }
            break;
        }

        i += len;
    }
}

SEC("tc/egress")
int clear_tcp_syn_ts(struct __sk_buff* skb)
{
    void* data = (void*)(long)skb->data;
    void* data_end = (void*)(long)skb->data_end;

    __u32 off = 0;
    __u16 proto = 0;

    if (parse_l2(data, data_end, &off, &proto) < 0)
        return TC_ACT_OK;

    if (proto != ETH_P_IP)
        return TC_ACT_OK;

    struct iphdr* ip = NULL;
    if (parse_ipv4_tcp(data, data_end, &off, &ip) < 0)
        return TC_ACT_OK;

    struct tcphdr* tcp = NULL;
    int opt_off = 0, opt_len = 0;

    if (parse_tcp_hdr(data, data_end, off, &tcp, &opt_off, &opt_len) < 0)
        return TC_ACT_OK;

    if (!is_first_syn(tcp))
        return TC_ACT_OK;

    clear_tcp_ts_option(skb, opt_off, opt_len, off + offsetof(struct tcphdr, check));

    return TCX_NEXT;
}

SEC("tc/egress")
int set_tcp_syn_window(struct __sk_buff* skb)
{
    void* data = (void*)(long)skb->data;
    void* data_end = (void*)(long)skb->data_end;

    __u32 off = 0;
    __u16 proto = 0;

    if (parse_l2(data, data_end, &off, &proto) < 0)
        return TC_ACT_OK;

    if (proto != ETH_P_IP)
        return TC_ACT_OK;

    struct iphdr* ip = NULL;
    if (parse_ipv4_tcp(data, data_end, &off, &ip) < 0)
        return TC_ACT_OK;

    struct tcphdr* tcp = NULL;
    int opt_off = 0, opt_len = 0;

    if (parse_tcp_hdr(data, data_end, off, &tcp, &opt_off, &opt_len) < 0)
        return TC_ACT_OK;

    if (!is_first_syn(tcp))
        return TC_ACT_OK;

    __be16 old_window = tcp->window;
    __be16 new_window = bpf_htons(65535);

    if (old_window == new_window)
        return TCX_NEXT;

    (void)bpf_l4_csum_replace(skb, off + offsetof(struct tcphdr, check),
        old_window, new_window, sizeof(new_window));
    (void)bpf_skb_store_bytes(skb, off + offsetof(struct tcphdr, window),
        &new_window, sizeof(new_window), 0);

    return TCX_NEXT;
}

SEC("tc/egress")
int set_ip_id_zero(struct __sk_buff* skb)
{
    void* data = (void*)(long)skb->data;
    void* data_end = (void*)(long)skb->data_end;

    __u32 off = 0;
    __u16 proto = 0;

    if (parse_l2(data, data_end, &off, &proto) < 0)
        return TC_ACT_OK;

    if (proto != ETH_P_IP)
        return TC_ACT_OK;

    __u8* cursor = data;
    struct iphdr* ip = (struct iphdr*)(cursor + off);
    if ((void*)(ip + 1) > data_end)
        return TC_ACT_OK;

    if (ip->version != 4)
        return TC_ACT_OK;

    int ip_hlen = ip->ihl * 4;
    if (ip_hlen < (int)sizeof(*ip))
        return TC_ACT_OK;

    if ((void*)(cursor + off + ip_hlen) > data_end)
        return TC_ACT_OK;

    if (ip->id == 0)
        return TCX_NEXT;

    __u16 old_id = ip->id;
    __u16 new_id = 0;

    // Update IP checksum to account for the id field change, then zero the id
    bpf_l3_csum_replace(skb, off + offsetof(struct iphdr, check), old_id, new_id, 2);
    (void)bpf_skb_store_bytes(skb, off + offsetof(struct iphdr, id),
        &new_id, sizeof(new_id), BPF_F_RECOMPUTE_CSUM);

    return TCX_NEXT;
}

SEC("tc/egress")
int set_ip_ttl(struct __sk_buff* skb)
{
    void* data = (void*)(long)skb->data;
    void* data_end = (void*)(long)skb->data_end;

    __u32 off = 0;
    __u16 proto = 0;

    if (parse_l2(data, data_end, &off, &proto) < 0)
        return TC_ACT_OK;

    if (proto != ETH_P_IP)
        return TC_ACT_OK;

    __u8* cursor = data;
    struct iphdr* ip = (struct iphdr*)(cursor + off);
    if ((void*)(ip + 1) > data_end)
        return TC_ACT_OK;

    if (ip->version != 4)
        return TC_ACT_OK;

    int ip_hlen = ip->ihl * 4;
    if (ip_hlen < (int)sizeof(*ip))
        return TC_ACT_OK;

    if ((void*)(cursor + off + ip_hlen) > data_end)
        return TC_ACT_OK;

    if (ip->ttl == IP_TTL_DEFAULT)
        return TCX_NEXT;

    __u8 new_ttl = IP_TTL_DEFAULT;
    __u16 old_ttl = bpf_htons((__u16)ip->ttl << 8);
    __u16 new_ttl_word = bpf_htons((__u16)new_ttl << 8);

    bpf_l3_csum_replace(skb, off + offsetof(struct iphdr, check),
        old_ttl, new_ttl_word, sizeof(new_ttl_word));
    (void)bpf_skb_store_bytes(skb, off + offsetof(struct iphdr, ttl),
        &new_ttl, sizeof(new_ttl), BPF_F_RECOMPUTE_CSUM);

    return TCX_NEXT;
}

char LICENSE[] SEC("license") = "GPL";

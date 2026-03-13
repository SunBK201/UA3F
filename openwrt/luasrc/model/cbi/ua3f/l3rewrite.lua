local M = {}

local cbi = require("luci.cbi")
local i18n = require("luci.i18n")
local utils = require("luci.model.cbi.ua3f.utils")
local translate = i18n.translate

local Flag = cbi.Flag
local DummyValue = cbi.DummyValue

function M.add_l3rewrite_fields(section)
    -- TTL Setting
    local ttl = section:taboption("l3rewrite", Flag, "l3_rewrite_ttl", translate("Set TTL"))
    ttl.description = translate("Set the TTL 64 for packets")

    -- TCP Timestamp Deletion
    local tcpts = section:taboption("l3rewrite", Flag, "l3_rewrite_tcpts", translate("Delete TCP Timestamps"))
    tcpts.description = translate("Remove TCP Timestamp option")

    -- TCP Initial Window
    local tcp_init_window = section:taboption("l3rewrite", Flag, "l3_rewrite_tcpwin", translate("Set TCP Initial Window"))
    tcp_init_window.description = translate("Set the TCP Initial Window to 65535 for SYN packets")

    -- IP ID Setting
    local ipid = section:taboption("l3rewrite", Flag, "l3_rewrite_ipid", translate("Set IP ID"))
    ipid.description = translate("Set the IP ID to 0 for packets")

    -- BPF Offloading
    local bpf_offload = section:taboption("l3rewrite", Flag, "l3_rewrite_bpf_offload", translate("L3 Rewriting eBPF Offloading"))
    bpf_offload.description = translate("Speed up L3 rewriting by enabling tc egress offloading (requires kernel support)")
    bpf_offload:depends("l3_rewrite_ttl", 1)
    bpf_offload:depends("l3_rewrite_ipid", 1)
    bpf_offload:depends("l3_rewrite_tcpts", 1)
    bpf_offload:depends("l3_rewrite_tcpwin", 1)

    if not utils.nfqueue_exists() then
        local nfqueue_warning = section:taboption("l3rewrite", DummyValue, "_l3rewrite_nfqueue_warning", " ")
        nfqueue_warning.rawhtml = true
        nfqueue_warning:depends("l3_rewrite_tcpts", 1)
        nfqueue_warning:depends("l3_rewrite_tcpwin", 1)
        nfqueue_warning:depends("l3_rewrite_ipid", 1)
        function nfqueue_warning.cfgvalue(self, section)
            return "<strong style='color:red;'>" ..
                translate("Recommend install kmod-nft-queue package for compatibility") .. "</strong>"
        end
    end


    if not utils.sched_bpf_exists() then
        local sched_bpf_warning = section:taboption("l3rewrite", DummyValue, "_l3rewrite_sched_bpf_warning", " ")
        sched_bpf_warning.rawhtml = true
        sched_bpf_warning:depends("l3_rewrite_bpf_offload", 1)
        function sched_bpf_warning.cfgvalue(self, section)
            return "<strong style='color:red;'>" ..
                translate("eBFP offloading requires kmod-sched-bpf package") .. "</strong>"
        end
    end
end

return M

local M = {}

local cbi = require("luci.cbi")
local i18n = require("luci.i18n")
local utils = require("luci.model.cbi.ua3f.utils")
local translate = i18n.translate

local Flag = cbi.Flag
local DummyValue = cbi.DummyValue

function M.add_others_fields(section)
    -- TTL Setting
    local ttl = section:taboption("others", Flag, "set_ttl", translate("Set TTL"))
    ttl.description = translate("Set the TTL 64 for packets")

    -- TCP Timestamp Deletion
    local tcpts = section:taboption("others", Flag, "del_tcpts", translate("Delete TCP Timestamps"))
    tcpts.description = translate("Remove TCP Timestamp option")

    -- TCP Initial Window
    local tcp_init_window = section:taboption("others", Flag, "set_tcp_init_window", translate("Set TCP Initial Window"))
    tcp_init_window.description = translate("Set the TCP Initial Window to 65535 for SYN packets")

    -- IP ID Setting
    local ipid = section:taboption("others", Flag, "set_ipid", translate("Set IP ID"))
    ipid.description = translate("Set the IP ID to 0 for packets")

    if not utils.nfqueue_exists() then
        local nfqueue_warning = section:taboption("others", DummyValue, "_others_nfqueue_warning", " ")
        nfqueue_warning.rawhtml = true
        nfqueue_warning:depends("del_tcpts", 1)
        nfqueue_warning:depends("set_tcp_init_window", 1)
        nfqueue_warning:depends("set_ipid", 1)
        function nfqueue_warning.cfgvalue(self, section)
            return "<strong style='color:red;'>" ..
                translate("Recommend install kmod-nft-queue package for compatibility") .. "</strong>"
        end
    end
end

return M

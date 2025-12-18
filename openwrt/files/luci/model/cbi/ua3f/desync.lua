local M = {}

local cbi = require("luci.cbi")
local i18n = require("luci.i18n")
local utils = require("luci.model.cbi.ua3f.utils")
local translate = i18n.translate

local Flag = cbi.Flag
local Value = cbi.Value
local DummyValue = cbi.DummyValue

function M.add_desync_fields(section)
    -- Enable TCP Desync
    local desync_reorder = section:taboption("desync", Flag, "desync_reorder", translate("Enable TCP Desync Reordering"))
    desync_reorder.description = translate("Enable TCP Reordering to resist DPI")

    if not utils.nfqueue_exists() then
        local nfqueue_warning = section:taboption("desync", DummyValue, "_desync_reorder_nfqueue_warning", " ")
        nfqueue_warning.rawhtml = true
        nfqueue_warning:depends("desync_reorder", 1)
        function nfqueue_warning.cfgvalue(self, section)
            return "<strong style='color:red;'>" ..
                translate("Recommend install kmod-nft-queue package for NFQUEUE mode") .. "</strong>"
        end
    end

    if utils.offloading_enabled() then
        local offloading_warning = section:taboption("desync", DummyValue, "_desync_reorder_offloading_warning", " ")
        offloading_warning.rawhtml = true
        offloading_warning:depends("desync_reorder", 1)
        function offloading_warning.cfgvalue(self, section)
            return "<strong style='color:red;'>" ..
                translate(
                    "Flow Offloading is enabled in firewall settings, it may cause TCP Desync to not work properly") ..
                "</strong>"
        end
    end

    -- CT Byte Setting
    local ct_byte = section:taboption("desync", Value, "desync_reorder_bytes", translate("Desync Reorder Bytes"))
    ct_byte.placeholder = "1500"
    ct_byte.datatype = "uinteger"
    ct_byte.description = translate("Number of bytes for fragmented random emission")
    ct_byte:depends("desync_reorder", "1")

    -- CT Packets Setting
    local ct_packets = section:taboption("desync", Value, "desync_reorder_packets", translate("Desync Reorder Packets"))
    ct_packets.placeholder = "8"
    ct_packets.datatype = "uinteger"
    ct_packets.description = translate("Number of packets for fragmented random emission")
    ct_packets:depends("desync_reorder", "1")

    local desync_inject = section:taboption("desync", Flag, "desync_inject", translate("Enable TCP Desync Injection"))
    desync_inject.description = translate("Enable TCP Injection to resist DPI")

    if not utils.nfqueue_exists() then
        local nfqueue_warning = section:taboption("desync", DummyValue, "_desync_inject_nfqueue_warning", " ")
        nfqueue_warning.rawhtml = true
        nfqueue_warning:depends("desync_inject", 1)
        function nfqueue_warning.cfgvalue(self, section)
            return "<strong style='color:red;'>" ..
                translate("Recommend install kmod-nft-queue package for NFQUEUE mode") .. "</strong>"
        end
    end

    if utils.offloading_enabled() then
        local offloading_warning = section:taboption("desync", DummyValue, "_desync_inject_offloading_warning", " ")
        offloading_warning.rawhtml = true
        offloading_warning:depends("desync_inject", 1)
        function offloading_warning.cfgvalue(self, section)
            return "<strong style='color:red;'>" ..
                translate(
                    "Flow Offloading is enabled in firewall settings, it may cause TCP Desync to not work properly") ..
                "</strong>"
        end
    end

    local inject_ttl = section:taboption("desync", Value, "desync_inject_ttl", translate("Desync Inject TTL"))
    inject_ttl.placeholder = "3"
    inject_ttl.datatype = "uinteger"
    inject_ttl.description = translate("TTL value for injected packets")
    inject_ttl:depends("desync_inject", "1")
end

return M

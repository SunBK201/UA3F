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
    local desync_enabled = section:taboption("desync", Flag, "desync_enabled", translate("Enable TCP Desync"))
    desync_enabled.description = translate("Enable TCP Desynchronization to evade DPI")

    if not utils.nfqueue_exists() then
        local nfqueue_warning = section:taboption("desync", DummyValue, "_desync_nfqueue_warning", " ")
        nfqueue_warning.rawhtml = true
        nfqueue_warning:depends("desync_enabled", 1)
        function nfqueue_warning.cfgvalue(self, section)
            return "<strong style='color:red;'>" ..
                translate("Recommend install kmod-nft-queue package for NFQUEUE mode") .. "</strong>"
        end
    end

    if utils.offloading_enabled() then
        local offloading_warning = section:taboption("desync", DummyValue, "_desync_offloading_warning", " ")
        offloading_warning.rawhtml = true
        offloading_warning:depends("desync_enabled", 1)
        function offloading_warning.cfgvalue(self, section)
            return "<strong style='color:red;'>" ..
                translate(
                    "Flow Offloading is enabled in firewall settings, it may cause TCP Desync to not work properly") ..
                "</strong>"
        end
    end

    -- CT Byte Setting
    local ct_byte = section:taboption("desync", Value, "desync_ct_bytes", translate("Desync Bytes"))
    ct_byte.placeholder = "1500"
    ct_byte.datatype = "uinteger"
    ct_byte.description = translate("Number of bytes for fragmented random emission")
    ct_byte:depends("desync_enabled", "1")

    -- CT Packets Setting
    local ct_packets = section:taboption("desync", Value, "desync_ct_packets", translate("Desync Packets"))
    ct_packets.placeholder = "8"
    ct_packets.datatype = "uinteger"
    ct_packets.description = translate("Number of packets for fragmented random emission")
    ct_packets:depends("desync_enabled", "1")
end

return M

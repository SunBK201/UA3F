local M = {}

local cbi = require("luci.cbi")
local i18n = require("luci.i18n")
local utils = require("luci.model.cbi.ua3f.utils")
local translate = i18n.translate

local Flag = cbi.Flag
local Value = cbi.Value
local ListValue = cbi.ListValue
local DummyValue = cbi.DummyValue

function M.add_general_fields(section)
    -- Server Mode
    local server_mode = section:taboption("general", ListValue, "server_mode", translate("Server Mode"))
    server_mode:value("HTTP", "HTTP")
    server_mode:value("SOCKS5", "SOCKS5")
    server_mode:value("TPROXY", "TPROXY")
    server_mode:value("REDIRECT", "REDIRECT")
    server_mode:value("NFQUEUE", "NFQUEUE")
    server_mode.default = "TPROXY"

    if not utils.tproxy_exists() then
        local tproxy_warning = section:taboption("general", DummyValue, "_tproxy_warning", " ")
        tproxy_warning.rawhtml = true
        tproxy_warning:depends("server_mode", "TPROXY")
        function tproxy_warning.cfgvalue(self, section)
            return "<strong style='color:red;'>" ..
                translate("Recommend install kmod-nft-tproxy package for TPROXY mode") .. "</strong>"
        end
    end

    if not utils.nfqueue_exists() then
        local nfqueue_warning = section:taboption("general", DummyValue, "_nfqueue_warning", " ")
        nfqueue_warning.rawhtml = true
        nfqueue_warning:depends("server_mode", "NFQUEUE")
        function nfqueue_warning.cfgvalue(self, section)
            return "<strong style='color:red;'>" ..
                translate("Recommend install kmod-nft-queue package for NFQUEUE mode") .. "</strong>"
        end
    end

    -- Bind Address
    local bind = section:taboption("general", Value, "bind", translate("Bind Address"))
    bind:value("127.0.0.1")
    bind:value("0.0.0.0")
    bind:depends("server_mode", "HTTP")
    bind:depends("server_mode", "SOCKS5")

    -- Port
    local port = section:taboption("general", Value, "port", translate("Port"))
    port.placeholder = "1080"
    port:depends("server_mode", "HTTP")
    port:depends("server_mode", "SOCKS5")
    port:depends("server_mode", "TPROXY")
    port:depends("server_mode", "REDIRECT")

    -- Rewrite Mode
    local rewrite_mode = section:taboption("general", ListValue, "rewrite_mode", translate("Rewrite Mode"))
    rewrite_mode:value("DIRECT", translate("Direct Forward"))
    rewrite_mode:value("GLOBAL", translate("Global Rewrite"))
    rewrite_mode:value("RULES", translate("Rule Based"))
    rewrite_mode.default = "GLOBAL"
    rewrite_mode.description = translate(
        "Direct Forward: No rewriting. Global Rewrite: Rewrite all User-Agents. Rule Based: Use rewrite rules to determine behavior.")

    -- User-Agent (for Global Rewrite)
    local ua = section:taboption("general", Value, "ua", translate("User-Agent"))
    ua.placeholder = "FFF"
    ua.description = translate("User-Agent after rewrite")
    ua:depends("rewrite_mode", "GLOBAL")
    ua:depends("server_mode", "NFQUEUE")

    -- User-Agent Regex
    local regex = section:taboption("general", Value, "ua_regex", translate("User-Agent Regex"))
    regex.description = translate("Regular expression pattern for matching User-Agent")
    regex:depends("rewrite_mode", "GLOBAL")
    regex:depends("server_mode", "NFQUEUE")

    -- Partial Replace
    local partialReplace = section:taboption("general", Flag, "partial_replace", translate("Partial Replace"))
    partialReplace.description =
        translate(
            "Replace only the matched part of the User-Agent, only works when User-Agent Regex is not empty")
    partialReplace.default = "0"
    partialReplace:depends("rewrite_mode", "GLOBAL")
    partialReplace:depends("server_mode", "NFQUEUE")
end

return M

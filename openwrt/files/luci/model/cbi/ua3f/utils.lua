local M = {}

local sys = require("luci.sys")

local function cmd_exists(cmd)
    return sys.call("command -v " .. cmd .. " >/dev/null 2>&1") == 0
end

function M.nfqueue_exists()
    local opkg = cmd_exists("opkg") and sys.call("opkg list-installed kmod-nft-queue | grep -q kmod-nft-queue") == 0
    local apk = cmd_exists("apk") and (sys.call("apk info | grep -q kmod-nft-queue") == 0)
    return opkg or apk
end

function M.tproxy_exists()
    local opkg = cmd_exists("opkg") and sys.call("opkg list-installed kmod-nft-tproxy | grep -q kmod-nft-tproxy") == 0
    local apk = cmd_exists("apk") and (sys.call("apk info | grep -q kmod-nft-tproxy") == 0)
    return opkg or apk
end

function M.offloading_enabled()
    -- uci get firewall.@defaults[0].flow_offloading
    local uci = require("luci.model.uci").cursor()
    local flow_offloading = uci:get("firewall", "@defaults[0]", "flow_offloading")
    return flow_offloading == "1"
end

return M

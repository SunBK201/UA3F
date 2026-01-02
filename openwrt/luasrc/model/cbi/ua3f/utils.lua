local M = {}

local sys = require("luci.sys")

local function cmd_exists(cmd)
    return sys.call("command -v " .. cmd .. " >/dev/null 2>&1") == 0
end

local function opkg_installed(pkg)
    if not cmd_exists("opkg") then return false end
    local output = sys.exec("opkg list-installed " .. pkg .. " 2>&1")
    if output:find(pkg, 1, true) then
        return true
    end
    if output:find("Could not lock /var/lock/opkg.lock", 1, true) then
        return true
    end
    return false
end

local function apk_installed(pkg)
    if not cmd_exists("apk") then return false end
    local output = sys.exec("apk info | grep " .. pkg .. " 2>&1")
    return output:find(pkg, 1, true) ~= nil
end

function M.nfqueue_exists()
    return opkg_installed("kmod-nft-queue") or apk_installed("kmod-nft-queue")
end

function M.tproxy_exists()
    return opkg_installed("kmod-nft-tproxy") or apk_installed("kmod-nft-tproxy")
end

function M.offloading_enabled()
    -- uci get firewall.@defaults[0].flow_offloading
    local uci = require("luci.model.uci").cursor()
    local flow_offloading = uci:get("firewall", "@defaults[0]", "flow_offloading")
    return flow_offloading == "1"
end

return M

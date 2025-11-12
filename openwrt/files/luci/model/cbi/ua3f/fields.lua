local M = {}

local cbi = require("luci.cbi")
local i18n = require("luci.i18n")
local sys = require("luci.sys")
local translate = i18n.translate

local Flag = cbi.Flag
local Value = cbi.Value
local ListValue = cbi.ListValue
local DummyValue = cbi.DummyValue
local TextValue = cbi.TextValue
local Button = cbi.Button

-- Status Section Fields
function M.add_status_fields(section)
    -- Enabled Flag
    section:option(Flag, "enabled", translate("Enabled"))

    -- Running Status Display
    local running = section:option(DummyValue, "running", translate("Status"))
    running.rawhtml = true
    running.cfgvalue = function(self, section)
        local pid = sys.exec("pidof ua3f")
        if pid == "" then
            return "<input disabled type='button' style='opacity: 1;' class='btn cbi-button cbi-button-reset' value='" ..
                translate("Stop") .. "'/>"
        else
            return "<input disabled type='button' style='opacity: 1;' class='btn cbi-button cbi-button-add' value='" ..
                translate("Running") .. "'/>"
        end
    end
end

-- General Tab Fields
function M.add_general_fields(section)
    -- Server Mode
    local server_mode = section:taboption("general", ListValue, "server_mode", translate("Server Mode"))
    server_mode:value("HTTP", "HTTP")
    server_mode:value("SOCKS5", "SOCKS5")
    server_mode:value("TPROXY", "TPROXY")
    server_mode:value("REDIRECT", "REDIRECT")
    server_mode:value("NFQUEUE", "NFQUEUE")
    server_mode.default = "SOCKS5"

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
        "Direct Forward: No rewriting. Global Rewrite: Rewrite all User-Agents to the specified value. Rule Based: Use rewrite rules to determine behavior.")

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

-- Rewrite Rules Tab Fields
function M.add_rewrite_fields(section)
    local rules = section:taboption("rewrite", DummyValue, "")
    rules.template = "ua3f/rules"
end

-- Statistics Tab Fields
function M.add_stats_fields(section)
    local stats = section:taboption("stats", DummyValue, "")
    stats.template = "ua3f/statistics"
end

-- Log Tab Fields
function M.add_log_fields(section)
    -- Log Display
    local log = section:taboption("log", TextValue, "log")
    log.readonly = true
    log.rows = 30
    function log.cfgvalue(self, section)
        local logfile = "/var/log/ua3f/ua3f.log"
        local fs = require("nixio.fs")
        if not fs.access(logfile) then
            return ""
        end
        local n = tonumber(luci.model.uci.cursor():get("ua3f", section, "log_lines")) or 1000
        return luci.sys.exec("tail -n " .. n .. " " .. logfile)
    end

    function log.write(self, section, value) end

    function log.render(self, section, scope)
        TextValue.render(self, section, scope)
        luci.http.write("<script>")
        luci.http.write([[
            var textarea = document.getElementById('cbid.ua3f.main.log');
            if (textarea) {
                textarea.scrollTop = textarea.scrollHeight;
            }
        ]])
        luci.http.write("</script>")
    end

    -- Log Level
    local log_level = section:taboption("log", ListValue, "log_level", translate("Log Level"))
    log_level:value("debug")
    log_level:value("info")
    log_level:value("warn")
    log_level:value("error")
    log_level:value("fatal")
    log_level:value("panic")
    log_level.description = translate(
        "Sets the logging level. Do not keep the log level set to debug/info/warn for an extended period of time.")

    -- Log Lines
    local logLines = section:taboption("log", Value, "log_lines", translate("Display Lines"))
    logLines.default = "1000"
    logLines.datatype = "uinteger"
    logLines.rmempty = false

    -- Clear Log Button
    local clearlog = section:taboption("log", Button, "_clearlog", translate("Clear Logs"))
    clearlog.inputtitle = translate("Clear Logs")
    clearlog.inputstyle = "reset"
    function clearlog.write(self, section)
    end

    function clearlog.render(self, section, scope)
        Button.render(self, section, scope)
        luci.http.write([[
            <script>
            document.querySelector("input[name='cbid.ua3f.main._clearlog']").addEventListener("click", function(e) {
                e.preventDefault();
                fetch(']] .. luci.dispatcher.build_url("admin/services/ua3f/clear_log") .. [[', {method: 'POST'})
                .then(resp => {
                    if (resp.ok) {
                        var textarea = document.getElementById('cbid.ua3f.main.log');
                        if (textarea) textarea.value = "";
                    }
                });
            });
            </script>
        ]])
    end

    -- Download Log Button
    local download = section:taboption("log", Button, "_download", translate("Download Logs"))
    download.inputtitle = translate("Download Logs")
    download.inputstyle = "apply"
    function download.write(self, section)
        luci.http.redirect(luci.dispatcher.build_url("admin/services/ua3f/download_log"))
    end

    -- Issue Report Button
    local issue = section:taboption("log", Button, "_issue", translate("Issue Report"))
    issue.inputtitle = translate("Issue Report")
    issue.inputstyle = "save"
    function issue.write(self, section)
    end

    function issue.render(self, section, scope)
        Button.render(self, section, scope)
        luci.http.write([[
            <script>
            document.querySelector("input[name='cbid.ua3f.main._issue']").addEventListener("click", function(e) {
                e.preventDefault();
                window.open('https://github.com/SunBK201/UA3F/issues/new?template=bug-report.md', '_blank');
            });
            </script>
        ]])
    end

    return log
end

-- Others Tab Fields
function M.add_others_fields(section)
    -- TTL Setting
    local ttl = section:taboption("others", Flag, "set_ttl", translate("Set TTL"))
    ttl.description = translate("Set the TTL 64 for packets")

    -- TCP Timestamp Deletion
    local tcpts = section:taboption("others", Flag, "del_tcpts", translate("Delete TCP Timestamps"))
    tcpts.description = translate("Remove TCP Timestamp option")

    -- IP ID Setting
    local ipid = section:taboption("others", Flag, "set_ipid", translate("Set IP ID"))
    ipid.description = translate("Set the IP ID to 0 for packets")
end

return M

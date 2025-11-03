local uci = require("luci.model.uci").cursor()

ua3f = Map("ua3f",
    "UA3F",
    [[
        <a href="https://github.com/SunBK201/UA3F" target="_blank">Version: 1.1.0</a>
        <br>
        Across the Campus we can reach every corner in the world.
    ]]
)

status = ua3f:section(NamedSection, "enabled", "ua3f", translate("Status"))
general = ua3f:section(NamedSection, "main", "ua3f", translate("General"))

status:option(Flag, "enabled", translate("Enabled"))

running = status:option(DummyValue, "running", translate("Status"))
running.rawhtml = true
running.cfgvalue = function(self, section)
    local pid = luci.sys.exec("pidof ua3f")
    if pid == "" then
        return "<input disabled type='button' style='opacity: 1;' class='btn cbi-button cbi-button-reset' value='" ..
            translate("Stop") .. "'/>"
    else
        return "<input disabled type='button' style='opacity: 1;' class='btn cbi-button cbi-button-add' value='" ..
            translate("Running") .. "'/>"
    end
end

general:tab("general", translate("Settings"))
general:tab("stats", translate("Statistics"))
general:tab("log", translate("Log"))

server_mode = general:taboption("general", ListValue, "server_mode", translate("Server Mode"))
server_mode:value("SOCKS5", "SOCKS5")
server_mode:value("TPROXY", "TPROXY")
server_mode:value("REDIRECT", "REDIRECT")

port = general:taboption("general", Value, "port", translate("Port"))
port.placeholder = "1080"

bind = general:taboption("general", Value, "bind", translate("Bind Address"))
bind:value("127.0.0.1")
bind:value("0.0.0.0")

log_level = general:taboption("general", ListValue, "log_level", translate("Log Level"))
log_level:value("debug")
log_level:value("info")
log_level:value("warn")
log_level:value("error")
log_level:value("fatal")
log_level:value("panic")
log_level.description = translate(
    "Sets the logging level. Do not keep the log level set to debug/info/warn for an extended period of time.")

ua = general:taboption("general", Value, "ua", translate("User-Agent"))
ua.placeholder = "FFF"
ua.description = translate("User-Agent to be rewritten")

uaRegexPattern = general:taboption("general", Value, "ua_regex", translate("User-Agent Regex Pattern"))
uaRegexPattern.description = translate("Regular expression pattern for matching User-Agent")

partialRepalce = general:taboption("general", Flag, "partial_replace", translate("Partial Replace"))
partialRepalce.description =
    translate("Replace only the matched part of the User-Agent, only works when User-Agent Regex Pattern is not empty")
partialRepalce.default = "0"

log = general:taboption("log", TextValue, "log")
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

logLines = general:taboption("log", Value, "log_lines", translate("Display Lines"))
logLines.default = "1000"
logLines.datatype = "uinteger"
logLines.rmempty = false

download = general:taboption("log", Button, "_download", translate("Download Logs"))
download.inputtitle = translate("Download Logs")
download.inputstyle = "apply"
function download.write(self, section)
    luci.http.redirect(luci.dispatcher.build_url("admin/services/ua3f/download_log"))
end

stats = general:taboption("stats", DummyValue, "")
stats.template = "ua3f/statistics"

return ua3f

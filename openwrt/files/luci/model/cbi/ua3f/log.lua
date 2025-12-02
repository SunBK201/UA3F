local M = {}

local cbi = require("luci.cbi")
local i18n = require("luci.i18n")
local translate = i18n.translate

local Value = cbi.Value
local ListValue = cbi.ListValue
local DummyValue = cbi.DummyValue
local TextValue = cbi.TextValue

function M.add_log_fields(section)
    -- Log Display
    local log = section:taboption("log", TextValue, "log")
    log.readonly = true
    log.rows = 30
    log.wrap = "off"
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
                textarea.removeAttribute('name');
                textarea.scrollTop = textarea.scrollHeight;
            }
        ]])
        luci.http.write("</script>")
    end

    -- Log Level
    local log_level = section:taboption("log", ListValue, "log_level", translate("Log Level"))
    log_level:value("DEBUG")
    log_level:value("INFO")
    log_level:value("WARN")
    log_level:value("ERROR")
    log_level.default = "WARN"
    log_level.description = translate(
        "Sets the logging level. Do not keep the log level set to DEBUG for an extended period of time.")

    -- Log Lines
    local logLines = section:taboption("log", Value, "log_lines", translate("Display Lines"))
    logLines.default = "1000"
    logLines.datatype = "uinteger"
    logLines.rmempty = false

    -- Button Container (DummyValue to hold all buttons)
    local button_container = section:taboption("log", DummyValue, "_button_container", translate("Log Actions"))
    button_container.rawhtml = true

    function button_container.cfgvalue(self, section)
        return ""
    end

    function button_container.render(self, section, scope)
        luci.http.write([[
            <div class="cbi-value" id="cbi-ua3f-main-_button_container">
                <label class="cbi-value-title">]] .. translate("Log Management") .. [[</label>
                <div class="cbi-value-field" style="display: flex; gap: 10px; flex-wrap: wrap;">
                    <input type="button" class="btn cbi-button cbi-button-reset"
                           value="]] .. translate("Clear Logs") .. [[" id="ua3f-clearlog-btn"/>
                    <input type="button" class="btn cbi-button cbi-button-apply"
                           value="]] .. translate("Download Logs") .. [[" id="ua3f-download-btn"/>
                    <input type="button" class="btn cbi-button cbi-button-save"
                           value="]] .. translate("Issue Report") .. [[" id="ua3f-issue-btn"/>
                </div>
            </div>
            <script>
            (function() {
                // Clear Log Button
                document.getElementById('ua3f-clearlog-btn').addEventListener('click', function(e) {
                    e.preventDefault();
                    fetch(']] .. luci.dispatcher.build_url("admin/services/ua3f/clear_log") .. [[', {method: 'POST'})
                    .then(resp => {
                        if (resp.ok) {
                            var textarea = document.getElementById('cbid.ua3f.main.log');
                            if (textarea) textarea.value = "";
                        }
                    });
                });

                // Download Log Button
                document.getElementById('ua3f-download-btn').addEventListener('click', function(e) {
                    e.preventDefault();
                    window.location.href = ']] .. luci.dispatcher.build_url("admin/services/ua3f/download_log") .. [[';
                });

                // Issue Report Button
                document.getElementById('ua3f-issue-btn').addEventListener('click', function(e) {
                    e.preventDefault();
                    window.open('https://github.com/SunBK201/UA3F/issues/new?template=bug-report.md', '_blank');
                });
            })();
            </script>
        ]])
    end

    return log
end

return M

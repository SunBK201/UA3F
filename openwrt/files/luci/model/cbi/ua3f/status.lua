local M = {}

local cbi = require("luci.cbi")
local i18n = require("luci.i18n")
local sys = require("luci.sys")
local translate = i18n.translate

local Flag = cbi.Flag
local DummyValue = cbi.DummyValue

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

return M

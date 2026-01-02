local M = {}

local cbi = require("luci.cbi")
local i18n = require("luci.i18n")
local translate = i18n.translate

local Flag = cbi.Flag
local DummyValue = cbi.DummyValue

function M.add_status_fields(section)
    -- Enabled Flag
    section:option(Flag, "enabled", translate("Enabled"))

    -- Running Status Display
    local running = section:option(DummyValue, "running", translate("Status"))
    running.template = "ua3f/status"
end

return M

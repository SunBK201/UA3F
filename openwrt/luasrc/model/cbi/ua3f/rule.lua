local M = {}

local cbi = require("luci.cbi")

local DummyValue = cbi.DummyValue

function M.add_rule_fields(section)
    local rules = section:taboption("rules", DummyValue, "")
    rules.template = "ua3f/rules"
end

return M

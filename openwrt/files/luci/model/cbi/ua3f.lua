local cbi = require("luci.cbi")
local i18n = require("luci.i18n")
local translate = i18n.translate
local NamedSection = cbi.NamedSection

local ua3f = cbi.Map("ua3f",
    "UA3F",
    [[
        <a href="https://github.com/SunBK201/UA3F" target="_blank">Version: 2.0.0</a>
        <br>
        Across the Campus we can reach every corner in the world.
    ]]
)
local status = require("luci.model.cbi.ua3f.status")
local general = require("luci.model.cbi.ua3f.general")
local rule = require("luci.model.cbi.ua3f.rule")
local desync = require("luci.model.cbi.ua3f.desync")
local others = require("luci.model.cbi.ua3f.others")
local statistics = require("luci.model.cbi.ua3f.statistics")
local log = require("luci.model.cbi.ua3f.log")

function create_sections(map)
    local sections = {}

    -- Status Section
    sections.status = map:section(NamedSection, "enabled", "ua3f", translate("Status"))

    -- General Section with tabs
    sections.general = map:section(NamedSection, "main", "ua3f", translate("General"))
    sections.general:tab("general", translate("Settings"))
    sections.general:tab("rules", translate("Rewrite Rules"))
    sections.general:tab("desync", translate("Desync Settings"))
    sections.general:tab("others", translate("Others Settings"))
    sections.general:tab("statistics", translate("Statistics"))
    sections.general:tab("log", translate("Log"))

    return sections
end

local sections = create_sections(ua3f)

status.add_status_fields(sections.status)
general.add_general_fields(sections.general)
rule.add_rule_fields(sections.general)
desync.add_desync_fields(sections.general)
others.add_others_fields(sections.general)
statistics.add_statistics_fields(sections.general)
log.add_log_fields(sections.general)

return ua3f

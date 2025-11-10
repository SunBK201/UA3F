local cbi = require("luci.cbi")
local i18n = require("luci.i18n")
local translate = i18n.translate
local NamedSection = cbi.NamedSection

local ua3f = cbi.Map("ua3f",
    "UA3F",
    [[
        <a href="https://github.com/SunBK201/UA3F" target="_blank">Version: 1.7.0</a>
        <br>
        Across the Campus we can reach every corner in the world.
    ]]
)
local fields = require("luci.model.cbi.ua3f.fields")

function create_sections(map)
    local sections = {}

    -- Status Section
    sections.status = map:section(NamedSection, "enabled", "ua3f", translate("Status"))

    -- General Section with tabs
    sections.general = map:section(NamedSection, "main", "ua3f", translate("General"))
    sections.general:tab("general", translate("Settings"))
    sections.general:tab("rewrite", translate("Rewrite Rules"))
    sections.general:tab("stats", translate("Statistics"))
    sections.general:tab("log", translate("Log"))
    sections.general:tab("others", translate("Others Settings"))

    return sections
end

local sections = create_sections(ua3f)

fields.add_status_fields(sections.status)
fields.add_general_fields(sections.general)
fields.add_rewrite_fields(sections.general)
fields.add_stats_fields(sections.general)
fields.add_log_fields(sections.general)
fields.add_others_fields(sections.general)

return ua3f

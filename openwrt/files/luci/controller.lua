module("luci.controller.ua3f", package.seeall)

function index()
    entry({"admin", "services", "ua3f"}, cbi("ua3f"), _("UA3F"), 1)
end
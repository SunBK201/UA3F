module("luci.controller.ua3f", package.seeall)

function index()
    entry({ "admin", "services", "ua3f" }, cbi("ua3f"), _("UA3F"), 1)
    entry({ "admin", "services", "ua3f", "download_log" }, call("action_download_log")).leaf = true
    entry({ "admin", "services", "ua3f", "clear_log" }, call("clear_log")).leaf = true
    entry({ "admin", "services", "ua3f", "get_rules" }, call("get_rules")).leaf = true
    entry({ "admin", "services", "ua3f", "save_rules" }, call("save_rules")).leaf = true
end

local fs = require("nixio.fs")

function create_log_archive()
    local tmpfile = "/tmp/ua3f_logs.tar.gz"
    local cmd = "cd /var/log && tar -czf " .. tmpfile .. " ua3f >/dev/null 2>&1"
    os.execute(cmd)
    return tmpfile
end

function send_file_download(filepath, filename)
    local http = require("luci.http")

    if not fs.access(filepath) then
        http.status(500, "Internal Server Error")
        http.prepare_content("text/plain")
        http.write("Failed to create archive")
        return false
    end

    http.header("Content-Type", "application/gzip")
    http.header("Content-Disposition", 'attachment; filename="' .. filename .. '"')
    http.header("Content-Length", tostring(fs.stat(filepath).size))

    local fp = io.open(filepath, "rb")
    if fp then
        while true do
            local chunk = fp:read(2048)
            if not chunk then break end
            http.write(chunk)
        end
        fp:close()
    end

    return true
end

function clear_log_file(logfile)
    if fs.access(logfile) then
        fs.writefile(logfile, "")
        return true
    end
    return false
end

function action_download_log()
    local tmpfile = create_log_archive()
    local success = send_file_download(tmpfile, "ua3f_logs.tar.gz")
    if success then
        fs.remove(tmpfile)
    end
end

function clear_log()
    local logfile = "/var/log/ua3f/ua3f.log"
    local success = clear_log_file(logfile)

    local http = luci.http
    if success then
        http.status(200, "OK")
        http.write("Log cleared")
    else
        http.status(404, "Not Found")
        http.write("Log file not found")
    end
end

function get_rules()
    local http = require("luci.http")
    local uci = require("luci.model.uci").cursor()
    local json = require("luci.jsonc")

    http.prepare_content("application/json")

    local rules = {}
    local rules_data = uci:get("ua3f", "main", "rewrite_rules")

    if rules_data then
        -- Parse the rules from UCI config
        -- Rules are stored as JSON string in UCI
        local success, parsed_rules = pcall(json.parse, rules_data)
        if success and parsed_rules then
            rules = parsed_rules
        end
    end

    http.write(json.stringify({
        success = true,
        rules = rules
    }))
end

function save_rules()
    local http = require("luci.http")
    local uci = require("luci.model.uci").cursor()
    local json = require("luci.jsonc")

    http.prepare_content("application/json")

    -- Read POST data
    local content_length = tonumber(http.getenv("CONTENT_LENGTH"))
    if not content_length or content_length == 0 then
        http.write(json.stringify({
            success = false,
            error = "No data provided"
        }))
        return
    end

    local post_data = http.content()
    if not post_data then
        http.write(json.stringify({
            success = false,
            error = "Failed to read request data"
        }))
        return
    end

    -- Parse JSON data
    local success, data = pcall(json.parse, post_data)
    if not success or not data or not data.rules then
        http.write(json.stringify({
            success = false,
            error = "Invalid JSON data"
        }))
        return
    end

    -- Ensure FINAL rule exists and is at the end
    local has_final = false
    local final_rule_index = nil
    for i, rule in ipairs(data.rules) do
        if rule.type == "FINAL" then
            has_final = true
            final_rule_index = i
            break
        end
    end

    -- If FINAL rule exists but not at the end, move it
    if has_final and final_rule_index ~= #data.rules then
        local final_rule = table.remove(data.rules, final_rule_index)
        table.insert(data.rules, final_rule)
    end

    -- If no FINAL rule, add one
    if not has_final then
        table.insert(data.rules, {
            type = "FINAL",
            match_value = "",
            action = "DIRECT",
            rewrite_value = "",
            description = "Default fallback rule",
            enabled = true
        })
    else
        -- Ensure FINAL rule is always enabled
        for i, rule in ipairs(data.rules) do
            if rule.type == "FINAL" then
                rule.enabled = true
                break
            end
        end
    end

    -- Save rules to UCI
    local rules_json = json.stringify(data.rules)
    uci:set("ua3f", "main", "rewrite_rules", rules_json)

    http.write(json.stringify({
        success = true,
        message = "Rules saved successfully"
    }))
end

module("luci.controller.ua3f", package.seeall)

function index()
    entry({ "admin", "services", "ua3f" }, cbi("ua3f"), _("UA3F"), 1)
    entry({ "admin", "services", "ua3f", "status" }, call("get_status")).leaf = true
    entry({ "admin", "services", "ua3f", "download_log" }, call("action_download_log")).leaf = true
    entry({ "admin", "services", "ua3f", "clear_log" }, call("clear_log")).leaf = true
    entry({ "admin", "services", "ua3f", "get_rules" }, call("get_rules")).leaf = true
    entry({ "admin", "services", "ua3f", "save_header_rules" }, call("save_header_rules")).leaf = true
    entry({ "admin", "services", "ua3f", "save_body_rules" }, call("save_body_rules")).leaf = true
    entry({ "admin", "services", "ua3f", "save_url_redirect_rules" }, call("save_url_redirect_rules")).leaf = true
    entry({ "admin", "services", "ua3f", "mitm_generate_cert" }, call("mitm_generate_cert")).leaf = true
    entry({ "admin", "services", "ua3f", "mitm_export_cert" }, call("mitm_export_cert")).leaf = true
end

local fs = require("nixio.fs")

function get_status()
    local http = require("luci.http")
    local sys = require("luci.sys")
    local json = require("luci.jsonc")

    http.prepare_content("application/json")

    local pid = sys.exec("pidof ua3f")
    local running = (pid ~= nil and pid ~= "")

    http.write(json.stringify({
        running = running,
        pid = running and pid:gsub("%s+", "") or nil
    }))
end

function create_log_archive()
    local tmpfile = "/tmp/ua3f_logs.tar.gz"
    local copyCfg = "cp /etc/config/ua3f /var/log/ua3f/config >/dev/null 2>&1"
    os.execute(copyCfg)
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
    local rules_data = uci:get("ua3f", "main", "rules")

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

function save_header_rules()
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
            action = "DIRECT",
            rewrite_value = "",
            rewrite_header = "User-Agent",
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
    uci:set("ua3f", "main", "header_rewrite", rules_json)

    http.write(json.stringify({
        success = true,
        message = "Rules saved successfully"
    }))
end

function save_body_rules()
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

    -- Save rules to UCI
    local rules_json = json.stringify(data.rules)
    uci:set("ua3f", "main", "body_rewrite", rules_json)

    http.write(json.stringify({
        success = true,
        message = "Body rules saved successfully"
    }))
end

function mitm_generate_cert()
    local http = require("luci.http")
    local uci = require("luci.model.uci").cursor()
    local json = require("luci.jsonc")
    local sys = require("luci.sys")

    http.prepare_content("application/json")

    -- Get passphrase from POST data
    local passphrase = http.formvalue("passphrase") or ""

    -- Call ua3f cert generate to create a new CA
    local cmd = "/usr/bin/ua3f cert generate"
    if passphrase ~= "" then
        cmd = cmd .. " --passphrase '" .. passphrase:gsub("'", "'\\''") .. "'"
    end
    local p12_base64 = sys.exec(cmd .. " 2>/dev/null")
    p12_base64 = p12_base64:gsub("%s+$", "")

    if p12_base64 == "" then
        http.write(json.stringify({
            success = false,
            error = "Failed to generate certificate. Check if ua3f binary exists."
        }))
        return
    end

    -- Save cert and passphrase to UCI
    uci:set("ua3f", "main", "mitm_ca_p12_base64", p12_base64)
    uci:set("ua3f", "main", "mitm_ca_passphrase", passphrase)
    uci:commit("ua3f")

    http.write(json.stringify({
        success = true,
        message = "Certificate generated successfully"
    }))
end

function mitm_export_cert()
    local http = require("luci.http")
    local uci = require("luci.model.uci").cursor()
    local sys = require("luci.sys")

    local p12_base64 = uci:get("ua3f", "main", "mitm_ca_p12_base64") or ""
    local passphrase = uci:get("ua3f", "main", "mitm_ca_passphrase") or ""

    if p12_base64 == "" then
        http.status(404, "Not Found")
        http.prepare_content("text/plain")
        http.write("No CA certificate configured. Please generate one first.")
        return
    end

    -- Call ua3f cert export to get PEM
    local cmd = "/usr/bin/ua3f cert export --p12-base64 '" .. p12_base64 .. "'"
    if passphrase ~= "" then
        cmd = cmd .. " --passphrase '" .. passphrase:gsub("'", "'\\''") .. "'"
    end
    local pem_data = sys.exec(cmd .. " 2>/dev/null")

    if pem_data == "" then
        http.status(500, "Internal Server Error")
        http.prepare_content("text/plain")
        http.write("Failed to export certificate")
        return
    end

    http.header("Content-Type", "application/x-pem-file")
    http.header("Content-Disposition", 'attachment; filename="ua3f-ca.pem"')
    http.header("Content-Length", tostring(#pem_data))
    http.write(pem_data)
end

function save_url_redirect_rules()
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

    -- Save rules to UCI
    local rules_json = json.stringify(data.rules)
    uci:set("ua3f", "main", "url_redirect", rules_json)

    http.write(json.stringify({
        success = true,
        message = "URL redirect rules saved successfully"
    }))
end

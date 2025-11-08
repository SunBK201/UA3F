module("luci.controller.ua3f", package.seeall)

function index()
    entry({ "admin", "services", "ua3f" }, cbi("ua3f"), _("UA3F"), 1)
    entry({ "admin", "services", "ua3f", "download_log" }, call("action_download_log")).leaf = true
    entry({ "admin", "services", "ua3f", "clear_log" }, call("clear_log")).leaf = true
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

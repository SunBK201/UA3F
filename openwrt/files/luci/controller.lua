module("luci.controller.ua3f", package.seeall)

function index()
    entry({ "admin", "services", "ua3f" }, cbi("ua3f"), _("UA3F"), 1)
    entry({ "admin", "services", "ua3f", "download_log" }, call("action_download_log"), _("Download Logs"), 11).leaf = true
end

function action_download_log()
    local nixio = require "nixio"
    local fs = require "nixio.fs"
    local http = require "luci.http"
    local tmpfile = "/tmp/ua3f_logs.tar.gz"

    local cmd = "cd /var/log && tar -czf " .. tmpfile .. " ua3f >/dev/null 2>&1"
    os.execute(cmd)

    if not fs.access(tmpfile) then
        http.status(500, "Internal Server Error")
        http.prepare_content("text/plain")
        http.write("Failed to create archive")
        return
    end

    http.header("Content-Type", "application/gzip")
    http.header("Content-Disposition", 'attachment; filename="ua3f_logs.tar.gz"')
    http.header("Content-Length", tostring(fs.stat(tmpfile).size))

    local fp = io.open(tmpfile, "rb")
    if fp then
        while true do
            local chunk = fp:read(2048)
            if not chunk then break end
            http.write(chunk)
        end
        fp:close()
    end

    nixio.fs.remove(tmpfile)
end

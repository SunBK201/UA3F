-- UA3F Statistics Data Module
local M = {}

-- Read rewrite statistics from file
function M.read_rewrite_stats()
    local stats = {}
    local file = io.open("/var/log/ua3f/rewrite_stats", "r")
    if file then
        for line in file:lines() do
            local host, count, origin_ua, mocked_ua = line:match("^(%S+)%s+(%d+)%s+(.-)SEQSEQ(.-)%s*$")
            if host and count then
                table.insert(stats, {
                    host = host,
                    count = count,
                    origin_ua = origin_ua,
                    mocked_ua = mocked_ua
                })
            end
        end
        file:close()
    end
    return stats
end

-- Read pass-through statistics from file
function M.read_pass_stats()
    local stats = {}
    local file = io.open("/var/log/ua3f/pass_stats", "r")
    if file then
        for line in file:lines() do
            local srcAddr, destAddr, count, ua = line:match("^(%S+)%s(%S+)%s(%d+)%s(.+)$")
            if ua and count then
                table.insert(stats, {
                    ua = ua,
                    count = count,
                    srcAddr = srcAddr,
                    destAddr = destAddr
                })
            end
        end
        file:close()
    end
    return stats
end

-- Read connection statistics from file
function M.read_conn_stats()
    local stats = {}
    local file = io.open("/var/log/ua3f/conn_stats", "r")
    if file then
        for line in file:lines() do
            local protocol, srcAddr, destAddr, duration = line:match("^(%S+)%s(%S+)%s(%S+)%s(.+)$")
            if protocol and srcAddr and destAddr and duration then
                table.insert(stats, {
                    protocol = protocol,
                    srcAddr = srcAddr,
                    destAddr = destAddr,
                    duration = duration
                })
            end
        end
        file:close()
    end
    return stats
end

-- Generate row style class
function M.rowstyle(i)
    return (i % 2 == 0) and "cbi-rowstyle-2" or "cbi-rowstyle-1"
end

return M

if bit32 == nil then
    bit32 = {
        band = load("return function(a, b) return a & b end")(),
        bor = load("return function(a, b) return a | b end")(),
        bxor = load("return function(a, b) return a ~ b end")(),
    }
end

if fs ~= nil then
    return
end

local function quote(path)
    return "'" .. path:gsub("'", "'\\''") .. "'"
end

local function succeeded(ok)
    return ok == true or ok == 0
end

local function run(command)
    return succeeded((os.execute(command)))
end

local function fail(path, message)
    error(path .. ": " .. message, 3)
end

local Handle = {}
Handle.__index = Handle

function Handle:read(count)
    if count == nil then
        return self.file:read(1)
    end
    local data = self.file:read(count)
    return data
end

function Handle:readAll()
    return self.file:read("a")
end

function Handle:readLine(keep)
    local line = self.file:read(keep and "L" or "l")
    return line
end

function Handle:write(data)
    self.file:write(data)
end

function Handle:writeLine(data)
    self.file:write(data, "\n")
end

function Handle:seek(whence, offset)
    return self.file:seek(whence or "cur", offset or 0)
end

function Handle:flush()
    self.file:flush()
end

function Handle:close()
    self.file:close()
end

fs = {}

function fs.exists(path)
    return run("test -e " .. quote(path))
end

function fs.isDir(path)
    return run("test -d " .. quote(path))
end

function fs.open(path, mode)
    if fs.isDir(path) then
        return nil, path .. ": Not a file"
    end
    local file, err = io.open(path, mode)
    if file == nil then
        return nil, err
    end
    return setmetatable({file = file}, Handle)
end

function fs.getSize(path)
    local file = io.open(path, "rb")
    if file == nil then
        fail(path, "No such file")
    end
    local size = file:seek("end")
    file:close()
    return size
end

function fs.list(path)
    if not fs.isDir(path) then
        fail(path, "Not a directory")
    end
    local names = {}
    local pipe = io.popen("ls -A " .. quote(path))
    for name in pipe:lines() do
        names[#names + 1] = name
    end
    pipe:close()
    table.sort(names)
    return names
end

function fs.makeDir(path)
    if fs.exists(path) and not fs.isDir(path) then
        fail(path, "File exists")
    end
    if not run("mkdir -p " .. quote(path)) then
        fail(path, "Access denied")
    end
end

function fs.delete(path)
    if not run("rm -rf " .. quote(path)) then
        fail(path, "Access denied")
    end
end

function fs.move(from, to)
    if not fs.exists(from) then
        fail(from, "No such file")
    end
    if fs.exists(to) then
        fail(to, "File exists")
    end
    if not run("mv " .. quote(from) .. " " .. quote(to)) then
        fail(to, "Access denied")
    end
end

function fs.getDir(path)
    return (path:match("^(.*)/[^/]*$")) or ""
end

function fs.getName(path)
    return (path:match("([^/]*)$"))
end

function fs.combine(base, ...)
    local parts = {base, ...}
    return (table.concat(parts, "/"):gsub("//+", "/"))
end

function fs.attributes(path)
    if not fs.exists(path) then
        fail(path, "No such file")
    end
    local dir = fs.isDir(path)
    return {size = dir and 0 or fs.getSize(path), isDir = dir, isReadOnly = false, created = 0, modified = 0}
end

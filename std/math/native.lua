local __wisp_module = {}

__wisp_module.abs = function(x) return x < 0 and -x or x end
__wisp_module.fabs = math.abs

__wisp_module.min = math.min
__wisp_module.fmin = math.min

__wisp_module.max = math.max
__wisp_module.fmax = math.max

__wisp_module.floor = math.floor
__wisp_module.ceil = math.ceil
__wisp_module.round = function(x) return math.floor(x + 0.5) end

__wisp_module.sqrt = math.sqrt
__wisp_module.pow = function(base, exp) return base ^ exp end

__wisp_module.sin = math.sin
__wisp_module.cos = math.cos
__wisp_module.tan = math.tan

__wisp_module.log = math.log
__wisp_module.exp = math.exp

return __wisp_module

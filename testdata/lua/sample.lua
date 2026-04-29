-- greet returns a greeting string
function greet(name)
  return "Hello, " .. name
end

-- add returns the sum of two numbers
local function add(a, b)
  return a + b
end

function Point.new(x, y)
  local self = setmetatable({}, Point)
  self.x = x
  self.y = y
  return self
end

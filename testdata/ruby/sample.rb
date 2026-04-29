# greet returns a greeting for the given name.
def greet(name)
  "Hello, #{name}!"
end

# add returns the sum of two numbers.
def add(a, b)
  a + b
end

# Point represents a 2D point.
class Point
  # initialize sets x and y.
  def initialize(x, y)
    @x = x
    @y = y
  end

  # distance returns the distance from the origin.
  def distance
    Math.sqrt(@x**2 + @y**2)
  end
end

# Shape is an abstract module for geometric shapes.
module Shape
  # area must be implemented by subclasses.
  def area
    raise NotImplementedError
  end
end

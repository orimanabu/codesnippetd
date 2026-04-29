# greet returns a greeting for the given name.
def greet(name):
    return "Hello, " + name + "!"


# add returns the sum of two numbers.
def add(a, b):
    return a + b


class Point:
    """A 2D point."""

    # __init__ initializes x and y.
    def __init__(self, x, y):
        self.x = x
        self.y = y

    # distance returns the distance from the origin.
    def distance(self):
        return (self.x**2 + self.y**2) ** 0.5


class Shape:
    """Abstract shape interface."""

    # area must be implemented by subclasses.
    def area(self):
        raise NotImplementedError

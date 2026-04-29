package sample

// Greet returns a greeting message for the given name.
func Greet(name string) string {
	return "Hello, " + name + "!"
}

// Add adds two integers and returns their sum.
func Add(a, b int) int {
	return a + b
}

// GoPoint represents a 2D point with X and Y coordinates.
type GoPoint struct {
	X float64
	Y float64
}

// GoCircle is a circle with a radius.
type GoCircle struct {
	Radius float64
}

// Area returns the area of the GoCircle.
func (c *GoCircle) Area() float64 {
	return 3.14159 * c.Radius * c.Radius
}

// GoShape is an interface for geometric shapes.
type GoShape interface {
	Area() float64
}

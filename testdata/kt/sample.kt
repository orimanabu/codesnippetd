fun greet(name: String): String {
    return "Hello, $name!"
}

fun add(x: Int, y: Int): Int = x + y

data class Point(val x: Double, val y: Double)

class Circle(private val radius: Double) {
    fun area(): Double {
        return Math.PI * radius * radius
    }

    fun perimeter(): Double = 2 * Math.PI * radius
}

interface Shape {
    fun area(): Double
    fun perimeter(): Double
}

object MathUtils {
    fun square(n: Int): Int = n * n
}

enum class Direction {
    UP, DOWN, LEFT, RIGHT
}

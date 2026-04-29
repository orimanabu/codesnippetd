// Greeter class
class Greeter {
public:
    // greet method
    std::string greet(std::string name) {
        return "Hello, " + name + "!";
    }

    // add method
    int add(int a, int b) {
        return a + b;
    }
};

// Point struct
struct Point {
    double x;
    double y;
};

// Color enum
enum class Color { Red, Green, Blue };

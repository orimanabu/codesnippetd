function greet(name) {
  return "Hello, " + name;
}

const add = (a, b) => {
  return a + b;
};

class Point {
  constructor(x, y) {
    this.x = x;
    this.y = y;
  }
  distance() {
    return Math.sqrt(this.x ** 2 + this.y ** 2);
  }
}

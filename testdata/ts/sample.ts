function greet(name: string): string {
  return "Hello, " + name;
}

const add = (a: number, b: number): number => {
  return a + b;
};

interface Shape {
  area(): number;
}

type Point = {
  x: number;
  y: number;
};

class Circle implements Shape {
  constructor(private radius: number) {}
  area(): number {
    return Math.PI * this.radius ** 2;
  }
}

enum Direction {
  Up,
  Down,
  Left,
  Right,
}

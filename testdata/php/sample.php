<?php

function greet(string $name): string {
  return "Hello, " . $name . "!";
}

function add(int $x, int $y): int {
  return $x + $y;
}

class Point {
  public float $x;
  public float $y;

  public function __construct(float $x, float $y) {
    $this->x = $x;
    $this->y = $y;
  }

  public function distance(): float {
    return sqrt($this->x ** 2 + $this->y ** 2);
  }
}

interface Shape {
  public function area(): float;
}

trait Greetable {
  public function hello(): string {
    return "Hello from " . get_class($this);
  }
}

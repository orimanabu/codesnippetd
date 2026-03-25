package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sample JavaScript source used across unit tests.
var jsSample = []byte(`function greet(name) {
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
`)

// sample Rust source used across unit tests.
var rustSample = []byte(`fn greet(name: &str) -> String {
    format!("Hello, {}!", name)
}

fn add(a: i32, b: i32) -> i32 {
    a + b
}

struct RustPoint {
    x: f64,
    y: f64,
}

impl RustPoint {
    fn new(x: f64, y: f64) -> Self {
        RustPoint { x, y }
    }
}
`)

// ---- resolveEndWithTreeSitterRust tests ----

func TestResolveEndWithTreeSitterRust_TopLevelFunction(t *testing.T) {
	// fn greet starts at line 1, ends at line 3
	end, err := resolveEndWithTreeSitterRust(rustSample, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 3 {
		t.Errorf("end: got %d, want 3", end)
	}
}

func TestResolveEndWithTreeSitterRust_SecondFunction(t *testing.T) {
	// fn add starts at line 5, ends at line 7
	end, err := resolveEndWithTreeSitterRust(rustSample, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 7 {
		t.Errorf("end: got %d, want 7", end)
	}
}

func TestResolveEndWithTreeSitterRust_Struct(t *testing.T) {
	// struct RustPoint starts at line 9, ends at line 12
	end, err := resolveEndWithTreeSitterRust(rustSample, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 12 {
		t.Errorf("end: got %d, want 12", end)
	}
}

func TestResolveEndWithTreeSitterRust_ImplBlock(t *testing.T) {
	// impl RustPoint starts at line 14, ends at line 18
	end, err := resolveEndWithTreeSitterRust(rustSample, 14)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 18 {
		t.Errorf("end: got %d, want 18", end)
	}
}

func TestResolveEndWithTreeSitterRust_Method(t *testing.T) {
	// fn new inside impl starts at line 15, ends at line 17
	end, err := resolveEndWithTreeSitterRust(rustSample, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 17 {
		t.Errorf("end: got %d, want 17", end)
	}
}

func TestResolveEndWithTreeSitterRust_LineNotFound(t *testing.T) {
	// line 4 is blank — no definition starts there
	_, err := resolveEndWithTreeSitterRust(rustSample, 4)
	if err == nil {
		t.Fatal("expected error for blank line with no definition")
	}
}

// ---- resolveEndWithTreeSitterJS tests ----

func TestResolveEndWithTreeSitterJS_FunctionDeclaration(t *testing.T) {
	// function greet starts at line 1, ends at line 3
	end, err := resolveEndWithTreeSitterJS(jsSample, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 3 {
		t.Errorf("end: got %d, want 3", end)
	}
}

func TestResolveEndWithTreeSitterJS_ArrowFunction(t *testing.T) {
	// const add = (a, b) => { ... } starts at line 5, ends at line 7
	end, err := resolveEndWithTreeSitterJS(jsSample, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 7 {
		t.Errorf("end: got %d, want 7", end)
	}
}

func TestResolveEndWithTreeSitterJS_Class(t *testing.T) {
	// class Point starts at line 9, ends at line 17
	end, err := resolveEndWithTreeSitterJS(jsSample, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 17 {
		t.Errorf("end: got %d, want 17", end)
	}
}

func TestResolveEndWithTreeSitterJS_Method(t *testing.T) {
	// constructor starts at line 10, ends at line 13
	end, err := resolveEndWithTreeSitterJS(jsSample, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 13 {
		t.Errorf("end: got %d, want 13", end)
	}
}

func TestResolveEndWithTreeSitterJS_SecondMethod(t *testing.T) {
	// distance() starts at line 14, ends at line 16
	end, err := resolveEndWithTreeSitterJS(jsSample, 14)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 16 {
		t.Errorf("end: got %d, want 16", end)
	}
}

func TestResolveEndWithTreeSitterJS_LineNotFound(t *testing.T) {
	// line 4 is blank — no definition starts there
	_, err := resolveEndWithTreeSitterJS(jsSample, 4)
	if err == nil {
		t.Fatal("expected error for blank line with no definition")
	}
}

// sample TypeScript source used across unit tests.
var tsSample = []byte(`function greet(name: string): string {
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
`)

// ---- resolveEndWithTreeSitterTS tests ----

func TestResolveEndWithTreeSitterTS_FunctionDeclaration(t *testing.T) {
	// function greet starts at line 1, ends at line 3
	end, err := resolveEndWithTreeSitterTS(tsSample, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 3 {
		t.Errorf("end: got %d, want 3", end)
	}
}

func TestResolveEndWithTreeSitterTS_ArrowFunction(t *testing.T) {
	// const add = ... starts at line 5, ends at line 7
	end, err := resolveEndWithTreeSitterTS(tsSample, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 7 {
		t.Errorf("end: got %d, want 7", end)
	}
}

func TestResolveEndWithTreeSitterTS_Interface(t *testing.T) {
	// interface Shape starts at line 9, ends at line 11
	end, err := resolveEndWithTreeSitterTS(tsSample, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 11 {
		t.Errorf("end: got %d, want 11", end)
	}
}

func TestResolveEndWithTreeSitterTS_TypeAlias(t *testing.T) {
	// type Point = { ... }; starts at line 13, ends at line 16
	end, err := resolveEndWithTreeSitterTS(tsSample, 13)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 16 {
		t.Errorf("end: got %d, want 16", end)
	}
}

func TestResolveEndWithTreeSitterTS_Class(t *testing.T) {
	// class Circle starts at line 18, ends at line 23
	end, err := resolveEndWithTreeSitterTS(tsSample, 18)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 23 {
		t.Errorf("end: got %d, want 23", end)
	}
}

func TestResolveEndWithTreeSitterTS_Method(t *testing.T) {
	// area() starts at line 20, ends at line 22
	end, err := resolveEndWithTreeSitterTS(tsSample, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 22 {
		t.Errorf("end: got %d, want 22", end)
	}
}

func TestResolveEndWithTreeSitterTS_Enum(t *testing.T) {
	// enum Direction starts at line 25, ends at line 30
	end, err := resolveEndWithTreeSitterTS(tsSample, 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 30 {
		t.Errorf("end: got %d, want 30", end)
	}
}

func TestResolveEndWithTreeSitterTS_LineNotFound(t *testing.T) {
	// line 4 is blank — no definition starts there
	_, err := resolveEndWithTreeSitterTS(tsSample, 4)
	if err == nil {
		t.Fatal("expected error for blank line with no definition")
	}
}

// sample Kotlin source used across unit tests.
// Content must match testdata/kt/sample.kt exactly.
var ktSample = []byte(`fun greet(name: String): String {
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
`)

// ---- resolveEndWithTreeSitterKotlin tests ----

func TestResolveEndWithTreeSitterKotlin_FunctionMultiLine(t *testing.T) {
	// fun greet starts at line 1, ends at line 3
	end, err := resolveEndWithTreeSitterKotlin(ktSample, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 3 {
		t.Errorf("end: got %d, want 3", end)
	}
}

func TestResolveEndWithTreeSitterKotlin_FunctionExpressionBody(t *testing.T) {
	// fun add ... = x + y starts at line 5, ends at line 5
	end, err := resolveEndWithTreeSitterKotlin(ktSample, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 5 {
		t.Errorf("end: got %d, want 5", end)
	}
}

func TestResolveEndWithTreeSitterKotlin_DataClass(t *testing.T) {
	// data class Point starts at line 7, ends at line 7
	end, err := resolveEndWithTreeSitterKotlin(ktSample, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 7 {
		t.Errorf("end: got %d, want 7", end)
	}
}

func TestResolveEndWithTreeSitterKotlin_Class(t *testing.T) {
	// class Circle starts at line 9, ends at line 15
	end, err := resolveEndWithTreeSitterKotlin(ktSample, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 15 {
		t.Errorf("end: got %d, want 15", end)
	}
}

func TestResolveEndWithTreeSitterKotlin_Method(t *testing.T) {
	// fun area inside Circle starts at line 10, ends at line 12
	end, err := resolveEndWithTreeSitterKotlin(ktSample, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 12 {
		t.Errorf("end: got %d, want 12", end)
	}
}

func TestResolveEndWithTreeSitterKotlin_Interface(t *testing.T) {
	// interface Shape starts at line 17, ends at line 20
	end, err := resolveEndWithTreeSitterKotlin(ktSample, 17)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 20 {
		t.Errorf("end: got %d, want 20", end)
	}
}

func TestResolveEndWithTreeSitterKotlin_Object(t *testing.T) {
	// object MathUtils starts at line 22, ends at line 24
	end, err := resolveEndWithTreeSitterKotlin(ktSample, 22)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 24 {
		t.Errorf("end: got %d, want 24", end)
	}
}

func TestResolveEndWithTreeSitterKotlin_Enum(t *testing.T) {
	// enum class Direction starts at line 26, ends at line 28
	end, err := resolveEndWithTreeSitterKotlin(ktSample, 26)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 28 {
		t.Errorf("end: got %d, want 28", end)
	}
}

func TestResolveEndWithTreeSitterKotlin_LineNotFound(t *testing.T) {
	// line 4 is blank — no definition starts there
	_, err := resolveEndWithTreeSitterKotlin(ktSample, 4)
	if err == nil {
		t.Fatal("expected error for blank line with no definition")
	}
}

// sample PHP source used across unit tests.
// Content must match testdata/php/sample.php exactly.
var phpSample = []byte(`<?php

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
`)

// ---- resolveEndWithTreeSitterPHP tests ----

func TestResolveEndWithTreeSitterPHP_FunctionMultiLine(t *testing.T) {
	// function greet starts at line 3, ends at line 5
	end, err := resolveEndWithTreeSitterPHP(phpSample, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 5 {
		t.Errorf("end: got %d, want 5", end)
	}
}

func TestResolveEndWithTreeSitterPHP_FunctionSingleLine(t *testing.T) {
	// function add starts at line 7, ends at line 9
	end, err := resolveEndWithTreeSitterPHP(phpSample, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 9 {
		t.Errorf("end: got %d, want 9", end)
	}
}

func TestResolveEndWithTreeSitterPHP_Class(t *testing.T) {
	// class Point starts at line 11, ends at line 23
	end, err := resolveEndWithTreeSitterPHP(phpSample, 11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 23 {
		t.Errorf("end: got %d, want 23", end)
	}
}

func TestResolveEndWithTreeSitterPHP_Method(t *testing.T) {
	// __construct starts at line 15, ends at line 18
	end, err := resolveEndWithTreeSitterPHP(phpSample, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 18 {
		t.Errorf("end: got %d, want 18", end)
	}
}

func TestResolveEndWithTreeSitterPHP_Interface(t *testing.T) {
	// interface Shape starts at line 25, ends at line 27
	end, err := resolveEndWithTreeSitterPHP(phpSample, 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 27 {
		t.Errorf("end: got %d, want 27", end)
	}
}

func TestResolveEndWithTreeSitterPHP_Trait(t *testing.T) {
	// trait Greetable starts at line 29, ends at line 33
	end, err := resolveEndWithTreeSitterPHP(phpSample, 29)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 33 {
		t.Errorf("end: got %d, want 33", end)
	}
}

func TestResolveEndWithTreeSitterPHP_LineNotFound(t *testing.T) {
	// line 2 is blank — no definition starts there
	_, err := resolveEndWithTreeSitterPHP(phpSample, 2)
	if err == nil {
		t.Fatal("expected error for blank line with no definition")
	}
}

// sample OCaml implementation (.ml) source used across unit tests.
// Content must match testdata/ocaml/sample.ml exactly.
var mlSample = []byte(`let greet name =
  "Hello, " ^ name ^ "!"

let add x y = x + y

type color = Red | Green | Blue

type point = {
  x: float;
  y: float;
}

module type SHAPE = sig
  val area : point -> float
end

module Circle = struct
  let area r = Float.pi *. r *. r
end

class counter init = object
  val mutable count = init
  method increment = count <- count + 1
  method get = count
end
`)

// sample OCaml interface (.mli) source used across unit tests.
// Content must match testdata/ocaml/sample.mli exactly.
var mliSample = []byte(`val greet : string -> string

val add : int -> int -> int

type color = Red | Green | Blue

type point = {
  x: float;
  y: float;
}

module type SHAPE = sig
  val area : point -> float
end
`)

// ---- resolveEndWithTreeSitterOCaml tests (.ml) ----

func TestResolveEndWithTreeSitterOCaml_FunctionMultiLine(t *testing.T) {
	// let greet name = ... starts at line 1, ends at line 2
	end, err := resolveEndWithTreeSitterOCaml(mlSample, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 2 {
		t.Errorf("end: got %d, want 2", end)
	}
}

func TestResolveEndWithTreeSitterOCaml_FunctionSingleLine(t *testing.T) {
	// let add x y = x + y starts at line 4, ends at line 4
	end, err := resolveEndWithTreeSitterOCaml(mlSample, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 4 {
		t.Errorf("end: got %d, want 4", end)
	}
}

func TestResolveEndWithTreeSitterOCaml_TypeVariant(t *testing.T) {
	// type color starts at line 6, ends at line 6
	end, err := resolveEndWithTreeSitterOCaml(mlSample, 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 6 {
		t.Errorf("end: got %d, want 6", end)
	}
}

func TestResolveEndWithTreeSitterOCaml_TypeRecord(t *testing.T) {
	// type point = { ... } starts at line 8, ends at line 11
	end, err := resolveEndWithTreeSitterOCaml(mlSample, 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 11 {
		t.Errorf("end: got %d, want 11", end)
	}
}

func TestResolveEndWithTreeSitterOCaml_ModuleType(t *testing.T) {
	// module type SHAPE starts at line 13, ends at line 15
	end, err := resolveEndWithTreeSitterOCaml(mlSample, 13)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 15 {
		t.Errorf("end: got %d, want 15", end)
	}
}

func TestResolveEndWithTreeSitterOCaml_Module(t *testing.T) {
	// module Circle starts at line 17, ends at line 19
	end, err := resolveEndWithTreeSitterOCaml(mlSample, 17)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 19 {
		t.Errorf("end: got %d, want 19", end)
	}
}

func TestResolveEndWithTreeSitterOCaml_Class(t *testing.T) {
	// class counter starts at line 21, ends at line 25
	end, err := resolveEndWithTreeSitterOCaml(mlSample, 21)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 25 {
		t.Errorf("end: got %d, want 25", end)
	}
}

func TestResolveEndWithTreeSitterOCaml_LineNotFound(t *testing.T) {
	// line 3 is blank — no definition starts there
	_, err := resolveEndWithTreeSitterOCaml(mlSample, 3)
	if err == nil {
		t.Fatal("expected error for blank line with no definition")
	}
}

// ---- resolveEndWithTreeSitterOCamlInterface tests (.mli) ----

func TestResolveEndWithTreeSitterOCamlInterface_ValSingleLine(t *testing.T) {
	// val greet starts at line 1, ends at line 1
	end, err := resolveEndWithTreeSitterOCamlInterface(mliSample, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 1 {
		t.Errorf("end: got %d, want 1", end)
	}
}

func TestResolveEndWithTreeSitterOCamlInterface_TypeRecord(t *testing.T) {
	// type point = { ... } starts at line 7, ends at line 10
	end, err := resolveEndWithTreeSitterOCamlInterface(mliSample, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 10 {
		t.Errorf("end: got %d, want 10", end)
	}
}

func TestResolveEndWithTreeSitterOCamlInterface_ModuleType(t *testing.T) {
	// module type SHAPE starts at line 12, ends at line 14
	end, err := resolveEndWithTreeSitterOCamlInterface(mliSample, 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 14 {
		t.Errorf("end: got %d, want 14", end)
	}
}

func TestResolveEndWithTreeSitterOCamlInterface_LineNotFound(t *testing.T) {
	// line 2 is blank — no definition starts there
	_, err := resolveEndWithTreeSitterOCamlInterface(mliSample, 2)
	if err == nil {
		t.Fatal("expected error for blank line with no definition")
	}
}

// sample Haskell source used across unit tests.
// Content must match testdata/hs/sample.hs exactly.
var hsSample = []byte(`greet :: String -> String
greet name =
  "Hello, " ++ name ++ "!"

add :: Int -> Int -> Int
add x y = x + y

data Color
  = Red
  | Green
  | Blue

data Point = Point
  { px :: Double
  , py :: Double
  }

class Shape a where
  area :: a -> Double
  perimeter :: a -> Double
`)

// ---- resolveEndWithTreeSitterHS tests ----

func TestResolveEndWithTreeSitterHS_FunctionMultiLine(t *testing.T) {
	// greet function starts at line 2, ends at line 3
	end, err := resolveEndWithTreeSitterHS(hsSample, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 3 {
		t.Errorf("end: got %d, want 3", end)
	}
}

func TestResolveEndWithTreeSitterHS_FunctionSingleLine(t *testing.T) {
	// add function starts at line 6, ends at line 6
	end, err := resolveEndWithTreeSitterHS(hsSample, 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 6 {
		t.Errorf("end: got %d, want 6", end)
	}
}

func TestResolveEndWithTreeSitterHS_TypeSignature(t *testing.T) {
	// greet type signature starts at line 1, ends at line 1
	end, err := resolveEndWithTreeSitterHS(hsSample, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 1 {
		t.Errorf("end: got %d, want 1", end)
	}
}

func TestResolveEndWithTreeSitterHS_DataType(t *testing.T) {
	// data Color starts at line 8, ends at line 11
	end, err := resolveEndWithTreeSitterHS(hsSample, 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 11 {
		t.Errorf("end: got %d, want 11", end)
	}
}

func TestResolveEndWithTreeSitterHS_DataTypeRecord(t *testing.T) {
	// data Point starts at line 13, ends at line 16
	end, err := resolveEndWithTreeSitterHS(hsSample, 13)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 16 {
		t.Errorf("end: got %d, want 16", end)
	}
}

func TestResolveEndWithTreeSitterHS_Class(t *testing.T) {
	// class Shape starts at line 18, ends at line 20
	end, err := resolveEndWithTreeSitterHS(hsSample, 18)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end != 20 {
		t.Errorf("end: got %d, want 20", end)
	}
}

func TestResolveEndWithTreeSitterHS_LineNotFound(t *testing.T) {
	// line 4 is blank — no definition starts there
	_, err := resolveEndWithTreeSitterHS(hsSample, 4)
	if err == nil {
		t.Fatal("expected error for blank line with no definition")
	}
}

// ---- isRustFile / isJSFile tests ----

func TestIsRustFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"main.rs", true},
		{"src/lib.rs", true},
		{"main.go", false},
		{"sample.js", false},
		{"README.md", false},
		{"noextension", false},
	}
	for _, c := range cases {
		if got := isRustFile(c.path); got != c.want {
			t.Errorf("isRustFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsJSFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"app.js", true},
		{"src/index.js", true},
		{"main.go", false},
		{"main.rs", false},
		{"app.ts", false},
		{"noextension", false},
	}
	for _, c := range cases {
		if got := isJSFile(c.path); got != c.want {
			t.Errorf("isJSFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsTSFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"app.ts", true},
		{"src/index.ts", true},
		{"main.go", false},
		{"main.rs", false},
		{"app.js", false},
		{"noextension", false},
	}
	for _, c := range cases {
		if got := isTSFile(c.path); got != c.want {
			t.Errorf("isTSFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsPHPFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"index.php", true},
		{"src/app.php", true},
		{"main.go", false},
		{"main.rs", false},
		{"app.ts", false},
		{"noextension", false},
	}
	for _, c := range cases {
		if got := isPHPFile(c.path); got != c.want {
			t.Errorf("isPHPFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsMLFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"main.ml", true},
		{"src/lib.ml", true},
		{"main.mli", false},
		{"main.go", false},
		{"main.rs", false},
		{"noextension", false},
	}
	for _, c := range cases {
		if got := isMLFile(c.path); got != c.want {
			t.Errorf("isMLFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsMLIFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"main.mli", true},
		{"src/lib.mli", true},
		{"main.ml", false},
		{"main.go", false},
		{"main.rs", false},
		{"noextension", false},
	}
	for _, c := range cases {
		if got := isMLIFile(c.path); got != c.want {
			t.Errorf("isMLIFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsHSFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"Main.hs", true},
		{"src/Lib.hs", true},
		{"main.go", false},
		{"main.rs", false},
		{"app.ts", false},
		{"noextension", false},
	}
	for _, c := range cases {
		if got := isHSFile(c.path); got != c.want {
			t.Errorf("isHSFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsKtFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"Main.kt", true},
		{"src/App.kt", true},
		{"main.go", false},
		{"main.rs", false},
		{"app.ts", false},
		{"noextension", false},
	}
	for _, c := range cases {
		if got := isKtFile(c.path); got != c.want {
			t.Errorf("isKtFile(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// ---- HTTP handler integration tests for Rust files ----

func TestSnippetHandler_RustFile_UsesTreeSitter(t *testing.T) {
	// The rust/tags file has no "end" field, so tree-sitter must supply it.
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/greet?context=rust")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		s := snippets[0]
		if s.Start != 1 {
			t.Errorf("Start: got %d, want 1", s.Start)
		}
		if s.End != 3 {
			t.Errorf("End: got %d, want 3 (tree-sitter should resolve this)", s.End)
		}
		if !strings.Contains(s.Code, "fn greet") {
			t.Errorf("Code should contain fn greet, got %q", s.Code)
		}
		if strings.Contains(s.Code, "fn add") {
			t.Errorf("Code should not extend past end of function, got %q", s.Code)
		}
	})
}

func TestLinesHandler_RustFile_UsesTreeSitter(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/add?context=rust")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(ranges) != 1 {
			t.Fatalf("expected 1 range, got %d", len(ranges))
		}
		lr := ranges[0]
		if lr.Start != 5 {
			t.Errorf("Start: got %d, want 5", lr.Start)
		}
		if lr.End != 7 {
			t.Errorf("End: got %d, want 7 (tree-sitter should resolve this)", lr.End)
		}
	})
}

func TestSnippetHandler_RustFile_Method(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/new?context=rust")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 15 || snippets[0].End != 17 {
			t.Errorf("Start/End: got %d/%d, want 15/17", snippets[0].Start, snippets[0].End)
		}
	})
}

// ---- HTTP handler integration tests for JavaScript files ----

func TestSnippetHandler_JSFile_FunctionDeclaration(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/greet?context=js")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		s := snippets[0]
		if s.Start != 1 || s.End != 3 {
			t.Errorf("Start/End: got %d/%d, want 1/3", s.Start, s.End)
		}
		if !strings.Contains(s.Code, "function greet") {
			t.Errorf("Code should contain function greet, got %q", s.Code)
		}
	})
}

func TestSnippetHandler_JSFile_ArrowFunction(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/add?context=js")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 5 || snippets[0].End != 7 {
			t.Errorf("Start/End: got %d/%d, want 5/7", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_JSFile_Class(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Point?context=js")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 9 || snippets[0].End != 17 {
			t.Errorf("Start/End: got %d/%d, want 9/17", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestLinesHandler_JSFile_UsesTreeSitter(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/distance?context=js")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(ranges) != 1 {
			t.Fatalf("expected 1 range, got %d", len(ranges))
		}
		if ranges[0].Start != 14 || ranges[0].End != 16 {
			t.Errorf("Start/End: got %d/%d, want 14/16", ranges[0].Start, ranges[0].End)
		}
	})
}

// ---- HTTP handler integration tests for TypeScript files ----

func TestSnippetHandler_TSFile_FunctionDeclaration(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/greet?context=ts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		s := snippets[0]
		if s.Start != 1 || s.End != 3 {
			t.Errorf("Start/End: got %d/%d, want 1/3", s.Start, s.End)
		}
		if !strings.Contains(s.Code, "function greet") {
			t.Errorf("Code should contain function greet, got %q", s.Code)
		}
		if strings.Contains(s.Code, "const add") {
			t.Errorf("Code should not extend past end of function, got %q", s.Code)
		}
	})
}

func TestSnippetHandler_TSFile_Interface(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Shape?context=ts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 9 || snippets[0].End != 11 {
			t.Errorf("Start/End: got %d/%d, want 9/11", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_TSFile_TypeAlias(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Point?context=ts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 13 || snippets[0].End != 16 {
			t.Errorf("Start/End: got %d/%d, want 13/16", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_TSFile_Class(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Circle?context=ts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 18 || snippets[0].End != 23 {
			t.Errorf("Start/End: got %d/%d, want 18/23", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_TSFile_Enum(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Direction?context=ts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 25 || snippets[0].End != 30 {
			t.Errorf("Start/End: got %d/%d, want 25/30", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestLinesHandler_TSFile_UsesTreeSitter(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/add?context=ts")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(ranges) != 1 {
			t.Fatalf("expected 1 range, got %d", len(ranges))
		}
		if ranges[0].Start != 5 || ranges[0].End != 7 {
			t.Errorf("Start/End: got %d/%d, want 5/7", ranges[0].Start, ranges[0].End)
		}
	})
}

// ---- HTTP handler integration tests for Haskell files ----

func TestSnippetHandler_HSFile_FunctionMultiLine(t *testing.T) {
	// greet function spans lines 2-3; tree-sitter must resolve the end.
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/greet?context=hs")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		s := snippets[0]
		if s.Start != 2 || s.End != 3 {
			t.Errorf("Start/End: got %d/%d, want 2/3", s.Start, s.End)
		}
		if !strings.Contains(s.Code, "greet name") {
			t.Errorf("Code should contain greet name, got %q", s.Code)
		}
		if strings.Contains(s.Code, "add x y") {
			t.Errorf("Code should not extend past end of function, got %q", s.Code)
		}
	})
}

func TestSnippetHandler_HSFile_FunctionSingleLine(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/add?context=hs")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 6 || snippets[0].End != 6 {
			t.Errorf("Start/End: got %d/%d, want 6/6", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_HSFile_DataType(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Color?context=hs")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 8 || snippets[0].End != 11 {
			t.Errorf("Start/End: got %d/%d, want 8/11", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_HSFile_Class(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Shape?context=hs")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 18 || snippets[0].End != 20 {
			t.Errorf("Start/End: got %d/%d, want 18/20", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestLinesHandler_HSFile_UsesTreeSitter(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/Point?context=hs")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(ranges) != 1 {
			t.Fatalf("expected 1 range, got %d", len(ranges))
		}
		if ranges[0].Start != 13 || ranges[0].End != 16 {
			t.Errorf("Start/End: got %d/%d, want 13/16", ranges[0].Start, ranges[0].End)
		}
	})
}

// ---- HTTP handler integration tests for OCaml .ml files ----

func TestSnippetHandler_MLFile_FunctionMultiLine(t *testing.T) {
	// let greet spans lines 1-2; tree-sitter must resolve the end.
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/greet?context=ocaml")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		// greet exists in both sample.ml (line 1) and sample.mli (line 1)
		if len(snippets) != 2 {
			t.Fatalf("expected 2 snippets (ml + mli), got %d", len(snippets))
		}
		byPath := map[string]Snippet{}
		for _, s := range snippets {
			byPath[s.Path] = s
		}

		ml := byPath["sample.ml"]
		if ml.Start != 1 || ml.End != 2 {
			t.Errorf(".ml Start/End: got %d/%d, want 1/2", ml.Start, ml.End)
		}
		if !strings.Contains(ml.Code, "let greet") {
			t.Errorf(".ml Code should contain let greet, got %q", ml.Code)
		}

		mli := byPath["sample.mli"]
		if mli.Start != 1 || mli.End != 1 {
			t.Errorf(".mli Start/End: got %d/%d, want 1/1", mli.Start, mli.End)
		}
		if !strings.Contains(mli.Code, "val greet") {
			t.Errorf(".mli Code should contain val greet, got %q", mli.Code)
		}
	})
}

func TestSnippetHandler_MLFile_TypeRecord(t *testing.T) {
	// type point spans lines 8-11 in .ml, lines 7-10 in .mli
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/point?context=ocaml")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 2 {
			t.Fatalf("expected 2 snippets (ml + mli), got %d", len(snippets))
		}
		byPath := map[string]Snippet{}
		for _, s := range snippets {
			byPath[s.Path] = s
		}
		if byPath["sample.ml"].Start != 8 || byPath["sample.ml"].End != 11 {
			ml := byPath["sample.ml"]
			t.Errorf(".ml Start/End: got %d/%d, want 8/11", ml.Start, ml.End)
		}
		if byPath["sample.mli"].Start != 7 || byPath["sample.mli"].End != 10 {
			mli := byPath["sample.mli"]
			t.Errorf(".mli Start/End: got %d/%d, want 7/10", mli.Start, mli.End)
		}
	})
}

func TestSnippetHandler_MLFile_Module(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Circle?context=ocaml")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 17 || snippets[0].End != 19 {
			t.Errorf("Start/End: got %d/%d, want 17/19", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_MLFile_Class(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/counter?context=ocaml")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 21 || snippets[0].End != 25 {
			t.Errorf("Start/End: got %d/%d, want 21/25", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestLinesHandler_MLFile_UsesTreeSitter(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/color?context=ocaml")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(ranges) != 1 {
			t.Fatalf("expected 1 range, got %d", len(ranges))
		}
		if ranges[0].Start != 6 || ranges[0].End != 6 {
			t.Errorf("Start/End: got %d/%d, want 6/6", ranges[0].Start, ranges[0].End)
		}
	})
}

// ---- HTTP handler integration tests for PHP files ----

func TestSnippetHandler_PHPFile_Function(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/greet?context=php")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		s := snippets[0]
		if s.Start != 3 || s.End != 5 {
			t.Errorf("Start/End: got %d/%d, want 3/5", s.Start, s.End)
		}
		if !strings.Contains(s.Code, "function greet") {
			t.Errorf("Code should contain function greet, got %q", s.Code)
		}
		if strings.Contains(s.Code, "function add") {
			t.Errorf("Code should not extend past end of function, got %q", s.Code)
		}
	})
}

func TestSnippetHandler_PHPFile_Class(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Point?context=php")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 11 || snippets[0].End != 23 {
			t.Errorf("Start/End: got %d/%d, want 11/23", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_PHPFile_Method(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/distance?context=php")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 20 || snippets[0].End != 22 {
			t.Errorf("Start/End: got %d/%d, want 20/22", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_PHPFile_Interface(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Shape?context=php")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 25 || snippets[0].End != 27 {
			t.Errorf("Start/End: got %d/%d, want 25/27", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_PHPFile_Trait(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Greetable?context=php")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 29 || snippets[0].End != 33 {
			t.Errorf("Start/End: got %d/%d, want 29/33", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestLinesHandler_PHPFile_UsesTreeSitter(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/add?context=php")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(ranges) != 1 {
			t.Fatalf("expected 1 range, got %d", len(ranges))
		}
		if ranges[0].Start != 7 || ranges[0].End != 9 {
			t.Errorf("Start/End: got %d/%d, want 7/9", ranges[0].Start, ranges[0].End)
		}
	})
}

// ---- HTTP handler integration tests for Kotlin files ----

func TestSnippetHandler_KTFile_Function(t *testing.T) {
	// kt/tags has no "end" field, so tree-sitter must supply it.
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/greet?context=kt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		s := snippets[0]
		if s.Start != 1 || s.End != 3 {
			t.Errorf("Start/End: got %d/%d, want 1/3", s.Start, s.End)
		}
		if !strings.Contains(s.Code, "fun greet") {
			t.Errorf("Code should contain fun greet, got %q", s.Code)
		}
		if strings.Contains(s.Code, "fun add") {
			t.Errorf("Code should not extend past end of function, got %q", s.Code)
		}
	})
}

func TestSnippetHandler_KTFile_Class(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/Circle?context=kt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 9 || snippets[0].End != 15 {
			t.Errorf("Start/End: got %d/%d, want 9/15", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestSnippetHandler_KTFile_Object(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/snippets/MathUtils?context=kt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var snippets []Snippet
		if err := json.NewDecoder(resp.Body).Decode(&snippets); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(snippets) != 1 {
			t.Fatalf("expected 1 snippet, got %d", len(snippets))
		}
		if snippets[0].Start != 22 || snippets[0].End != 24 {
			t.Errorf("Start/End: got %d/%d, want 22/24", snippets[0].Start, snippets[0].End)
		}
	})
}

func TestLinesHandler_KTFile_UsesTreeSitter(t *testing.T) {
	withCwd(t, "testdata", func() {
		srv := httptest.NewServer(newHandler(true))
		defer srv.Close()

		resp, err := http.Get(srv.URL + "/lines/greet?context=kt")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var ranges []LineRange
		if err := json.NewDecoder(resp.Body).Decode(&ranges); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(ranges) != 1 {
			t.Fatalf("expected 1 range, got %d", len(ranges))
		}
		if ranges[0].Start != 1 || ranges[0].End != 3 {
			t.Errorf("Start/End: got %d/%d, want 1/3", ranges[0].Start, ranges[0].End)
		}
	})
}

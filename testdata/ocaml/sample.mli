val greet : string -> string

val add : int -> int -> int

type color = Red | Green | Blue

type point = {
  x: float;
  y: float;
}

module type SHAPE = sig
  val area : point -> float
end

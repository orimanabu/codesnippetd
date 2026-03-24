let greet name =
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

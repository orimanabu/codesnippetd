greet :: String -> String
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

module HitungBensin where
hitungBensin :: Int -> Int -> Int
hitungBensin a b
  | a == b    = hitungBensinS a 
  | otherwise = hitungBensin a (b - 1) + hitungBensinS b  
hitungBensinS :: Int -> Int
hitungBensinS x
  | x == 1          = 0 
  | even x         = 1 + hitungBensinS (div x 2)  
  | otherwise      = 1 + hitungBensinS (3 * x + 1)
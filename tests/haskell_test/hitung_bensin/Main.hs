import qualified HitungBensin (hitungBensin)

main = do
    a <- getLine
    b <- getLine
    print (HitungBensin.hitungBensin (read a :: Int) (read b :: Int))

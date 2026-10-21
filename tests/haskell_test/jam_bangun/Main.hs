import qualified JamBangun (jamBangun)

main = do
    a <- getLine
    b <- getLine
    c <- getLine
    print (JamBangun.jamBangun (read a :: Int) (read b :: Int) (read c :: Int))

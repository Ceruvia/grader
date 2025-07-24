module JamBangun where

jamBangun :: Int -> Int -> Int -> (Bool, Int, Int, Int)
jamBangun j m d = (isTelat, sj, sm, sd)
    where
        isTelat =  (j < 7 || (j == 7 && m < 45) || (m == 45 && d < 0))
        sd = abs(0 - d)
        sm = if ((isTelat) && sd > 0) then (abs(45 - m)-1)  else abs(45 - m) 
        sj =  abs(7 - j) 
package config

type LedPosition struct {
    Universe int
    Index    int
}

func GenerateXYMapping() map[[2]int]LedPosition {
    mapping := make(map[[2]int]LedPosition)

    for band := 0; band < 64; band++ {
        uEven := band * 2
        uOdd := band*2 + 1

        for i := 0; i < 64; i++ {
            x := band
            y := i
            mapping[[2]int{x, y}] = LedPosition{
                Universe: uEven,
                Index:    1 + i,
            }
        }
        for i := 0; i < 64; i++ {
            x := band
            y := 64 + i
            mapping[[2]int{x, y}] = LedPosition{
                Universe: uOdd,
                Index:    63 - i,
            }
        }
    }

    return mapping
}

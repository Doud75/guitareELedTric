package ehub

import (
    "guitarHetic/internal/config"
    "guitarHetic/internal/domain/artnet"
    "time"
)

func SimulateMessages(output artnet.Output) {
    go func() {
        ticker := time.NewTicker(50 * time.Millisecond)
        defer ticker.Stop()

        universe := 0
        index := 0

        for range ticker.C {
            maxLED := 170
            if universe%2 != 0 {
                maxLED = 88
            }

            output.Send(artnet.LEDMessage{
                Universe: universe,
                Index:    index,
                Color:    [3]byte{0, 0, 255},
            })
            index++
            if index >= maxLED {
                index = 0
                universe++
                if universe >= 128 {
                    universe = 0
                }
            }
        }
    }()
}

func DrawRedSquareCenter(output artnet.Output) {
    mapping := config.GenerateXYMapping()

    centerX := 32
    centerY := 64
    size := 10

    for dx := -size / 2; dx < size/2; dx++ {
        for dy := -size / 2; dy < size/2; dy++ {
            x := centerX + dx
            y := centerY + dy

            pos, ok := mapping[[2]int{x, y}]
            if !ok {
                continue
            }

            output.Send(artnet.LEDMessage{
                Universe: pos.Universe,
                Index:    pos.Index,
                Color:    [3]byte{255, 0, 0},
            })
        }
    }
}

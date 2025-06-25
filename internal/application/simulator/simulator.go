package simulator

import (
    "context"
    "time"

    "guitarHetic/internal/config"
    domainArtnet "guitarHetic/internal/domain/artnet"
)

func RunSpecific(ctx context.Context, in chan<- domainArtnet.LEDMessage, cfg *config.Config) {
    ranges := [][2]int{
        {101, 111}, {347, 357}, {401, 411},
        {647, 657}, {701, 711}, {947, 957},
        {1001, 1011}, {1247, 1257}, {1301, 1311},
        {1547, 1557},
    }
    go func() {
        t := time.NewTicker(100 * time.Millisecond)
        defer t.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-t.C:
                frames := make(map[int][510]byte)
                for _, r := range ranges {
                    for id := r[0]; id <= r[1]; id++ {
                        for _, entry := range cfg.RoutingTable {
                            if entry.EntityID == id {
                                buf := frames[entry.Universe]
                                buf[entry.DMXOffset] = 20
                                frames[entry.Universe] = buf
                                break
                            }
                        }
                    }
                }
                for u, data := range frames {
                    in <- domainArtnet.LEDMessage{Universe: u, Data: data}
                }
            }
        }
    }()
}

func RunMovement(ctx context.Context, in chan<- domainArtnet.LEDMessage, cfg *config.Config) {
    ranges := [][2]int{{101, 111}, {347, 357}, {401, 411}, {647, 657}, {701, 711}, {947, 957}, {1001, 1011}, {1247, 1257}, {1301, 1311}, {1547, 1557}}

    rawMap := make(map[int]config.RoutingEntry, len(cfg.RoutingTable))
    for _, e := range cfg.RoutingTable {
        rawMap[e.EntityID] = e
    }

    dir := 1
    step := 0

    frameTicker := time.NewTicker(60 * time.Millisecond)
    moveTicker := time.NewTicker(500 * time.Millisecond)

    go func() {
        defer frameTicker.Stop()
        defer moveTicker.Stop()
        for {
            select {
            case <-ctx.Done():
                return

            case <-moveTicker.C:
                step += dir
                if step >= 20 || step <= 0 {
                    dir *= -1
                }

            case <-frameTicker.C:
                frames := make(map[int][510]byte, len(cfg.UniverseIP))
                for u := range cfg.UniverseIP {
                    frames[u] = [510]byte{}
                }
                for _, r := range ranges {
                    for id0 := r[0]; id0 <= r[1]; id0++ {
                        e := rawMap[id0]
                        var id1 int
                        if e.Universe%2 == 0 {
                            id1 = id0 + step*dir
                        } else {
                            id1 = id0 - step*dir
                        }
                        if e1, ok := rawMap[id1]; ok {
                            buf := frames[e1.Universe]
                            buf[e1.DMXOffset] = 20
                            frames[e1.Universe] = buf
                        }
                    }
                }
                for u, data := range frames {
                    in <- domainArtnet.LEDMessage{Universe: u, Data: data}
                }
            }
        }
    }()
}

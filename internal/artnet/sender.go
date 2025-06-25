package artnet

import (
    "guitarHetic/internal/domain/artnet"
    "net"
    "time"
)

type sender struct {
    mapping map[int]artnet.UniverseMapping
    frames  map[int][]byte
    queue   chan artnet.LEDMessage
    ticker  *time.Ticker
}

var instance *sender

func Init(mapping map[int]artnet.UniverseMapping) {
    instance = &sender{
        mapping: mapping,
        queue:   make(chan artnet.LEDMessage, 1000),
        frames:  make(map[int][]byte),
        ticker:  time.NewTicker(40 * time.Millisecond),
    }

    for u := 0; u < 128; u++ {
        instance.frames[u] = make([]byte, 510)
    }
}

func Start() {
    go func() {
        for {
            select {
            case msg := <-instance.queue:
                frame := instance.frames[msg.Universe]
                pos := msg.Index * 3
                if pos+2 < len(frame) {
                    frame[pos] = msg.Color[0]
                    frame[pos+1] = msg.Color[1]
                    frame[pos+2] = msg.Color[2]
                }

            case <-instance.ticker.C:
                for u, frame := range instance.frames {
                    if isNonZero(frame) {
                        _ = send(u, frame)
                    }
                }
            }
        }
    }()
}

func isNonZero(data []byte) bool {
    for _, b := range data {
        if b != 0 {
            return true
        }
    }
    return false
}

func Instance() artnet.Output {
    return instance
}

func (s *sender) Send(msg artnet.LEDMessage) {
    s.queue <- msg
}

func send(universe int, data []byte) error {
    ip := instance.mapping[universe].IP
    addr := &net.UDPAddr{IP: net.ParseIP(ip), Port: 6454}
    conn, err := net.DialUDP("udp", nil, addr)
    if err != nil {
        return err
    }
    defer conn.Close()
    packet := Build(universe, data)
    _, err = conn.Write(packet)
    return err
}

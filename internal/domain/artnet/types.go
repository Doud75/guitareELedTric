package artnet

type LEDMessage struct {
    Universe int
    Data     [510]byte
}

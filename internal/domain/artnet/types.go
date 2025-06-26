package artnet

type LEDMessage struct {
    Universe int
    Data     [512]byte
}

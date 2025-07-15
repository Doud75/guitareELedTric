package artnet

type LEDMessage struct {
    DestinationIP string
    Universe      int
    Data          [512]byte
}

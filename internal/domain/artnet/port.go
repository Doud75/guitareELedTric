package artnet

type Output interface {
    Send(LEDMessage)
}

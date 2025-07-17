package artnet

import (
	"fmt"
)

type ArtNetDMXPacket struct {
	SourceIP string
	Universe int
	Sequence uint8
	Data     [512]byte
}

func (p ArtNetDMXPacket) String() string {
	return fmt.Sprintf("[ArtNet-IN] Source: %s | Univers: %d | Seq: %d | Data[0..3]: { %d, %d, %d, %d }",
		p.SourceIP, p.Universe, p.Sequence, p.Data[0], p.Data[1], p.Data[2], p.Data[3])
}
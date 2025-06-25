package ehub

import (
	"fmt"
	"net"
)

type RawPacket struct {
	Data []byte      
	From *net.UDPAddr 
}

type EHubConfigRange struct {
	SextuorStart uint16
	EntityStart  uint16
	SextuorEnd   uint16
	EntityEnd    uint16
}

type EHubConfigMsg struct {
	Universe int
	Ranges   []EHubConfigRange
}

func (m EHubConfigMsg) String() string {
	return fmt.Sprintf("Message Config [Univers eHuB: %d, %d plages]", m.Universe, len(m.Ranges))
}

type EHubEntityState struct {
	ID    uint16
	Red   byte
	Green byte
	Blue  byte
	White byte
}

type EHubUpdateMsg struct {
	Universe int
	Entities []EHubEntityState
}

func (m EHubUpdateMsg) String() string {
	return fmt.Sprintf("Message Update [Univers eHuB: %d, %d entités]", m.Universe, len(m.Entities))
}
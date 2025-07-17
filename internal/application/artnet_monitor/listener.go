// internal/application/artnet_monitor/service.go
package artnet_monitor

import (
	"bytes"
	"encoding/binary"
	"fmt"
	domain "guitarHetic/internal/domain/artnet"
	infra "guitarHetic/internal/infrastructure/artnet_monitor"
)

// --- PARSER ---

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(packet []byte) (*domain.ArtNetDMXPacket, error) {
	if !bytes.Equal(packet[0:8], []byte("Art-Net\x00")) {
		return nil, fmt.Errorf("signature Art-Net invalide")
	}

	opCode := binary.LittleEndian.Uint16(packet[8:10])
	if opCode != 0x5000 { // OpOutput (DMX)
		return nil, fmt.Errorf("paquet Art-Net ignoré (OpCode: 0x%x)", opCode)
	}

	if len(packet) < 18+1 { // Au moins 1 octet de data DMX
		return nil, fmt.Errorf("paquet Art-Net DMX trop court")
	}

	sequence := packet[12]
	universe := int(binary.LittleEndian.Uint16(packet[14:16]))
	dataLength := int(binary.BigEndian.Uint16(packet[16:18]))

	var dmxData [512]byte
	copy(dmxData[:], packet[18:18+dataLength])

	return &domain.ArtNetDMXPacket{
		Universe: universe,
		Sequence: sequence,
		Data:     dmxData,
	}, nil
}


type Service struct {
	rawPacketIn <-chan infra.RawArtNetPacket
	parser      *Parser
	parsedOut   chan<- *domain.ArtNetDMXPacket
}

func NewService(
	rawIn <-chan infra.RawArtNetPacket,
	parsedOut chan<- *domain.ArtNetDMXPacket,
) *Service {
	return &Service{
		rawPacketIn: rawIn,
		parser:      NewParser(),
		parsedOut:   parsedOut,
	}
}

func (s *Service) Start() {
	go func() {
		for rawPkt := range s.rawPacketIn {
			parsedMsg, err := s.parser.Parse(rawPkt.Data)
			if err != nil {
				continue
			}
			parsedMsg.SourceIP = rawPkt.From.IP.String()

			s.parsedOut <- parsedMsg
		}
	}()
}
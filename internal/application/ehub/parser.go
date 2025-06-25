package ehub

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"guitarHetic/internal/domain/ehub"
	"io"
)


type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(packet []byte) (any, error) {
	if len(packet) < 10 {
		return nil, fmt.Errorf("paquet trop petit pour être un message eHuB (taille: %d)", len(packet))
	}

	if string(packet[0:4]) != "eHuB" {
		return nil, fmt.Errorf("signature 'eHuB' non trouvée")
	}

	messageType := packet[4] 
	eHubUniverse := int(packet[5])
	
	compressedPayloadSize := binary.LittleEndian.Uint16(packet[8:10])
	if int(compressedPayloadSize)+10 > len(packet) {
		return nil, fmt.Errorf("taille de payload incohérente")
	}
	
	compressedPayload := packet[10 : 10+compressedPayloadSize]

	// Unzip stage
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedPayload))
	if err != nil {
		return nil, fmt.Errorf("impossible de créer le lecteur gzip: %w", err)
	}
	defer gzipReader.Close()

	payload, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("impossible de décompresser le payload: %w", err)
	}

	switch messageType {
	case 1: // config
		return p.parseConfigPayload(eHubUniverse, payload)
	case 2: // update
		return p.parseUpdatePayload(eHubUniverse, payload)
	default:
		return nil, fmt.Errorf("type de message eHuB inconnu: %d", messageType)
	}
}

func (p *Parser) parseConfigPayload(universe int, payload []byte) (*ehub.EHubConfigMsg, error) {
	reader := bytes.NewReader(payload)
	var ranges []ehub.EHubConfigRange
	
	for reader.Len() >= 8 {
		var r ehub.EHubConfigRange
		if err := binary.Read(reader, binary.LittleEndian, &r); err != nil {
			return nil, err
		}
		ranges = append(ranges, r)
	}
	
	return &ehub.EHubConfigMsg{
		Universe: universe,
		Ranges:   ranges,
	}, nil
}

func (p *Parser) parseUpdatePayload(universe int, payload []byte) (*ehub.EHubUpdateMsg, error) {
	reader := bytes.NewReader(payload)
	var entities []ehub.EHubEntityState

	for reader.Len() >= 6 {
		var entity ehub.EHubEntityState
		
		if err := binary.Read(reader, binary.LittleEndian, &entity.ID); err != nil {
			return nil, err
		}
		
		colors := make([]byte, 4)
		if _, err := reader.Read(colors); err != nil {
			return nil, err
		}
		entity.Red = colors[0]
		entity.Green = colors[1]
		entity.Blue = colors[2]
		entity.White = colors[3]
		entities = append(entities, entity)
	}

	return &ehub.EHubUpdateMsg{
		Universe: universe,
		Entities: entities,
	}, nil
}
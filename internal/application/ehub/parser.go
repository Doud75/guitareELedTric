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
	fmt.Printf("🔄 eHub PARSER: Tentative de parsing d'un paquet de %d bytes\n", len(packet))
	
	if len(packet) < 10 {
		fmt.Printf("❌ eHub PARSER: Paquet trop petit (%d bytes)\n", len(packet))
		return nil, fmt.Errorf("paquet trop petit pour être un message eHuB (taille: %d)", len(packet))
	}

	if string(packet[0:4]) != "eHuB" {
		fmt.Printf("❌ eHub PARSER: Signature incorrecte: '%s'\n", string(packet[0:4]))
		return nil, fmt.Errorf("signature 'eHuB' non trouvée")
	}

	messageType := packet[4] 
	eHubUniverse := int(packet[5])
	
	// Log pour TOUS les univers pour voir ce qui passe
	fmt.Printf("🌍 eHub PARSER: Univers %d, Type %d\n", eHubUniverse, messageType)
	
	// Log spécial pour les univers problématiques
	if eHubUniverse == 13 || eHubUniverse == 18 {
		fmt.Printf("🔍 eHub PARSER FOCUS: Univers %d, Type %d, Packet size: %d bytes\n", eHubUniverse, messageType, len(packet))
	}
	
	compressedPayloadSize := binary.LittleEndian.Uint16(packet[8:10])
	if int(compressedPayloadSize)+10 > len(packet) {
		return nil, fmt.Errorf("taille de payload incohérente")
	}
	
	compressedPayload := packet[10 : 10+compressedPayloadSize]

	// Création d'un nouveau lecteur gzip pour chaque décompression
	// Cette approche est plus sûre que l'utilisation d'un pool
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedPayload))
	if err != nil {
		return nil, fmt.Errorf("impossible de créer le lecteur gzip: %w", err)
	}
	defer gzipReader.Close() // Important : fermer le lecteur
	
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
		
		// Log spécialement pour les entités des bandes 13 et 18
		if (entity.ID >= 1900 && entity.ID <= 2069) || (entity.ID >= 2670 && entity.ID <= 2758) {
			bandeName := ""
			if entity.ID >= 1900 && entity.ID <= 2069 {
				bandeName = "BANDE 13"
			} else {
				bandeName = "BANDE 18"
			}
			fmt.Printf("🎯 %s: Entity %d -> R:%d G:%d B:%d W:%d (Univers eHub: %d)\n", 
				bandeName, entity.ID, entity.Red, entity.Green, entity.Blue, entity.White, universe)
		}
		
		entities = append(entities, entity)
	}

	// Log de synthèse pour les univers contenant les bandes problématiques
	bande13Count := 0
	bande18Count := 0
	for _, entity := range entities {
		if entity.ID >= 1900 && entity.ID <= 2069 {
			bande13Count++
		}
		if entity.ID >= 2670 && entity.ID <= 2758 {
			bande18Count++
		}
	}
	
	if bande13Count > 0 || bande18Count > 0 {
		fmt.Printf("📊 eHub PARSER: Univers %d -> %d entités total", universe, len(entities))
		if bande13Count > 0 {
			fmt.Printf(", BANDE 13: %d entités", bande13Count)
		}
		if bande18Count > 0 {
			fmt.Printf(", BANDE 18: %d entités", bande18Count)
		}
		fmt.Printf("\n")
	}

	return &ehub.EHubUpdateMsg{
		Universe: universe,
		Entities: entities,
	}, nil
}
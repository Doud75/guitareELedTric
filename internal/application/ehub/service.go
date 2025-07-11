package ehub

import (
	"guitarHetic/internal/domain/ehub"
	"log"
)

type Service struct {
	rawPacketIn   <-chan ehub.RawPacket 
	parser        *Parser
	configOut chan<- *ehub.EHubConfigMsg
	updateOut chan<- *ehub.EHubUpdateMsg
}

func NewService(
	rawPacketIn <-chan ehub.RawPacket,
	parser *Parser,
	configOut chan<- *ehub.EHubConfigMsg,
	updateOut chan<- *ehub.EHubUpdateMsg,
) *Service { 
	return &Service{
		rawPacketIn:   rawPacketIn,
		parser:        parser,
		configOut:     configOut,
		updateOut:     updateOut,
	}
}

func (s *Service) Start() {
	go func() {
		log.Println("eHub Service: Démarré, prêt à parser et router les messages.")
		
		for rawPkt := range s.rawPacketIn {
			parsedMessage, err := s.parser.Parse(rawPkt.Data)
			if err != nil {
				log.Printf("eHub Service: Erreur de parsing: %v", err)
				continue
			}
			
			
			switch msg := parsedMessage.(type) {
			case *ehub.EHubConfigMsg:
				if msg.Universe == 13 || msg.Universe == 18 {
					log.Printf("📤 eHub SERVICE: Config pour univers %d", msg.Universe)
				}
				s.configOut <- msg
				
			case *ehub.EHubUpdateMsg:
				// Compter les entités des bandes problématiques
				bande13Count := 0
				bande18Count := 0
				for _, entity := range msg.Entities {
					if entity.ID >= 1900 && entity.ID <= 2069 {
						bande13Count++
					}
					if entity.ID >= 2670 && entity.ID <= 2758 {
						bande18Count++
					}
				}
				
				if bande13Count > 0 || bande18Count > 0 {
					log.Printf("📤 eHub SERVICE: Univers %d avec %d entités total", msg.Universe, len(msg.Entities))
					if bande13Count > 0 {
						log.Printf("   → BANDE 13: %d entités (1900-2069)", bande13Count)
					}
					if bande18Count > 0 {
						log.Printf("   → BANDE 18: %d entités (2670-2758)", bande18Count)
					}
				}
				
   				s.updateOut <- msg
			default:
				log.Printf("eHub Service: Type de message inconnu reçu du parser.")
			}
		}
	}()
}
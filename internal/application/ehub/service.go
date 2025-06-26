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
				s.configOut <- msg
				
			case *ehub.EHubUpdateMsg:
   				s.updateOut <- msg
			default:
				log.Printf("eHub Service: Type de message inconnu reçu du parser.")
			}
		}
	}()
}
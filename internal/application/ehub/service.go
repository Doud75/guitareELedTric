// File: internal/application/ehub/service.go
package ehub

import (
	"log"
	"guitarHetic/internal/domain/ehub" 
)

// La struct Service a maintenant besoin de canaux de sortie.
type Service struct {
	rawPacketIn   <-chan ehub.RawPacket // Canal d'entrée (en lecture seule)
	parser        *Parser
	
	// Canaux de SORTIE (en écriture seule) pour le Processor
	configOut chan<- *ehub.EHubConfigMsg
	updateOut chan<- *ehub.EHubUpdateMsg
}

// NewService : le constructeur est mis à jour pour accepter les canaux de sortie.
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

// Start : la logique est maintenant de router les messages parsés.
func (s *Service) Start() {
	go func() {
		log.Println("eHub Service: Démarré, prêt à parser et router les messages.")
		
		for rawPkt := range s.rawPacketIn {
			parsedMessage, err := s.parser.Parse(rawPkt.Data)
			if err != nil {
				log.Printf("eHub Service: Erreur de parsing: %v", err)
				continue
			}
			
			// On utilise un "type switch". C'est comme une série de `if (typeof message === '...')`
			// qui permet de vérifier le type exact de `parsedMessage`.
			switch msg := parsedMessage.(type) {
			case *ehub.EHubConfigMsg:
				// Si c'est un message de config, on le met sur le canal de config.
				s.configOut <- msg
				
			case *ehub.EHubUpdateMsg:
				 if len(msg.Entities) > 0 {
        log.Printf("eHub Service: Reçu UPDATE pour %d entités. Première entité ID: %d, Couleur R: %d", len(msg.Entities), msg.Entities[0].ID, msg.Entities[0].Red)
    }
    s.updateOut <- msg

			default:
				log.Printf("eHub Service: Type de message inconnu reçu du parser.")
			}
		}
	}()
}
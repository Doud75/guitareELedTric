package ehub

import (
	"log"
	"guitarHetic/internal/domain/ehub" 
)

type Service struct {
	rawPacketChan <-chan ehub.RawPacket 
	parser        *Parser
}

func NewService(rawPacketChan <-chan ehub.RawPacket, parser *Parser) *Service { 
	return &Service{
		rawPacketChan: rawPacketChan,
		parser:        parser,
	}
}

func (s *Service) Start() {
	go func() {
		log.Println("Application eHuB: Service démarré.")
		
		for rawPkt := range s.rawPacketChan {
			parsedMessage, err := s.parser.Parse(rawPkt.Data)
			if err != nil {
				log.Printf("Erreur de parsing eHuB: %v", err)
				continue
			}
			//@TODO  send to another canal or process the parsed message
			// For now, i just make a log, i'll change it once the router table is done
			log.Printf("Message eHuB décodé avec succès: %+v", parsedMessage)
		}
	}()
}
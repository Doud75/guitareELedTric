package artnet

import (
	"context"
	"log"
	"net"
	"time" 
    
	domainArtnet "guitarHetic/internal/domain/artnet"
)

const dmxDataSize = 512

type Sender struct {
	conns       map[int]*net.UDPConn
	headerCache map[int][]byte     
	ticker      *time.Ticker       
}


func NewSender(universeIP map[int]string) (*Sender, error) {
	s := &Sender{
		conns:       make(map[int]*net.UDPConn),
		headerCache: make(map[int][]byte),
		// 33 ms = 30 images par secondes 
		ticker:      time.NewTicker(33 * time.Millisecond),
	}

	log.Println("ArtNet Sender: Initialisation et pré-calcul des en-têtes...")
	for u, ip := range universeIP {
		// Pré-calculer et stocker l'en-tête pour cet univers. 
        // comme ça dans la boucle on a pas besoin de le recalculer 
		s.headerCache[u] = domainArtnet.BuildArtNetHeader(u)
		
		addr := &net.UDPAddr{IP: net.ParseIP(ip), Port: 6454}
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			s.Close()
			return nil, err
		}
		s.conns[u] = conn
	}
	log.Printf("ArtNet Sender: Initialisé pour %d univers.", len(universeIP))
	return s, nil
}


func (s *Sender) Run(ctx context.Context, in <-chan domainArtnet.LEDMessage) {
	log.Println("ArtNet Sender: Démarrage de la goroutine d'envoi (mode optimisé).")

    //la on fait ce que Kevin a dit , on envoi pas les messages direct 
	// on fait un map pour stocker l'état le plus récent de chaque univers.
	// La clé est le numéro de l'univers.
	// La valeur est un POINTEUR vers le tableau de données.
	latestFrames := make(map[int]*[dmxDataSize]byte)

	for {
		select {
		
		case <-ctx.Done():
			s.Close() 
			log.Println("ArtNet Sender: Goroutine d'envoi terminée.")
			return
			
		// Événement on recoit un message dans depuis le processor (en gros ehub)
		case msg := <-in:

			// On ne l'envoie PAS tout de suite.
			// On met simplement à jour notre map
			// du coup si y'a 10 messages pour l'univers 5 arrivent avant le prochain tick,
			// bah on ne gardera que le 10ème, le plus récent.
			
			// On vérifie si on a déjà un buffer pour cet univers.
			if _, ok := latestFrames[msg.Universe]; !ok {
				latestFrames[msg.Universe] = new([dmxDataSize]byte)
			}
			*latestFrames[msg.Universe] = msg.Data

		// La on on attend le go du ticker.
		case <-s.ticker.C:
			// On parcourt tous les univers dont on connaît l'état.
			for universe, data := range latestFrames {
				conn, ok := s.conns[universe]
				if !ok { continue } // ca faut que je creuse demain j'ai pas capté 

				header, ok := s.headerCache[universe]
				if !ok { continue } // Sécurité: pas d'en-tête pour cet univers.
				
				
				// On assemble le paquet final en utilisant notre en-tête pré-calculé.
				// (On pourrait encore optimiser en utilisant un pool de buffers ici visiblement).
				packet := make([]byte, 18 + dmxDataSize)
				copy(packet[0:18], header)
				copy(packet[18:], data[:])

                log.Printf("ArtNet Sender: Envoi de l'univers %d (%s) avec %d octets de données.", universe, conn.RemoteAddr().String(), packet[18:])
                // j'ai tout ignoré faudrait ajouter un log d'erreur mais j'voulais pas spam la console
				_, _ = conn.Write(packet)
			}
		}
	}
}

func (s *Sender) Close() {
	s.ticker.Stop() 
	log.Println("ArtNet Sender: Ticker arrêté.")
	for _, conn := range s.conns {
		if conn != nil {
			conn.Close()
		}
	}
	log.Println("ArtNet Sender: Connexions UDP fermées.")
}
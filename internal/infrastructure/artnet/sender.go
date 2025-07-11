package artnet

import (
	"context"
	"fmt"
	domainArtnet "guitarHetic/internal/domain/artnet"
	"log"
	"net"
	"time"
)


const dmxDataSize = 512
const tickDuration = 33 * time.Millisecond // 30 FPS la

type Sender struct {
	conns       map[int]*net.UDPConn
	headerCache map[int][]byte     
	ticker      *time.Ticker 
	lastSentFrames map[int]*[dmxDataSize]byte 
}


func NewSender(universeIP map[int]string) (*Sender, error) {
	s := &Sender{
		conns:       make(map[int]*net.UDPConn),
		headerCache: make(map[int][]byte),
		ticker:      time.NewTicker(tickDuration),
		lastSentFrames: make(map[int]*[dmxDataSize]byte),
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
	log.Println("ArtNet Sender: Démarrage de la goroutine d'envoi (TICKER + LAST FRAMES).")

	// Map pour stocker la dernière frame de chaque univers
	latestFrames := make(map[int]*[dmxDataSize]byte)

	for {
		select {
		case <-ctx.Done():
			s.Close()
			log.Println("ArtNet Sender: Goroutine d'envoi terminée.")
			return

		// Réception des messages: on stocke juste la dernière frame
		case msg := <-in:
			if _, ok := latestFrames[msg.Universe]; !ok {
				latestFrames[msg.Universe] = new([dmxDataSize]byte)
			}
			*latestFrames[msg.Universe] = msg.Data
			
			// Log spécial pour les univers des bandes problématiques
			if msg.Universe == 12 || msg.Universe == 17 {
				bandeName := ""
				if msg.Universe == 12 {
					bandeName = "BANDE 13"
				} else {
					bandeName = "BANDE 18"
				}
				fmt.Printf("📥 ArtNet SENDER: Reçu données pour %s (Univers ArtNet %d) -> DMX[0-11]: [%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d]\n", 
					bandeName, msg.Universe, 
					msg.Data[0], msg.Data[1], msg.Data[2], msg.Data[3],
					msg.Data[4], msg.Data[5], msg.Data[6], msg.Data[7],
					msg.Data[8], msg.Data[9], msg.Data[10], msg.Data[11])
			}

		// Envoi périodique à 30 FPS
		case <-s.ticker.C:
			packetCount := 0

			for universe, frameData := range latestFrames {
				// Récupération de la connexion
				conn, ok := s.conns[universe]
				if !ok {
					continue
				}

				// Récupération de l'en-tête pré-calculé
				header, ok := s.headerCache[universe]
				if !ok {
					continue
				}

				// Construction et envoi du paquet
				packet := make([]byte, 18+dmxDataSize)
				copy(packet[0:18], header)
				copy(packet[18:], frameData[:])

				_, err := conn.Write(packet)
				if err != nil {
					log.Printf("ArtNet Sender: Erreur envoi univers %d: %v", universe, err)
				} else {
					// Log spécial pour les univers des bandes problématiques
					if universe == 12 || universe == 17 {
						bandeName := ""
						if universe == 12 {
							bandeName = "BANDE 13"
						} else {
							bandeName = "BANDE 18"
						}
						fmt.Printf("📤 ArtNet SENDER: Envoi %s (Univers %d) -> DMX[0-11]: [%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d]\n", 
							bandeName, universe,
							frameData[0], frameData[1], frameData[2], frameData[3],
							frameData[4], frameData[5], frameData[6], frameData[7],
							frameData[8], frameData[9], frameData[10], frameData[11])
					}
				}
				
				packetCount++
			}

			if packetCount > 0 {
				log.Printf("ArtNet Sender: Envoyé %d paquets à 30 FPS", packetCount)
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
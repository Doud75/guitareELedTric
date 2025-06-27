package artnet

import (
	"bytes"
	"context"
	domainArtnet "guitarHetic/internal/domain/artnet"
	"log"
	"net"
	"time"
)


const dmxDataSize = 512
const tickDuration = 33 * time.Millisecond // 30 FPS la
const refreshRate = 30 // 30 FPS

type Sender struct {
	conns       map[int]*net.UDPConn
	headerCache map[int][]byte     
	ticker      *time.Ticker 
	lastSentFrames map[int]*[dmxDataSize]byte 
	refreshCounter int 
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
	log.Println("ArtNet Sender: Démarrage de la goroutine d'envoi (optimisée avec Diffing).")

	// la on fait ce que Kevin a dit , on envoi pas les messages direct
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

			if _, ok := latestFrames[msg.Universe]; !ok {
				latestFrames[msg.Universe] = new([dmxDataSize]byte)
			}
			*latestFrames[msg.Universe] = msg.Data

		case <-s.ticker.C:

			s.refreshCounter++

			isForceRefresh := s.refreshCounter >= refreshRate
			if isForceRefresh {
				s.refreshCounter = 0
			}

			var packetsToSend []struct {
				conn   *net.UDPConn
				packet []byte
				uni    int
			}

			for universe, currentData := range latestFrames {
				lastData, found := s.lastSentFrames[universe]
				
				// On envoie seulement si: c'est la première fois (!found) OU si les données ont changé.
				if isForceRefresh || !found || !bytes.Equal(lastData[:], currentData[:]) {
					conn, ok := s.conns[universe]
					if !ok {
						continue
					}
					header, ok := s.headerCache[universe]
					if !ok {
						continue
					}

					packet := make([]byte, 18+dmxDataSize)
					copy(packet[0:18], header)
					copy(packet[18:], currentData[:])
					
					// Au lieu d'envoyer tout de suite, on ajoute le paquet à notre liste de travail.
					packetsToSend = append(packetsToSend, struct {
						conn   *net.UDPConn
						packet []byte
						uni    int
					}{conn, packet, universe})

					// On met à jour la mémoire tout de suite.
					if !found {
						s.lastSentFrames[universe] = new([dmxDataSize]byte)
					}
					copy(s.lastSentFrames[universe][:], currentData[:])
				}
			}
			

			// 2. Maintenant, on envoie la liste de travail de manière étalée.
			if len(packetsToSend) > 0 {
				
				pacingDuration := (6*time.Millisecond) / time.Duration(len(packetsToSend))
				
				for _, p := range packetsToSend {
					// Log final avant l'envoi réel sur le réseau.
					log.Printf("ArtNet Sender: Envoi de l'univers %d (%s) avec %d octets.", p.uni, p.conn.RemoteAddr().String(), len(p.packet))
					// j'ai tout ignoré faudrait ajouter un log d'erreur mais j'voulais pas spam la console
					_, _ = p.conn.Write(p.packet)
					
					// On fait la pause.
					time.Sleep(pacingDuration)
				}
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
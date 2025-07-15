// internal/infrastructure/artnet/sender.go
package artnet

import (
    "context"
    "fmt"
    domainArtnet "guitarHetic/internal/domain/artnet"
    "log"
    "net"
    "sync"
)

// Sender est maintenant un expéditeur sans état.
type Sender struct {
    conn        *net.UDPConn // Une seule connexion pour envoyer, comme une "poste"
    headerCache *sync.Map    // Cache thread-safe pour les en-têtes
}

func NewSender() (*Sender, error) {
    // On crée une connexion sortante générique, sans destination fixe.
    conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
    if err != nil {
        return nil, fmt.Errorf("impossible de créer la connexion UDP sortante: %w", err)
    }

    log.Println("ArtNet Sender: Initialisé (mode sans état).")
    return &Sender{
        conn:        conn,
        headerCache: &sync.Map{},
    }, nil
}

func (s *Sender) getOrBuildHeader(universe int) []byte {
    header, found := s.headerCache.Load(universe)
    if found {
        return header.([]byte)
    }
    newHeader := domainArtnet.BuildArtNetHeader(universe)
    s.headerCache.Store(universe, newHeader)
    return newHeader
}

func (s *Sender) Run(ctx context.Context, in <-chan domainArtnet.LEDMessage) {
    log.Println("ArtNet Sender: Démarrage de la goroutine d'envoi.")
    defer s.conn.Close()
    defer log.Println("ArtNet Sender: Goroutine d'envoi terminée.")

    for {
        select {
        case <-ctx.Done():
            return
        case msg := <-in:
            // La destination est maintenant dans le message !
            destAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", msg.DestinationIP, 6454))
            if err != nil {
                log.Printf("ArtNet Sender: Adresse IP invalide '%s': %v", msg.DestinationIP, err)
                continue
            }

            header := s.getOrBuildHeader(msg.Universe)

            packet := make([]byte, 18+512)
            copy(packet[0:18], header)
            copy(packet[18:], msg.Data[:])

            // On envoie le paquet à la destination spécifiée dans le message.
            _, err = s.conn.WriteToUDP(packet, destAddr)
            if err != nil {
                log.Printf("ArtNet Sender: Erreur envoi vers %s (Univers %d): %v", msg.DestinationIP, msg.Universe, err)
            }
        }
    }
}

func (s *Sender) Close() {
    s.conn.Close()
}

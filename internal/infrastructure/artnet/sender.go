package artnet

import (
	"context"
	"log"
	"net"

	domainArtnet "guitarHetic/internal/domain/artnet"
)

type Sender struct {
    conns map[int]*net.UDPConn
}

func NewSender(universeIP map[int]string) (*Sender, error) {
    s := &Sender{conns: make(map[int]*net.UDPConn, len(universeIP))}
    for u, ip := range universeIP {
        addr := &net.UDPAddr{IP: net.ParseIP(ip), Port: 6454}
        conn, err := net.DialUDP("udp", nil, addr)
        if err != nil {
            return nil, err
        }
        s.conns[u] = conn
    }
    return s, nil
}

func (s *Sender) Run(ctx context.Context, in <-chan domainArtnet.LEDMessage) {
    for {
        select {
        case <-ctx.Done():
            s.Close()
            return
        case msg := <-in:
			log.Printf("ArtNet Sender: Reçu un buffer DMX à envoyer -> Univers: %d", msg.Universe)
			log.Printf("   -> Aperçu des données: %X", msg.Data[:12])

            packet := domainArtnet.Build(msg.Universe, msg.Data)
            conn, ok := s.conns[msg.Universe]
            if !ok {
                log.Printf("no connection for universe %d", msg.Universe)
                continue
            }
            if _, err := conn.Write(packet); err != nil {
                log.Printf("ArtNet send error (univ %d): %v", msg.Universe, err)
            }
        }
    }
}

func (s *Sender) Close() {
    for _, conn := range s.conns {
        conn.Close()
    }
}

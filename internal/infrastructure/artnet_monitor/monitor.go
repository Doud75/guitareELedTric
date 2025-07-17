// internal/infrastructure/artnet_monitor/.go
package artnet_monitor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
)

type RawArtNetPacket struct {
	Data []byte
	From *net.UDPAddr
}

type Listener struct {
	conn       *net.UDPConn
	packetChan chan<- RawArtNetPacket
}

func NewListener(port int, packetChan chan<- RawArtNetPacket) (*Listener, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("impossible de résoudre l'adresse UDP ArtNet: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("impossible d'écouter sur le port ArtNet %d: %w", port, err)
	}

	log.Printf("ArtNet Monitor: Prêt et à l'écoute sur le port %d", port)
	return &Listener{conn: conn, packetChan: packetChan}, nil
}

func (l *Listener) Start(ctx context.Context) {
	go func() {
		go func() {
			<-ctx.Done()
			l.conn.Close()
		}()

		buffer := make([]byte, 1024)
		for {
			n, remoteAddr, err := l.conn.ReadFromUDP(buffer)
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					log.Println("ArtNet Monitor: Connexion fermée, arrêt de l'écoute.")
					return
				}
				log.Printf("Erreur de lecture UDP (ArtNet): %v", err)
				continue
			}

			packetCopy := make([]byte, n)
			copy(packetCopy, buffer[:n])

			select {
			case l.packetChan <- RawArtNetPacket{Data: packetCopy, From: remoteAddr}:
			case <-ctx.Done():
				return
			}
		}
	}()
}
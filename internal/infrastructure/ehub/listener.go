// internal/infrastructure/ehub/listener.go
package ehub

import (
    "context"
    "errors"
    "fmt"
    "guitarHetic/internal/domain/ehub"
    "log"
    "net"
)

type Listener struct {
    conn       *net.UDPConn
    packetChan chan<- ehub.RawPacket
}

func NewListener(port int, packetChan chan<- ehub.RawPacket) (*Listener, error) {
    addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, fmt.Errorf("impossible de résoudre l'adresse UDP: %w", err)
    }

    conn, err := net.ListenUDP("udp", addr)
    if err != nil {
        return nil, fmt.Errorf("impossible d'écouter sur le port %d: %w", port, err)
    }

    log.Printf("Infrastructure eHuB: Listener prêt et à l'écoute sur le port %d", port)

    return &Listener{
        conn:       conn,
        packetChan: packetChan,
    }, nil
}

func (l *Listener) Start(ctx context.Context) {
    go func() {
        go func() {
            <-ctx.Done()
            l.conn.Close()
        }()

        buffer := make([]byte, 20000)
        for {
            n, remoteAddr, err := l.conn.ReadFromUDP(buffer)
            if err != nil {
                if errors.Is(err, net.ErrClosed) {
                    log.Println("Listener: Connexion fermée, arrêt de la goroutine d'écoute.")
                    return
                }
                log.Printf("Erreur de lecture UDP: %v", err)
                continue
            }

            packetCopy := make([]byte, n)
            copy(packetCopy, buffer[:n])

            select {
            case l.packetChan <- ehub.RawPacket{
                Data: packetCopy,
                From: remoteAddr,
            }:
            case <-ctx.Done():
                // Si le contexte est annulé pendant qu'on attend pour envoyer, on sort aussi.
                return
            }
        }
    }()
}

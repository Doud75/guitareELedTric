package artnet

import (
    "context"
    domainArtnet "guitarHetic/internal/domain/artnet"
    "log"
    "net"
    "time"
)

const dmxDataSize = 512
const tickDuration = 33 * time.Millisecond // 30 FPS la

type Sender struct {
    conns          map[int]*net.UDPConn
    headerCache    map[int][]byte
    ticker         *time.Ticker
    lastSentFrames map[int]*[dmxDataSize]byte
}

func NewSender(universeIP map[int]string) (*Sender, error) {
    s := &Sender{
        conns:          make(map[int]*net.UDPConn),
        headerCache:    make(map[int][]byte),
        ticker:         time.NewTicker(tickDuration),
        lastSentFrames: make(map[int]*[dmxDataSize]byte),
    }

    log.Println("ArtNet Sender: Initialisation et pré-calcul des en-têtes...")
    for u, ip := range universeIP {
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

    latestFrames := make(map[int]*[dmxDataSize]byte)

    for {
        select {
        case <-ctx.Done():
            s.Close()
            log.Println("ArtNet Sender: Goroutine d'envoi terminée.")
            return

        case msg := <-in:
            if _, ok := latestFrames[msg.Universe]; !ok {
                latestFrames[msg.Universe] = new([dmxDataSize]byte)
            }
            *latestFrames[msg.Universe] = msg.Data

        case <-s.ticker.C:
            packetCount := 0

            for universe, frameData := range latestFrames {
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
                copy(packet[18:], frameData[:])

                _, err := conn.Write(packet)
                if err != nil {
                    log.Printf("ArtNet Sender: Erreur envoi univers %d: %v", universe, err)
                }

                packetCount++
            }

            if packetCount > 0 {
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

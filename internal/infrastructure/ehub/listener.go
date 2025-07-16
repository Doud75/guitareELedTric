// internal/infrastructure/ehub/listener.go
package ehub

import (
	"context"
	"errors"
	"fmt"
	"guitarHetic/internal/domain/ehub" // S'assurer que l'import est correct
	"log"
	"net"
)

type Listener struct {
	conn       *net.UDPConn
	packetChan chan<- ehub.RawPacket // CORRECTION: Le type est bien ehub.RawPacket
}

func NewListener(port int, packetChan chan<- ehub.RawPacket) (*Listener, error) { // CORRECTION: Le type est bien ehub.RawPacket
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

// Start prend un contexte pour un arrêt propre.
func (l *Listener) Start(ctx context.Context) {
	go func() {
		// Lance une goroutine fille qui attend l'annulation du contexte.
		// Quand le contexte est annulé, elle ferme la connexion,
		// ce qui débloquera l'appel ReadFromUDP ci-dessous.
		go func() {
			<-ctx.Done()
			l.conn.Close() // Provoque la fin de la boucle ci-dessous.
		}()

		buffer := make([]byte, 20000)
		for {
			n, remoteAddr, err := l.conn.ReadFromUDP(buffer)
			if err != nil {
				// Vérifie si l'erreur est due à la fermeture volontaire de la connexion.
				if errors.Is(err, net.ErrClosed) {
					log.Println("Listener: Connexion fermée, arrêt de la goroutine d'écoute.")
					return // C'est le chemin de sortie normal.
				}
				// Pour les autres erreurs, on logue et on continue.
				log.Printf("Erreur de lecture UDP: %v", err)
				continue
			}

			// On ne traite le paquet que si la lecture a réussi.
			packetCopy := make([]byte, n)
			copy(packetCopy, buffer[:n])

			// Envoi sur le canal, avec une vérification de contexte pour ne pas rester bloqué.
			select {
			case l.packetChan <- ehub.RawPacket{ // CORRECTION: Le type est bien ehub.RawPacket
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
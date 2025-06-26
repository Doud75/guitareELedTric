// File: internal/application/processor/service.go
package processor

import (
	"log"
	"guitarHetic/internal/config"
	"guitarHetic/internal/domain/artnet"
	"guitarHetic/internal/domain/ehub"
)

// DestinationChannel est juste un alias pour le type de canal de sortie.
type DestinationChannel chan<- artnet.LEDMessage

// Service est notre objet Processor.
type Service struct {
	configIn <-chan *ehub.EHubConfigMsg
	updateIn <-chan *ehub.EHubUpdateMsg
	dest     DestinationChannel

	// PAS de table de routage pré-calculée ici !
	// À la place, on garde la configuration chargée du CSV.
	physicalConfig *config.Config
}

// NewService est le constructeur de notre Processor.
func NewService(
	configIn <-chan *ehub.EHubConfigMsg,
	updateIn <-chan *ehub.EHubUpdateMsg,
	dest DestinationChannel,
) *Service {
	return &Service{
		configIn:    configIn,
		updateIn:    updateIn,
		dest:        dest,
		// La configuration physique est vide au début.
		physicalConfig: nil,
	}
}

// Start démarre la boucle de travail du Processor.
func (s *Service) Start() {
	go func() {
		log.Println("Processor: Service démarré (mode simple, sans table pré-calculée).")

		for {
			select {
			// Cas 1: Un message de configuration arrive.
			// Pour l'instant, on l'utilise juste comme un signal pour charger notre propre config CSV.
			case <-s.configIn:
				log.Printf("Processor: Reçu un message CONFIG. Chargement de la configuration physique depuis le CSV...")
				
				// Charger la configuration physique (le plan de câblage).
				cfg, err := config.Load("internal/config/routing.csv")
				if err != nil {
					log.Printf("Processor: ERREUR, impossible de charger le fichier de routing: %v", err)
					s.physicalConfig = nil // S'assurer qu'on n'utilise pas une vieille config.
					continue
				}
				s.physicalConfig = cfg
				log.Println("Processor: Configuration physique chargée. Prêt à router les messages 'update'.")

			// Cas 2: Un message de mise à jour arrive.
			case updateMsg := <-s.updateIn:
				// Si on n'a pas encore chargé la config physique, on ne peut rien faire.
				if s.physicalConfig == nil {
					continue
				}

				// On prépare les buffers DMX pour chaque univers.
				frames := make(map[int]*[510]byte)
				
				// On parcourt chaque entité du message reçu.
				for _, entity := range updateMsg.Entities {
					
					// --- TRADUCTION "À LA VOLÉE" ---
					// Pour chaque entité, on doit trouver sa route.
					
					// 1. Chercher la règle de routage pour cet ID d'entité.
					// C'est l'étape lente que la table pré-calculée optimisera plus tard.
					var route *config.RoutingEntry
					for i := range s.physicalConfig.RoutingTable {
						if s.physicalConfig.RoutingTable[i].EntityID == int(entity.ID) {
							route = &s.physicalConfig.RoutingTable[i]
							break // On a trouvé, on arrête de chercher.
						}
					}
					
					// 2. Si on n'a pas trouvé de route, on passe à l'entité suivante.
					if route == nil {
						continue
					}
					
					// 3. On a une route ! On peut remplir le buffer DMX.
					
					// On récupère ou on crée le buffer pour l'univers de destination.
					if _, ok := frames[route.Universe]; !ok {
						frames[route.Universe] = new([510]byte)
					}
					
					// On écrit les couleurs à l'offset DMX calculé dans le loader.
					offset := route.DMXOffset
					if offset+2 < 510 {
						frames[route.Universe][offset+0] = entity.Red
						frames[route.Universe][offset+1] = entity.Green
						frames[route.Universe][offset+2] = entity.Blue
					}
				}

				// Maintenant, on envoie chaque buffer DMX modifié vers le Sender.
				for u, data := range frames {
					s.dest <- artnet.LEDMessage{Universe: u, Data: *data}
				}
			}
		}
	}()
}
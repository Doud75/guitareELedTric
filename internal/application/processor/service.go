// internal/application/processor/service.go
package processor

import (
	"log"
	"reflect"
	"sync" // Importé pour sync.Pool

	"guitarHetic/internal/config"
	"guitarHetic/internal/domain/artnet"
	"guitarHetic/internal/domain/ehub"
)

// Déclaration du pool au niveau du package.
// Il va stocker et réutiliser les maps de frames pour éviter des allocations constantes.
var frameMapPool = sync.Pool{
	New: func() interface{} {
		// Cette fonction est appelée par le pool quand il est vide et a besoin
		// d'un nouvel objet. Nous créons une nouvelle map et retournons son pointeur.
		m := make(map[int]*[512]byte)
		return &m
	},
}

// DestinationChannel est un alias pour plus de clarté.
type DestinationChannel chan<- artnet.LEDMessage

// FinalRouteInfo contient les informations pré-calculées pour un routage ultra-rapide.
type FinalRouteInfo struct {
	IsEnabled       bool
	TargetUniverse  int
	DMXBufferOffset int
}

// Service est le cœur du traitement logique.
type Service struct {
	configMsgIn        <-chan *ehub.EHubConfigMsg
	updateMsgIn        <-chan *ehub.EHubUpdateMsg
	PhysicalConfigIn   chan *config.Config
	dest               DestinationChannel
	routingTable       []FinalRouteInfo
	lastUsedConfigMsg  *ehub.EHubConfigMsg
	lastPhysicalConfig *config.Config
	// 🔥 NOUVEAU: Frame persistence - garde la dernière frame connue de chaque univers
	persistentFrames   map[int]*[512]byte
	framesMutex        sync.RWMutex // Protection pour l'accès concurrent aux frames persistantes
}

// NewService construit une nouvelle instance du service de processeur.
func NewService(
	configMsgIn <-chan *ehub.EHubConfigMsg,
	updateMsgIn <-chan *ehub.EHubUpdateMsg,
	dest DestinationChannel,
) (*Service, chan *config.Config) {

	physicalConfigChan := make(chan *config.Config)

	return &Service{
		configMsgIn:      configMsgIn,
		updateMsgIn:      updateMsgIn,
		PhysicalConfigIn: physicalConfigChan,
		dest:             dest,
		persistentFrames: make(map[int]*[512]byte), // 🔥 NOUVEAU: Initialisation des frames persistantes
	}, physicalConfigChan
}

// Start lance la goroutine principale du service.
func (s *Service) Start() {
	go func() {
		log.Println("Processor: Service démarré (mode optimisé, avec table pré-calculée, pooling et filtrage).")

		for {
			select {
			case newPhysicalConfig := <-s.PhysicalConfigIn:
				log.Println("Processor: Nouvelle configuration physique reçue. Mise à jour interne.")
				s.lastPhysicalConfig = newPhysicalConfig
				if s.lastUsedConfigMsg != nil {
					log.Println("   -> Reconstruction de la table de routage avec la nouvelle config physique.")
					s.buildRoutingTable(s.lastUsedConfigMsg, s.lastPhysicalConfig)
				}

			case newConfigMsg := <-s.configMsgIn:
				if s.lastUsedConfigMsg != nil && reflect.DeepEqual(s.lastUsedConfigMsg, newConfigMsg) {
					continue
				}

				log.Println("Processor: Nouvelle configuration eHuB détectée.")
				s.lastUsedConfigMsg = newConfigMsg
				if s.lastPhysicalConfig != nil {
					log.Println("   -> Reconstruction de la table de routage avec la nouvelle config eHuB.")
					s.buildRoutingTable(s.lastUsedConfigMsg, s.lastPhysicalConfig)
				}

			case updateMsg := <-s.updateMsgIn:
				if s.routingTable == nil {
					continue // On ne peut rien faire sans table de routage.
				}

				// --- OPTIMISATION #1: Utilisation du Pool de mémoire + Persistance ---
				// 1. On récupère une map pré-allouée du pool.
				framesPtr := frameMapPool.Get().(*map[int]*[512]byte)
				frames := *framesPtr

				// 2. TRÈS IMPORTANT: Vider la map avant utilisation, car elle a déjà servi.
				for k := range frames {
					delete(frames, k)
				}

				// 3. 🔥 PERSISTANCE: Copier les frames persistantes comme base
				s.framesMutex.RLock()
				for universe, persistentFrame := range s.persistentFrames {
					newFrame := new([512]byte)
					copy(newFrame[:], persistentFrame[:])
					frames[universe] = newFrame
				}
				s.framesMutex.RUnlock()

				// Boucle principale de traitement des entités du message update.
				for _, entity := range updateMsg.Entities {
					
					// --- OPTIMISATION #2: Filtrage du bruit (Thresholding) ---
					// Si les valeurs RVB sont très faibles, on les considère comme du bruit
					// et on les force à zéro pour stabiliser le signal de sortie.
					const noiseThreshold = 15 // Seuil de bruit, ajustable si besoin.
					if entity.Red < noiseThreshold && entity.Green < noiseThreshold && entity.Blue < noiseThreshold && entity.White < noiseThreshold {
						entity.Red, entity.Green, entity.Blue = 0, 0, 0
					}
					
					entityIndex := int(entity.ID)

					// Vérification des limites pour éviter un panic.
					if entityIndex >= len(s.routingTable) {
						continue
					}
					
					// On récupère les infos de routage pré-calculées. C'est une lecture très rapide.
					routeInfo := s.routingTable[entityIndex]
					
					if routeInfo.IsEnabled {
						// On vérifie si on a déjà un buffer DMX pour cet univers dans cette trame.
						targetFrame, ok := frames[routeInfo.TargetUniverse]
						if !ok {
							// Si ce n'est pas le cas, on en alloue un nouveau.
							targetFrame = new([512]byte)
							frames[routeInfo.TargetUniverse] = targetFrame
						}
						
						offset := routeInfo.DMXBufferOffset
						if offset+2 < 512 { // RGB seulement (3 canaux) pour avoir 170 entités max (170*3=510)
							targetFrame[offset+0] = entity.Red
							targetFrame[offset+1] = entity.Green
							targetFrame[offset+2] = entity.Blue
							// On ignore délibérément entity.White car l'écran LED est RGB, pas RGBW
						}
					}
				}

				// 🔥 PERSISTANCE: Sauvegarder les frames pour la prochaine fois
				s.framesMutex.Lock()
				for universe, frameData := range frames {
					if s.persistentFrames[universe] == nil {
						s.persistentFrames[universe] = new([512]byte)
					}
					copy(s.persistentFrames[universe][:], frameData[:])
				}
				s.framesMutex.Unlock()

				// On envoie tous les buffers DMX construits au Sender.
				for u, data := range frames {
					s.dest <- artnet.LEDMessage{Universe: u, Data: *data}
				}

				// 3. On remet la map dans le pool pour qu'elle soit réutilisée plus tard.
				frameMapPool.Put(framesPtr)
			}
		}
	}()
}

// buildRoutingTable pré-calcule toutes les informations de routage pour un accès instantané.
// Cette fonction est appelée rarement, donc son coût n'est pas critique.
func (s *Service) buildRoutingTable(eHubConfig *ehub.EHubConfigMsg, physicalConfig *config.Config) {
	// Étape 1: Transformer la liste de config physique en map pour un accès rapide (O(1)).
	physicalMap := make(map[int]config.RoutingEntry)
	for _, entry := range physicalConfig.RoutingTable {
		physicalMap[entry.EntityID] = entry
	}
	
	// Étape 2: Déterminer la taille maximale de notre table de routage.
	var maxEntityID uint16 = 0
	for _, r := range eHubConfig.Ranges {
		if r.EntityEnd > maxEntityID {
			maxEntityID = r.EntityEnd
		}
	}
	
	// Étape 3: Allouer et remplir la table de routage finale.
	newTable := make([]FinalRouteInfo, maxEntityID+1)
	log.Printf("Processor: Allocation d'une nouvelle table de routage pour %d entités max.", maxEntityID+1)
	
	for _, eHubRange := range eHubConfig.Ranges {
		for entityID := eHubRange.EntityStart; entityID <= eHubRange.EntityEnd; entityID++ {
			// On cherche la correspondance dans la config physique.
			if physicalRoute, ok := physicalMap[int(entityID)]; ok {
				// Si une correspondance est trouvée, on remplit notre table avec
				// les informations finales et pré-calculées.
				newTable[entityID] = FinalRouteInfo{
					IsEnabled:       true,
					TargetUniverse:  physicalRoute.Universe,
					DMXBufferOffset: physicalRoute.DMXOffset,
				}
			}
			// Si aucune correspondance n'est trouvée, la valeur par défaut (IsEnabled: false) est correcte.
		}
	}
	
	// Étape 4: Remplacer l'ancienne table par la nouvelle.
	s.routingTable = newTable
	log.Printf("Processor: Nouvelle table de routage construite et active.")
}
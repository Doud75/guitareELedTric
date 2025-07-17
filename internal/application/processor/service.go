// internal/application/processor/service.go
package processor

import (
    "guitarHetic/internal/ui"
    "log"
    "reflect"
    "sync" // Importé pour sync.Pool

    "guitarHetic/internal/config"
    "guitarHetic/internal/domain/artnet"
    "guitarHetic/internal/domain/ehub"
)

type DestinationChannel chan<- artnet.LEDMessage

type FinalRouteInfo struct {
	IsEnabled       bool
	TargetIP        string
	TargetUniverse  int
	DMXBufferOffset int
}

type Service struct {
	configMsgIn        <-chan *ehub.EHubConfigMsg
	updateMsgIn        <-chan *ehub.EHubUpdateMsg
	PhysicalConfigIn   chan *config.Config
	dest               DestinationChannel
	routingTable       []FinalRouteInfo
	lastUsedConfigMsg  *ehub.EHubConfigMsg
	lastPhysicalConfig *config.Config
	persistentStates   map[int]*[512]byte
	stateMutex         sync.Mutex
    monitorOut         chan<- *ui.UniverseMonitorData
}

func NewService(
    configMsgIn <-chan *ehub.EHubConfigMsg,
    updateMsgIn <-chan *ehub.EHubUpdateMsg,
    dest DestinationChannel,
    monitorOut chan<- *ui.UniverseMonitorData,
) (*Service, chan *config.Config) {
	physicalConfigChan := make(chan *config.Config)
	return &Service{
		configMsgIn:      configMsgIn,
		updateMsgIn:      updateMsgIn,
		PhysicalConfigIn: physicalConfigChan,
		dest:             dest,
		persistentStates: make(map[int]*[512]byte),
        monitorOut:       monitorOut,
	}, physicalConfigChan
}

func (s *Service) Start() {
	go func() {
		log.Println("Processor: Service démarré (mode stateful optimisé).")
		for {
			select {
			case newPhysicalConfig := <-s.PhysicalConfigIn:
				s.handleNewPhysicalConfig(newPhysicalConfig)
			case newConfigMsg := <-s.configMsgIn:
				s.handleNewEHubConfig(newConfigMsg)
			case updateMsg := <-s.updateMsgIn:
				s.processUpdate(updateMsg)
			}
		}
	}()
}

// processUpdate met à jour l'état persistant et envoie UNIQUEMENT les univers modifiés.
func (s *Service) processUpdate(updateMsg *ehub.EHubUpdateMsg) {
	if s.routingTable == nil || s.lastPhysicalConfig == nil {
		return
	}

	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

	modifiedUniverses := make(map[int]struct{})

	for _, entity := range updateMsg.Entities {
		const noiseThreshold = 15
		if entity.Red < noiseThreshold && entity.Green < noiseThreshold && entity.Blue < noiseThreshold {
			entity.Red, entity.Green, entity.Blue = 0, 0, 0
		}

		entityIndex := int(entity.ID)
		if entityIndex >= len(s.routingTable) {
			continue
		}

		routeInfo := s.routingTable[entityIndex]
		if !routeInfo.IsEnabled {
			continue
		}

		universe := routeInfo.TargetUniverse
		offset := routeInfo.DMXBufferOffset

		// S'assurer que le buffer pour cet univers existe dans notre état persistant.
		if _, ok := s.persistentStates[universe]; !ok {
			s.persistentStates[universe] = new([512]byte)
		}

		// On met à jour directement l'état persistant. PAS de copie massive.
		if offset+2 < 512 {
			s.persistentStates[universe][offset+0] = entity.Red
			s.persistentStates[universe][offset+1] = entity.Green
			s.persistentStates[universe][offset+2] = entity.Blue
			modifiedUniverses[universe] = struct{}{} // On note que cet univers a changé.
		}
	}

	// On envoie une copie de l'état complet des SEULS univers qui ont été modifiés.
	for universe := range modifiedUniverses {
		ip, ok := s.lastPhysicalConfig.UniverseIP[universe]
		if !ok {
			continue
		}
		if frameData := s.persistentStates[universe]; frameData != nil {
			// On envoie bien une copie pour que le Sender ne puisse pas modifier notre état.
			dataToSend := *frameData
			s.dest <- artnet.LEDMessage{
				DestinationIP: ip,
				Universe:      universe,
				Data:          dataToSend,
			}

            relevantEntities := make([]ehub.EHubEntityState, 0)


            for _, entity := range updateMsg.Entities {
                entityIndex := int(entity.ID)
                if entityIndex < len(s.routingTable) {
                    // On vérifie si l'entité appartient bien à l'univers 'u' actuel.
                    if s.routingTable[entityIndex].TargetUniverse == universe {
                        relevantEntities = append(relevantEntities, entity)
                    }
                }
            }

            // 3. On crée le message de monitoring avec la liste FILTRÉE.
            monitorData := &ui.UniverseMonitorData{
                UniverseID: universe,
                InputState: relevantEntities, // On utilise la petite liste filtrée !
                OutputDMX:  dataToSend,
            }

            // 4. On envoie au canal de monitoring.
            select {
            case s.monitorOut <- monitorData:
            default:
                log.Println("MONITOR_WARN: Le canal de monitoring UI est plein, un paquet est ignoré.")
            }
		}
	}
}

// Les fonctions de gestion de config et de build de la table de routage
// sont extraites pour plus de clarté.
func (s *Service) handleNewPhysicalConfig(cfg *config.Config) {
	log.Println("Processor: Nouvelle configuration physique reçue.")
	s.lastPhysicalConfig = cfg
	if s.lastUsedConfigMsg != nil {
		s.buildRoutingTable(s.lastUsedConfigMsg, s.lastPhysicalConfig)
	}
}

func (s *Service) handleNewEHubConfig(msg *ehub.EHubConfigMsg) {
	if s.lastUsedConfigMsg != nil && reflect.DeepEqual(s.lastUsedConfigMsg, msg) {
		return
	}
	log.Println("Processor: Nouvelle configuration eHuB détectée.")
	s.lastUsedConfigMsg = msg
	if s.lastPhysicalConfig != nil {
		s.buildRoutingTable(s.lastUsedConfigMsg, s.lastPhysicalConfig)
	}
}

// buildRoutingTable pré-calcule toutes les informations de routage, y compris l'IP de destination.
func (s *Service) buildRoutingTable(eHubConfig *ehub.EHubConfigMsg, physicalConfig *config.Config) {
    physicalMap := make(map[int]config.RoutingEntry)
    for _, entry := range physicalConfig.RoutingTable {
        physicalMap[entry.EntityID] = entry
    }

    var maxEntityID uint16 = 0
    for _, r := range eHubConfig.Ranges {
        if r.EntityEnd > maxEntityID {
            maxEntityID = r.EntityEnd
        }
    }

    newTable := make([]FinalRouteInfo, maxEntityID+1)
    log.Printf("Processor: Allocation d'une nouvelle table de routage pour %d entités max.", maxEntityID+1)

    for _, eHubRange := range eHubConfig.Ranges {
        for entityID := eHubRange.EntityStart; entityID <= eHubRange.EntityEnd; entityID++ {
            if physicalRoute, ok := physicalMap[int(entityID)]; ok {
                // On stocke l'IP de destination directement dans la table de routage
                // pour un accès instantané.
                newTable[entityID] = FinalRouteInfo{
                    IsEnabled:       true,
                    TargetIP:        physicalRoute.IP, // L'IP est maintenant pré-calculée.
                    TargetUniverse:  physicalRoute.Universe,
                    DMXBufferOffset: physicalRoute.DMXOffset,
                }
            }
        }
    }

    s.routingTable = newTable
    log.Printf("Processor: Nouvelle table de routage construite et active.")
}

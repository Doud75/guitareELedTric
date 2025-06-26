// File: internal/application/processor/service.go
package processor

import (
	"log"
	"reflect" // <<< ON A BESOIN DE CET OUTIL POUR COMPARER

	"guitarHetic/internal/config"
	"guitarHetic/internal/domain/artnet"
	"guitarHetic/internal/domain/ehub"
)

type DestinationChannel chan<- artnet.LEDMessage

type FinalRouteInfo struct {
	IsEnabled       bool
	TargetUniverse  int
	DMXBufferOffset int
}

// La struct Service a maintenant un champ pour se souvenir de la dernière config.
type Service struct {
	configIn       <-chan *ehub.EHubConfigMsg
	updateIn       <-chan *ehub.EHubUpdateMsg
	dest           DestinationChannel
	lastUsedConfig *ehub.EHubConfigMsg // <<< CHAMP AJOUTÉ
	routingTable   []FinalRouteInfo
}

// Le constructeur doit initialiser le nouveau champ à nil.
func NewService(
	configIn <-chan *ehub.EHubConfigMsg,
	updateIn <-chan *ehub.EHubUpdateMsg,
	dest DestinationChannel,
) *Service {
	return &Service{
		configIn:       configIn,
		updateIn:       updateIn,
		dest:           dest,
		lastUsedConfig: nil, // Important: on n'a encore rien utilisé.
		routingTable:   nil,
	}
}

func (s *Service) Start() {
	go func() {
		log.Println("Processor: Service démarré (mode optimisé, avec table pré-calculée).")

		for {
			select {
			case newConfigMsg := <-s.configIn: 

				if s.lastUsedConfig != nil && reflect.DeepEqual(s.lastUsedConfig, newConfigMsg) {
					continue
				}

				log.Println("Processor: Nouvelle configuration eHuB détectée. Reconstruction de la table de routage...")
				
				physicalConfig, err := config.Load("internal/config/routing.csv")
				if err != nil {
					log.Printf("Processor: ERREUR, impossible de charger le fichier de routing: %v", err)
					s.routingTable = nil
					continue
				}
				
				s.buildRoutingTable(newConfigMsg, physicalConfig)

				s.lastUsedConfig = newConfigMsg

			case updateMsg := <-s.updateIn:
				if s.routingTable == nil {
					continue
				}

				frames := make(map[int]*[512]byte)

				for _, entity := range updateMsg.Entities {
					entityIndex := int(entity.ID)

					if entityIndex >= len(s.routingTable) {
						continue
					}
					
					routeInfo := s.routingTable[entityIndex]
					
					if routeInfo.IsEnabled {
						if _, ok := frames[routeInfo.TargetUniverse]; !ok {
							frames[routeInfo.TargetUniverse] = new([512]byte)
						}
						
						offset := routeInfo.DMXBufferOffset
						if offset+2 < 512 {
							frames[routeInfo.TargetUniverse][offset+0] = entity.Red
							frames[routeInfo.TargetUniverse][offset+1] = entity.Green
							frames[routeInfo.TargetUniverse][offset+2] = entity.Blue
						}
					}
				}

				for u, data := range frames {
					s.dest <- artnet.LEDMessage{Universe: u, Data: *data}
				}
			}
		}
	}()
}


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
	
	newTable := make([]FinalRouteInfo, maxEntityID + 1)
	log.Printf("Processor: Allocation d'une nouvelle table de routage pour %d entités max.", maxEntityID+1)
	
	for _, eHubRange := range eHubConfig.Ranges {
		for entityID := eHubRange.EntityStart; entityID <= eHubRange.EntityEnd; entityID++ {
			if physicalRoute, ok := physicalMap[int(entityID)]; ok {
				newTable[entityID] = FinalRouteInfo{
					IsEnabled:       true,
					TargetUniverse:  physicalRoute.Universe,
					DMXBufferOffset: physicalRoute.DMXOffset,
				}
			}
		}
	}
	
	s.routingTable = newTable
	log.Printf("Processor: Nouvelle table de routage construite et active.")
}
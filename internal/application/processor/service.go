package processor

import (
	"log"
	"reflect" 
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

type Service struct {
	configMsgIn       <-chan *ehub.EHubConfigMsg
	updateMsgIn       <-chan *ehub.EHubUpdateMsg
	PhysicalConfigIn chan *config.Config 
	dest           DestinationChannel
	routingTable   []FinalRouteInfo
	lastUsedConfigMsg *ehub.EHubConfigMsg 
	lastPhysicalConfig *config.Config
}

func NewService(
	configMsgIn <-chan *ehub.EHubConfigMsg,
	updateMsgIn <-chan *ehub.EHubUpdateMsg,
	dest DestinationChannel,
) (*Service, chan *config.Config) { 
	
	physicalConfigChan := make(chan *config.Config)

	return &Service{
		configMsgIn:       configMsgIn,
		updateMsgIn:       updateMsgIn,
		PhysicalConfigIn: physicalConfigChan, 
		dest:           dest,
	}, physicalConfigChan 
}

func (s *Service) Start() {
	go func() {
		log.Println("Processor: Service démarré (mode optimisé, avec table pré-calculée).")

		for {
			select {
			
			case newPhysicalConfig := <-s.PhysicalConfigIn:
				log.Println("Processor: Nouvelle configuration physique reçue. Mise à jour interne.")
				s.lastPhysicalConfig = newPhysicalConfig
				// Si on a déjà une config eHuB, on reconstruit la table tout de suite.
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
				// Si on a déjà une config physique, on reconstruit la table.
				if s.lastPhysicalConfig != nil {
					log.Println("   -> Reconstruction de la table de routage avec la nouvelle config eHuB.")
					s.buildRoutingTable(s.lastUsedConfigMsg, s.lastPhysicalConfig)
				}


			case updateMsg := <-s.updateMsgIn:
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
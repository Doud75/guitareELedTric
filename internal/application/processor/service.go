// internal/application/processor/service.go
package processor

import (
    "guitarHetic/internal/config"
    "guitarHetic/internal/domain/artnet"
    "guitarHetic/internal/domain/ehub"
    "guitarHetic/internal/ui"
    "log"
    "reflect"
    "sync"
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
    patchMap           map[int]map[int][]int
    isPatchingActive   bool
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
        patchMap:         nil,
        isPatchingActive: false,
    }, physicalConfigChan
}

func (s *Service) SetPatchMap(p map[int]map[int][]int) {
    s.stateMutex.Lock()
    defer s.stateMutex.Unlock()
    s.patchMap = p
    log.Println("Processor: Nouvelle Patch Map appliquée.")
}

func (s *Service) SetPatchingActive(active bool) {
    s.stateMutex.Lock()
    defer s.stateMutex.Unlock()
    s.isPatchingActive = active
    if active {
        log.Println("Processor: Patching activé.")
    } else {
        log.Println("Processor: Patching désactivé.")
    }
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

// MODIFICATION: La logique de patch est complétée avec l'effacement de la source.
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

        if _, ok := s.persistentStates[universe]; !ok {
            s.persistentStates[universe] = new([512]byte)
        }

        if offset+2 < 512 {
            s.persistentStates[universe][offset+0] = entity.Red
            s.persistentStates[universe][offset+1] = entity.Green
            s.persistentStates[universe][offset+2] = entity.Blue
            modifiedUniverses[universe] = struct{}{}
        }
    }

    for universe := range modifiedUniverses {
        ip, ok := s.lastPhysicalConfig.UniverseIP[universe]
        if !ok {
            continue
        }
        if originalBuffer := s.persistentStates[universe]; originalBuffer != nil {
            bufferToSend := *originalBuffer

            if s.isPatchingActive {
                if patchForThisUniverse, ok := s.patchMap[universe]; ok {
                    patchedBuffer := *originalBuffer

                    for sourceChannel, destinationChannels := range patchForThisUniverse {
                        sourceIndex := (sourceChannel - 1) * 3
                        if sourceIndex+2 >= 512 {
                            continue
                        }

                        // Lecture depuis le buffer ORIGINAL
                        valR := originalBuffer[sourceIndex]
                        valG := originalBuffer[sourceIndex+1]
                        valB := originalBuffer[sourceIndex+2]

                        // Écriture sur toutes les destinations
                        for _, destChannel := range destinationChannels {
                            destIndex := (destChannel - 1) * 3
                            if destIndex+2 >= 512 {
                                continue
                            }
                            patchedBuffer[destIndex] = valR
                            patchedBuffer[destIndex+1] = valG
                            patchedBuffer[destIndex+2] = valB
                        }

                        // --- MODIFICATION CLÉ ---
                        // Après avoir copié la valeur, on éteint la source dans le buffer patché.
                        // On met à zéro les 3 canaux (RVB) de la source.
                        patchedBuffer[sourceIndex] = 0
                        patchedBuffer[sourceIndex+1] = 0
                        patchedBuffer[sourceIndex+2] = 0
                        // --- FIN DE LA MODIFICATION ---
                    }
                    bufferToSend = patchedBuffer
                }
            }

            s.dest <- artnet.LEDMessage{
                DestinationIP: ip,
                Universe:      universe,
                Data:          bufferToSend,
            }

            relevantEntities := make([]ehub.EHubEntityState, 0)
            for _, entity := range updateMsg.Entities {
                entityIndex := int(entity.ID)
                if entityIndex < len(s.routingTable) {
                    if s.routingTable[entityIndex].TargetUniverse == universe {
                        relevantEntities = append(relevantEntities, entity)
                    }
                }
            }

            monitorData := &ui.UniverseMonitorData{
                UniverseID: universe,
                InputState: relevantEntities,
                OutputDMX:  bufferToSend,
            }

            select {
            case s.monitorOut <- monitorData:
            default:
                log.Println("MONITOR_WARN: Le canal de monitoring UI est plein, un paquet est ignoré.")
            }
        }
    }
}

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
                newTable[entityID] = FinalRouteInfo{
                    IsEnabled:       true,
                    TargetIP:        physicalRoute.IP,
                    TargetUniverse:  physicalRoute.Universe,
                    DMXBufferOffset: physicalRoute.DMXOffset,
                }
            }
        }
    }

    s.routingTable = newTable
    log.Printf("Processor: Nouvelle table de routage construite et active.")
}

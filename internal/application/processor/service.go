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
        m := make(map[int]*[512]byte)
        return &m
    },
}

// DestinationChannel est un alias pour le canal de sortie vers le sender.
// Il transporte maintenant le message enrichi.
type DestinationChannel chan<- artnet.LEDMessage

// FinalRouteInfo contient les informations pré-calculées pour un routage ultra-rapide.
// Elle inclut maintenant l'adresse IP de destination.
type FinalRouteInfo struct {
    IsEnabled       bool
    TargetIP        string // L'adresse IP de destination pour cette entité.
    TargetUniverse  int
    DMXBufferOffset int
}

// Service est le cœur du traitement logique. Il ne gère plus le cycle de vie du sender.
type Service struct {
    configMsgIn        <-chan *ehub.EHubConfigMsg
    updateMsgIn        <-chan *ehub.EHubUpdateMsg
    PhysicalConfigIn   chan *config.Config
    dest               DestinationChannel
    routingTable       []FinalRouteInfo
    lastUsedConfigMsg  *ehub.EHubConfigMsg
    lastPhysicalConfig *config.Config
    persistentFrames   map[int]*[512]byte
    framesMutex        sync.RWMutex
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
        persistentFrames: make(map[int]*[512]byte),
    }, physicalConfigChan
}

// Start lance la goroutine principale du service.
func (s *Service) Start() {
    go func() {
        log.Println("Processor: Service démarré.")

        for {
            select {
            case newPhysicalConfig := <-s.PhysicalConfigIn:
                log.Println("Processor: Nouvelle configuration physique reçue. Reconstruction de la table de routage...")
                s.lastPhysicalConfig = newPhysicalConfig
                if s.lastUsedConfigMsg != nil {
                    s.buildRoutingTable(s.lastUsedConfigMsg, s.lastPhysicalConfig)
                }

            case newConfigMsg := <-s.configMsgIn:
                if s.lastUsedConfigMsg != nil && reflect.DeepEqual(s.lastUsedConfigMsg, newConfigMsg) {
                    continue
                }

                log.Println("Processor: Nouvelle configuration eHuB détectée.")
                s.lastUsedConfigMsg = newConfigMsg
                if s.lastPhysicalConfig != nil {
                    s.buildRoutingTable(s.lastUsedConfigMsg, s.lastPhysicalConfig)
                }

            case updateMsg := <-s.updateMsgIn:
                if s.routingTable == nil || s.lastPhysicalConfig == nil {
                    continue // On ne peut rien faire sans table de routage et config physique.
                }

                framesPtr := frameMapPool.Get().(*map[int]*[512]byte)
                frames := *framesPtr
                for k := range frames {
                    delete(frames, k)
                }

                s.framesMutex.RLock()
                for universe, persistentFrame := range s.persistentFrames {
                    newFrame := new([512]byte)
                    copy(newFrame[:], persistentFrame[:])
                    frames[universe] = newFrame
                }
                s.framesMutex.RUnlock()

                for _, entity := range updateMsg.Entities {
                    const noiseThreshold = 15
                    if entity.Red < noiseThreshold && entity.Green < noiseThreshold && entity.Blue < noiseThreshold && entity.White < noiseThreshold {
                        entity.Red, entity.Green, entity.Blue = 0, 0, 0
                    }

                    entityIndex := int(entity.ID)
                    if entityIndex >= len(s.routingTable) {
                        continue
                    }

                    routeInfo := s.routingTable[entityIndex]

                    if routeInfo.IsEnabled {
                        targetFrame, ok := frames[routeInfo.TargetUniverse]
                        if !ok {
                            targetFrame = new([512]byte)
                            frames[routeInfo.TargetUniverse] = targetFrame
                        }

                        offset := routeInfo.DMXBufferOffset
                        if offset+2 < 512 {
                            targetFrame[offset+0] = entity.Red
                            targetFrame[offset+1] = entity.Green
                            targetFrame[offset+2] = entity.Blue
                        }
                    }
                }

                s.framesMutex.Lock()
                for universe, frameData := range frames {
                    if s.persistentFrames[universe] == nil {
                        s.persistentFrames[universe] = new([512]byte)
                    }
                    copy(s.persistentFrames[universe][:], frameData[:])
                }
                s.framesMutex.Unlock()

                // On envoie tous les buffers DMX construits au Sender,
                // en les enrichissant avec l'adresse IP de destination.
                for u, data := range frames {
                    ip, ok := s.lastPhysicalConfig.UniverseIP[u]
                    if !ok {
                        log.Printf("Processor: IP non trouvée pour l'univers %d dans la configuration, paquet ignoré.", u)
                        continue
                    }

                    s.dest <- artnet.LEDMessage{
                        DestinationIP: ip, // On ajoute l'IP de destination au message.
                        Universe:      u,
                        Data:          *data,
                    }
                }

                frameMapPool.Put(framesPtr)
            }
        }
    }()
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

package simulator

import (
    "context"
    "fmt"
    "guitarHetic/internal/config"
    "guitarHetic/internal/domain/ehub"
    "log"
    "math"
    "slices"
    "sync"
    "time"
)

type Faker struct {
    updateOut    chan<- *ehub.EHubUpdateMsg
    configOut    chan<- *ehub.EHubConfigMsg
    modeSwitch   chan<- bool
    config       *config.Config
    allEntityIDs []uint16

    mu              sync.Mutex
    cancelAnimation context.CancelFunc
}

func NewFaker(updateOut chan<- *ehub.EHubUpdateMsg, configOut chan<- *ehub.EHubConfigMsg, modeSwitch chan<- bool, cfg *config.Config) *Faker {
    var entityIDs []uint16

    if cfg != nil {
        for _, entry := range cfg.RoutingTable {
            entityIDs = append(entityIDs, uint16(entry.EntityID))
        }
        slices.Sort(entityIDs)
        log.Printf("Faker: Initialisé avec %d entités.", len(entityIDs))
    } else {
        log.Println("Faker: Initialisé sans configuration (en attente de chargement).")
    }

    return &Faker{
        updateOut:    updateOut,
        configOut:    configOut,
        modeSwitch:   modeSwitch,
        config:       cfg,
        allEntityIDs: entityIDs,
    }
}

func (f *Faker) sendStaticPattern(entities []ehub.EHubEntityState) {
    f.modeSwitch <- true
    f.sendInitialConfig()
    updateMsg := &ehub.EHubUpdateMsg{
        Universe: 0,
        Entities: entities,
    }
    f.updateOut <- updateMsg
}

func (f *Faker) SendTestPattern(command string, color ...byte) {
    f.StopAnimation()

    switch command {
    case "white":
        f.sendSolidColor(255, 255, 255, 0)
    case "red":
        f.sendSolidColor(30, 0, 0, 0)
    case "green":
        f.sendSolidColor(0, 30, 0, 0)
    case "blue":
        f.sendSolidColor(0, 0, 30, 0)
    case "black", "off":
        f.sendSolidColor(0, 0, 0, 0)
    case "custom":
        if len(color) == 4 {
            log.Printf("[FAKER] Commande CustomColor: R:%d G:%d B:%d W:%d", color[0], color[1], color[2], color[3])
            f.sendSolidColor(color[0], color[1], color[2], color[3])
        } else {
            log.Println("[FAKER] Erreur: commande 'custom' sans les 4 valeurs de couleur.")
        }

    case "animation":
        f.StartWaveAnimation()

    case "stop":
        f.sendSolidColor(0, 0, 0, 0)
    default:
        fmt.Printf("Unknown pattern '%s'.\n", command)
    }
}

func (f *Faker) sendSolidColor(r, g, b, w byte) {
    entities := make([]ehub.EHubEntityState, len(f.allEntityIDs))
    for i, entityID := range f.allEntityIDs {
        entities[i] = ehub.EHubEntityState{ID: entityID, Red: r, Green: g, Blue: b, White: w}
    }
    f.sendStaticPattern(entities)
}

func (f *Faker) StartWaveAnimation() {
    f.mu.Lock()
    if f.cancelAnimation != nil {
        f.mu.Unlock()
        return
    }

    ctx, cancel := context.WithCancel(context.Background())
    f.cancelAnimation = cancel
    f.mu.Unlock()

    f.modeSwitch <- true

    go func() {
        defer func() {
            f.mu.Lock()
            f.cancelAnimation = nil
            f.mu.Unlock()
            log.Println("Faker: Goroutine d'animation terminée.")
        }()

        log.Println("Faker: Démarrage de l'animation de vague.")
        f.sendInitialConfig()

        ticker := time.NewTicker(50 * time.Millisecond)
        defer ticker.Stop()

        position := 0.0

        for {
            select {
            case <-ticker.C:
                position += 0.02
                if position > 1.0 {
                    position = 0.0
                }
                entities := f.calculateWaveFrame(position, 0.3, 255, 100, 0)
                f.updateOut <- &ehub.EHubUpdateMsg{Universe: 0, Entities: entities}

            case <-ctx.Done():
                return
            }
        }
    }()
}

func (f *Faker) StopAnimation() {
    f.mu.Lock()
    if f.cancelAnimation != nil {
        log.Println("Faker: Signal d'arrêt envoyé à l'animation.")
        f.cancelAnimation()
        f.cancelAnimation = nil
    }
    f.mu.Unlock()
}

func (f *Faker) Stop() {
    f.SwitchToLiveMode()
}

func (f *Faker) SwitchToLiveMode() {
    f.StopAnimation()
    log.Println("Faker: Retour au mode LIVE (écoute eHub).")
    f.modeSwitch <- false
}

func (f *Faker) sendInitialConfig() {
    if len(f.allEntityIDs) == 0 {
        return
    }
    configMsg := &ehub.EHubConfigMsg{
        Universe: 0,
        Ranges: []ehub.EHubConfigRange{{
            SextuorStart: 0,
            EntityStart:  f.allEntityIDs[0],
            SextuorEnd:   uint16(len(f.allEntityIDs) - 1),
            EntityEnd:    f.allEntityIDs[len(f.allEntityIDs)-1],
        }},
    }
    f.configOut <- configMsg
}

func (f *Faker) calculateWaveFrame(position, width float64, r, g, b byte) []ehub.EHubEntityState {
    entities := make([]ehub.EHubEntityState, len(f.allEntityIDs))
    totalEntities := len(f.allEntityIDs)
    for i, entityID := range f.allEntityIDs {
        entityPos := float64(i) / float64(totalEntities-1)
        distance := math.Abs(entityPos - position)
        var intensity float64
        if distance <= width/2 {
            intensity = 1.0 - (distance / (width / 2))
        }
        entities[i] = ehub.EHubEntityState{
            ID:    entityID,
            Red:   byte(float64(r) * intensity),
            Green: byte(float64(g) * intensity),
            Blue:  byte(float64(b) * intensity),
        }
    }
    return entities
}

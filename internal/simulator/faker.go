// internal/simulator/faker.go
package simulator

import (
	"context"
	"fmt"
	"guitarHetic/internal/config"
	"guitarHetic/internal/domain/ehub"
	"log"
	"math"
	"sync"
	"time"
	"slices"
)


const maxEntitiesPerUpdate = 1024

type Faker struct {
	updateOut    chan<- *ehub.EHubUpdateMsg
	configOut    chan<- *ehub.EHubConfigMsg
	modeSwitch   chan<- bool
	config       *config.Config
	allEntityIDs []uint16
	mu              sync.Mutex
	cancelCurrentOp context.CancelFunc
}

// NewFaker initialise le Faker.
func NewFaker(updateOut chan<- *ehub.EHubUpdateMsg, configOut chan<- *ehub.EHubConfigMsg, modeSwitch chan<- bool, cfg *config.Config) *Faker {
	var entityIDs []uint16
	for _, entry := range cfg.RoutingTable {
		entityIDs = append(entityIDs, uint16(entry.EntityID))
	}
	slices.Sort(entityIDs)

	log.Printf("Faker: Initialisé avec %d entités.", len(entityIDs))
	return &Faker{
		updateOut:    updateOut,
		configOut:    configOut,
		modeSwitch:   modeSwitch,
		config:       cfg,
		allEntityIDs: entityIDs,
	}
}

func (f *Faker) stopCurrentOperation() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cancelCurrentOp != nil {
		f.cancelCurrentOp()
		f.cancelCurrentOp = nil
		log.Println("Faker: Opération précédente arrêtée.")
	}
}

func (f *Faker) sendFragmented(entities []ehub.EHubEntityState) {
	if len(entities) == 0 {
		return
	}
	f.sendInitialConfig()

	for i := 0; i < len(entities); i += maxEntitiesPerUpdate {
		end := i + maxEntitiesPerUpdate
		if end > len(entities) {
			end = len(entities)
		}
		chunk := entities[i:end]
		updateMsg := &ehub.EHubUpdateMsg{Universe: 0, Entities: chunk}
		f.updateOut <- updateMsg
	}
}

func (f *Faker) SwitchToLiveMode() {
	f.stopCurrentOperation()
	log.Println("Faker: Retour au mode LIVE.")
	f.modeSwitch <- false
}
func (f *Faker) SendCustomColor(r, g, b, w byte) {
    // MODIFICATION DU LOG: Ajoutons un marqueur clair
    log.Printf("[FAKER] Commande CustomColor reçue: R:%d G:%d B:%d W:%d", r, g, b, w)
	f.stopCurrentOperation()
	log.Println("[FAKER] Envoi du signal pour passer en mode FAKER (true) à l'aiguilleur...")
	f.modeSwitch <- true
	f.SendSolidColor(r, g, b, w)
}

// SendTestPattern est le point d'entrée pour les commandes de l'UI.
func (f *Faker) SendTestPattern(command string) {
	f.stopCurrentOperation()
	if command != "help" && command != "stop" {
		f.modeSwitch <- true
	}

	switch command {
	case "white":
		f.SendSolidColor(255, 255, 255, 0)
	case "pure-white":
		f.SendSolidColor(0, 0, 0, 255)
	case "red":
		f.SendSolidColor(30, 0, 0, 0)
	case "green":
		f.SendSolidColor(0, 30, 0, 0)
	case "blue":
		f.SendSolidColor(0, 0, 30, 0)
	case "black", "off":
		f.SendSolidColor(0, 0, 0, 0)
	case "gradient":
		f.SendGradient(255, 0, 0, 0, 0, 255)
	case "animation":
		f.StartWaveAnimation()
	case "stop":
		f.SwitchToLiveMode()
	default:
		fmt.Printf("Unknown pattern '%s'.\n", command)
		f.SwitchToLiveMode()
	}
}

func (f *Faker) Stop() {
	f.SwitchToLiveMode()
}


// SendSolidColor lance une goroutine qui envoie la couleur en boucle jusqu'à ce qu'elle soit annulée.
func (f *Faker) SendSolidColor(r, g, b, w byte) {
	entities := make([]ehub.EHubEntityState, len(f.allEntityIDs))
	for i, entityID := range f.allEntityIDs {
		entities[i] = ehub.EHubEntityState{ID: entityID, Red: r, Green: g, Blue: b, White: w}
	}

	f.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	f.cancelCurrentOp = cancel
	f.mu.Unlock()

	go func() {
		ticker := time.NewTicker(40 * time.Millisecond) // ~25 FPS
		defer ticker.Stop()
		log.Printf("Faker: Démarrage de l'envoi en boucle de la couleur unie.")

		for {
			select {
			case <-ticker.C:
				f.sendFragmented(entities)
			case <-ctx.Done():
				log.Println("Faker: Arrêt de l'envoi de la couleur unie.")
				return
			}
		}
	}()
}

// SendGradient envoie une seule fois un état de dégradé.
func (f *Faker) SendGradient(startR, startG, startB, endR, endG, endB byte) {
	entities := make([]ehub.EHubEntityState, len(f.allEntityIDs))
	for i, entityID := range f.allEntityIDs {
		progress := float64(i) / float64(len(f.allEntityIDs)-1)
		r := byte(float64(startR) + progress*float64(int(endR)-int(startR)))
		g := byte(float64(startG) + progress*float64(int(endG)-int(startG)))
		b := byte(float64(startB) + progress*float64(int(endB)-int(startB)))
		entities[i] = ehub.EHubEntityState{ID: entityID, Red: r, Green: g, Blue: b}
	}
	f.sendFragmented(entities)
	// Après l'envoi, on revient au live car c'est un état non-répétitif.
	go func() {
		time.Sleep(100 * time.Millisecond) // Laisse le temps au dernier paquet d'être traité
		f.SwitchToLiveMode()
	}()
}

// StartWaveAnimation lance une animation en boucle.
func (f *Faker) StartWaveAnimation() {
	f.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	f.cancelCurrentOp = cancel
	f.mu.Unlock()

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		position := 0.0

		log.Println("Faker: Démarrage de l'animation de vague.")
		for {
			select {
			case <-ticker.C:
				position += 0.02
				if position > 1.0 {
					position = 0.0
				}
				entities := f.calculateWaveFrame(position, 0.3, 255, 100, 0)
				f.sendFragmented(entities)

			case <-ctx.Done():
				log.Println("Faker: Arrêt de l'animation de vague.")
				// On envoie un dernier message noir pour nettoyer l'écran.
				blackEntities := make([]ehub.EHubEntityState, len(f.allEntityIDs))
				for i, id := range f.allEntityIDs {
					blackEntities[i] = ehub.EHubEntityState{ID: id}
				}
				f.sendFragmented(blackEntities)
				return
			}
		}
	}()
}

func (f *Faker) sendInitialConfig() {
	if len(f.allEntityIDs) == 0 {
		return
	}
	configMsg := &ehub.EHubConfigMsg{
		Universe: 0,
		Ranges: []ehub.EHubConfigRange{{
			SextuorStart: 0, EntityStart: f.allEntityIDs[0],
			SextuorEnd: uint16(len(f.allEntityIDs) - 1), EntityEnd: f.allEntityIDs[len(f.allEntityIDs)-1],
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
// internal/simulator/faker.go
package simulator

import (
	"context" // Ré-importation nécessaire pour l'arrêt propre de l'animation
	"fmt"
	"guitarHetic/internal/config"
	"guitarHetic/internal/domain/ehub"
	"log"
	"math"
	"slices"
	"sync" // Ré-importation nécessaire pour le Mutex
	"time"
)

type Faker struct {
	updateOut    chan<- *ehub.EHubUpdateMsg
	configOut    chan<- *ehub.EHubConfigMsg
	modeSwitch   chan<- bool
	config       *config.Config
	allEntityIDs []uint16

	// --- GESTION DE TÂCHE DE FOND (UNIQUEMENT POUR L'ANIMATION) ---
	mu              sync.Mutex
	cancelAnimation context.CancelFunc // Fonction pour annuler l'animation
}

// NewFaker constructeur simple.
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

// sendStaticPattern gère l'envoi "atomique" des motifs non animés.
func (f *Faker) sendStaticPattern(entities []ehub.EHubEntityState) {
	f.modeSwitch <- true
	f.sendInitialConfig()
	updateMsg := &ehub.EHubUpdateMsg{
		Universe: 0,
		Entities: entities,
	}
	f.updateOut <- updateMsg
}

// SendTestPattern est le point d'entrée unique.
func (f *Faker) SendTestPattern(command string, color ...byte) {
	// 1. On arrête TOUJOURS une animation potentielle avant de faire autre chose.
	f.StopAnimation()

	switch command {
	// Commandes de couleur statiques
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
	
	// Commande qui lance la tâche de fond
	case "animation":
		f.StartWaveAnimation()

	// La commande "stop" arrête l'animation et éteint les LEDs.
	case "stop":
		f.sendSolidColor(0, 0, 0, 0) // L'arrêt de l'animation est déjà fait au début.
	default:
		fmt.Printf("Unknown pattern '%s'.\n", command)
	}
}

// sendSolidColor prépare les données pour une couleur unie et appelle sendStaticPattern.
func (f *Faker) sendSolidColor(r, g, b, w byte) {
	entities := make([]ehub.EHubEntityState, len(f.allEntityIDs))
	for i, entityID := range f.allEntityIDs {
		entities[i] = ehub.EHubEntityState{ID: entityID, Red: r, Green: g, Blue: b, White: w}
	}
	f.sendStaticPattern(entities)
}

// StartWaveAnimation lance la goroutine d'animation avec un contexte d'annulation.
func (f *Faker) StartWaveAnimation() {
	f.mu.Lock()
	// Si une animation est déjà en cours (même si elle est en train de s'arrêter), on ne fait rien.
	if f.cancelAnimation != nil {
		f.mu.Unlock()
		return
	}

	// On crée un contexte qui peut être annulé, et on stocke la fonction d'annulation.
	ctx, cancel := context.WithCancel(context.Background())
	f.cancelAnimation = cancel
	f.mu.Unlock()

	// On active le mode Faker avant de lancer la goroutine.
	f.modeSwitch <- true

	go func() {
		// Assurer le nettoyage à la fin de la goroutine
		defer func() {
			f.mu.Lock()
			f.cancelAnimation = nil // On nettoie la fonction d'annulation.
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
				if position > 1.0 { position = 0.0 }
				entities := f.calculateWaveFrame(position, 0.3, 255, 100, 0)
				f.updateOut <- &ehub.EHubUpdateMsg{Universe: 0, Entities: entities}

			// Si le contexte est annulé, on sort de la boucle.
			case <-ctx.Done():
				return
			}
		}
	}()
}

// StopAnimation appelle la fonction d'annulation stockée. C'est thread-safe.
func (f *Faker) StopAnimation() {
	f.mu.Lock()
	if f.cancelAnimation != nil {
		log.Println("Faker: Signal d'arrêt envoyé à l'animation.")
		f.cancelAnimation()
		f.cancelAnimation = nil // Important: on la met à nil pour éviter les doubles appels.
	}
	f.mu.Unlock()
}

// Stop est la méthode publique appelée par l'UI pour un arrêt total.
func (f *Faker) Stop() {
	f.SwitchToLiveMode()
}

// SwitchToLiveMode arrête tout et repasse l'aiguilleur en mode LIVE.
func (f *Faker) SwitchToLiveMode() {
	f.StopAnimation()
	log.Println("Faker: Retour au mode LIVE (écoute eHub).")
	f.modeSwitch <- false
}

// sendInitialConfig envoie la configuration nécessaire au Processor.
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

// calculateWaveFrame est une fonction utilitaire pour le calcul de l'animation.
func (f *Faker) calculateWaveFrame(position, width float64, r, g, b byte) []ehub.EHubEntityState {
	entities := make([]ehub.EHubEntityState, len(f.allEntityIDs))
	totalEntities := len(f.allEntityIDs)
	for i, entityID := range f.allEntityIDs {
		entityPos := float64(i) / float64(totalEntities - 1)
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
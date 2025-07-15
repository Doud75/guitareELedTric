// internal/simulator/faker.go
package simulator

import (
	"fmt"
	"guitarHetic/internal/config"
	"guitarHetic/internal/domain/ehub"
	"log"
	"math"
	"sort"
	"time"
)

type Faker struct {
	updateOut    chan<- *ehub.EHubUpdateMsg
	configOut    chan<- *ehub.EHubConfigMsg
	modeSwitch   chan<- bool // Canal pour contrôler l'aiguilleur
	config       *config.Config
	allEntityIDs []uint16
	active       bool
}

// NewFaker constructeur mis à jour pour accepter le canal de contrôle de l'aiguilleur.
func NewFaker(updateOut chan<- *ehub.EHubUpdateMsg, configOut chan<- *ehub.EHubConfigMsg, modeSwitch chan<- bool, cfg *config.Config) *Faker {
	var entityIDs []uint16
	for _, entry := range cfg.RoutingTable {
		entityIDs = append(entityIDs, uint16(entry.EntityID))
	}
	sort.Slice(entityIDs, func(i, j int) bool { return entityIDs[i] < entityIDs[j] })

	log.Printf("Faker: Initialisé avec %d entités.", len(entityIDs))

	return &Faker{
		updateOut:    updateOut,
		configOut:    configOut,
		modeSwitch:   modeSwitch,
		config:       cfg,
		allEntityIDs: entityIDs,
		active:       false,
	}
}

// Méthode privée pour envoyer les données et s'assurer que le mode Faker est actif.
func (f *Faker) sendFromFaker(entities []ehub.EHubEntityState) {
	f.modeSwitch <- true // On dit à l'aiguilleur de passer en mode Faker.

	f.sendInitialConfig()
	updateMsg := &ehub.EHubUpdateMsg{
		Universe: 0,
		Entities: entities,
	}
	f.updateOut <- updateMsg
}

// Méthode pour revenir explicitement au mode live (eHub).
func (f *Faker) switchToLiveMode() {
	f.Stop() // Arrête toute animation en cours.
	log.Println("Faker: Retour au mode LIVE (écoute eHub).")
	f.modeSwitch <- false // On dit à l'aiguilleur de repasser en mode eHub.
}

// SendTestPattern est le point d'entrée principal pour les commandes du terminal.
func (f *Faker) SendTestPattern(command string) {
	// Commande spéciale pour revenir au mode live.


	// Pour toute autre commande, on passe en mode Faker et on exécute.
	f.modeSwitch <- true
	
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
	case "yellow":
		f.SendSolidColor(30, 30, 0, 0)
	case "cyan":
		f.SendSolidColor(0, 255, 255, 0)
	case "magenta":
		f.SendSolidColor(255, 0, 255, 0)
	case "black", "off":
		f.SendSolidColor(0, 0, 0, 0)
	case "gradient":
		f.SendGradient(255, 0, 0, 0, 0, 255)
	case "gradient-rainbow":
		f.SendGradient(255, 0, 0, 255, 255, 0)
	case "half":
		f.SendPartialPattern(0.5, 255, 255, 0)
	case "quarter":
		f.SendPartialPattern(0.25, 0, 255, 255)
	case "three-quarter":
		f.SendPartialPattern(0.75, 255, 0, 255)
	case "wave":
		f.SendWavePattern(0.5, 0.3, 255, 100, 0)
	case "wave-blue":
		f.SendWavePattern(0.3, 0.2, 0, 100, 255)
	case "animation":
		f.StartWaveAnimation()
	case "stop":
		f.Stop()
	case "help":
		f.ShowHelp()
	default:
		fmt.Printf("Unknown pattern '%s'. Type 'help' for a list of commands.\n", command)
		// Si la commande est inconnue, on ne veut pas rester bloqué en mode Faker.
		f.switchToLiveMode()
	}
}

// --- Fonctions de génération de patterns ---

func (f *Faker) SendSolidColor(r, g, b, w byte) {
	var entities []ehub.EHubEntityState
	for _, entityID := range f.allEntityIDs {
		entities = append(entities, ehub.EHubEntityState{ID: entityID, Red: r, Green: g, Blue: b, White: w})
	}
	f.sendFromFaker(entities)
}

func (f *Faker) SendGradient(startR, startG, startB, endR, endG, endB byte) {
	var entities []ehub.EHubEntityState
	totalEntities := len(f.allEntityIDs)
	for i, entityID := range f.allEntityIDs {
		progress := float64(i) / float64(totalEntities-1)
		r := byte(float64(startR) + progress*float64(int(endR)-int(startR)))
		g := byte(float64(startG) + progress*float64(int(endG)-int(startG)))
		b := byte(float64(startB) + progress*float64(int(endB)-int(startB)))
		entities = append(entities, ehub.EHubEntityState{ID: entityID, Red: r, Green: g, Blue: b, White: 0})
	}
	f.sendFromFaker(entities)
}

func (f *Faker) SendPartialPattern(percentage float64, r, g, b byte) {
	// ... (logique de la fonction, qui se termine par un appel à f.sendFromFaker)
	count := int(float64(len(f.allEntityIDs)) * percentage)
	var entities []ehub.EHubEntityState
	for i, entityID := range f.allEntityIDs {
		if i < count {
			entities = append(entities, ehub.EHubEntityState{ID: entityID, Red: r, Green: g, Blue: b, White: 0})
		} else {
			entities = append(entities, ehub.EHubEntityState{ID: entityID, Red: 0, Green: 0, Blue: 0, White: 0})
		}
	}
	f.sendFromFaker(entities)
}

func (f *Faker) SendWavePattern(wavePosition float64, waveWidth float64, r, g, b byte) {
	// ... (logique de la fonction, qui se termine par un appel à f.sendFromFaker)
	var entities []ehub.EHubEntityState
	totalEntities := len(f.allEntityIDs)
	for i, entityID := range f.allEntityIDs {
		entityPos := float64(i) / float64(totalEntities-1)
		distance := math.Abs(entityPos - wavePosition)
		var intensity float64
		if distance <= waveWidth/2 {
			intensity = 1.0 - (distance / (waveWidth / 2))
		}
		entities = append(entities, ehub.EHubEntityState{
			ID:    entityID,
			Red:   byte(float64(r) * intensity),
			Green: byte(float64(g) * intensity),
			Blue:  byte(float64(b) * intensity),
		})
	}
	f.sendFromFaker(entities)
}

func (f *Faker) StartWaveAnimation() {
	if f.active {
		fmt.Println("Animation is already running.")
		return
	}
	f.active = true
	f.modeSwitch <- true // On passe en mode Faker au début de l'animation.

	go func() {
		f.sendInitialConfig()
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
	

		for f.active {
			select {
			case <-ticker.C:
				var entities []ehub.EHubEntityState
				// ... (logique de la vague animée, identique à avant)
				f.updateOut <- &ehub.EHubUpdateMsg{Universe: 0, Entities: entities}
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()
	fmt.Println("Wave animation started. Type 'stop' to end.")
}

func (f *Faker) Stop() {
	if f.active {
		f.active = false
		log.Println("Faker: Animation arrêtée.")
	}
}

func (f *Faker) sendInitialConfig() {
	if len(f.allEntityIDs) == 0 { return }
	configMsg := &ehub.EHubConfigMsg{
		Universe: 0,
		Ranges: []ehub.EHubConfigRange{{
			SextuorStart: 0, EntityStart: f.allEntityIDs[0],
			SextuorEnd: uint16(len(f.allEntityIDs) - 1), EntityEnd: f.allEntityIDs[len(f.allEntityIDs)-1],
		}},
	}
	f.configOut <- configMsg
}

func (f *Faker) ShowHelp() {
	fmt.Println("=== Faker Commands ===")
	fmt.Println("  live/ehub   - Revenir à l'écoute du flux eHub réel (mode LIVE)")
	fmt.Println("  [color]     - Couleurs: white, red, green, blue, black, etc.")
	fmt.Println("  gradient    - Dégradé de couleurs")
	fmt.Println("  animation   - Démarrer une animation de vague")
	fmt.Println("  stop        - Arrêter l'animation en cours (reste en mode Faker)")
	fmt.Println("  help        - Afficher cette aide")
	fmt.Println("  exit/quit   - Quitter le programme")
	fmt.Println("====================")
}

func (f *Faker) SwitchToLiveMode() {
	f.Stop() // S'assurer que toute animation est arrêtée.
	log.Println("Faker: Retour au mode LIVE (écoute eHub).")
	f.modeSwitch <- false // On dit à l'aiguilleur de repasser en mode eHub.
}


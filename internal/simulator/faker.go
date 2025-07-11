// internal/simulator/faker.go
package simulator

import (
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
	config       *config.Config
	allEntityIDs []int // Liste de TOUTES les entités de l'installation
	active       bool
}

func NewFaker(updateOut chan<- *ehub.EHubUpdateMsg, configOut chan<- *ehub.EHubConfigMsg, cfg *config.Config) *Faker {
	// ✅ Extraction de TOUTES les entités de la config
	var entityIDs []int
	for _, entry := range cfg.RoutingTable {
		entityIDs = append(entityIDs, entry.EntityID)
	}
	sort.Ints(entityIDs) // Tri pour les patterns séquentiels

	log.Printf("Faker: Trouvé %d entités dans la configuration", len(entityIDs))

	return &Faker{
		updateOut:    updateOut,
		configOut:    configOut,
		config:       cfg,
		allEntityIDs: entityIDs,
		active:       false,
	}
}

// ✅ Pattern universel : Toutes les entités d'une couleur
func (f *Faker) SendSolidColor(r, g, b byte) {
	log.Printf("Faker: Envoi couleur unie RGB(%d,%d,%d) sur %d entités", r, g, b, len(f.allEntityIDs))

	var entities []ehub.EHubEntityState
	for _, entityID := range f.allEntityIDs {
		entities = append(entities, ehub.EHubEntityState{
			ID:    uint16(entityID),
			Red:   r,
			Green: g,
			Blue:  b,
			White: 0,
		})
	}

	f.sendConfigAndUpdate(entities)
}

// ✅ Pattern universel : Gradient par EntityID
func (f *Faker) SendGradient(startR, startG, startB, endR, endG, endB byte) {
	log.Printf("Faker: Envoi gradient sur %d entités", len(f.allEntityIDs))

	var entities []ehub.EHubEntityState
	totalEntities := len(f.allEntityIDs)

	for i, entityID := range f.allEntityIDs {
		// Interpolation linéaire basée sur la position dans la liste
		progress := float64(i) / float64(totalEntities-1)

		r := byte(float64(startR) + progress*float64(int(endR)-int(startR)))
		g := byte(float64(startG) + progress*float64(int(endG)-int(startG)))
		b := byte(float64(startB) + progress*float64(int(endB)-int(startB)))

		entities = append(entities, ehub.EHubEntityState{
			ID:    uint16(entityID),
			Red:   r,
			Green: g,
			Blue:  b,
			White: 0,
		})
	}

	f.sendConfigAndUpdate(entities)
}

// ✅ Pattern universel : Sous-ensemble par pourcentage
func (f *Faker) SendPartialPattern(percentage float64, r, g, b byte) {
	if percentage < 0 || percentage > 1 {
		log.Printf("Faker: Pourcentage invalide: %.2f", percentage)
		return
	}

	count := int(float64(len(f.allEntityIDs)) * percentage)
	log.Printf("Faker: Envoi sur %d entités (%.1f%% de %d)", count, percentage*100, len(f.allEntityIDs))

	var entities []ehub.EHubEntityState

	// Les autres en noir
	for i, entityID := range f.allEntityIDs {
		if i < count {
			// Entités colorées
			entities = append(entities, ehub.EHubEntityState{
				ID:    uint16(entityID),
				Red:   r,
				Green: g,
				Blue:  b,
				White: 0,
			})
		} else {
			// Entités noires
			entities = append(entities, ehub.EHubEntityState{
				ID:    uint16(entityID),
				Red:   0,
				Green: 0,
				Blue:  0,
				White: 0,
			})
		}
	}

	f.sendConfigAndUpdate(entities)
}

// ✅ Pattern universel : Vague qui se propage
func (f *Faker) SendWavePattern(wavePosition float64, waveWidth float64, r, g, b byte) {
	log.Printf("Faker: Envoi vague position %.2f, largeur %.2f", wavePosition, waveWidth)

	var entities []ehub.EHubEntityState
	totalEntities := len(f.allEntityIDs)

	for i, entityID := range f.allEntityIDs {
		// Position normalisée de l'entité (0.0 à 1.0)
		entityPos := float64(i) / float64(totalEntities-1)

		// Distance à la vague
		distance := math.Abs(entityPos - wavePosition)

		// Intensité basée sur la distance (effet de vague)
		var intensity float64
		if distance <= waveWidth/2 {
			intensity = 1.0 - (distance/(waveWidth/2))
		} else {
			intensity = 0.0
		}

		entities = append(entities, ehub.EHubEntityState{
			ID:    uint16(entityID),
			Red:   byte(float64(r) * intensity),
			Green: byte(float64(g) * intensity),
			Blue:  byte(float64(b) * intensity),
			White: 0,
		})
	}

	f.sendConfigAndUpdate(entities)
}

// ✅ Pattern universel : Animation continue
func (f *Faker) StartWaveAnimation() {
	f.active = true
	go func() {
		// Envoyer la config une fois
		f.sendInitialConfig()

		ticker := time.NewTicker(50 * time.Millisecond) // 20 FPS
		defer ticker.Stop()

		position := 0.0
		direction := 0.02 // Vitesse de la vague

		for f.active {
			<-ticker.C
			f.SendWavePattern(position, 0.2, 255, 100, 0) // Vague orange

			position += direction
			if position >= 1.0 {
				position = 0.0 // Recommence
			}
		}
	}()

	log.Println("Faker: Animation vague démarrée")
}

// ✅ Méthodes d'aide
func (f *Faker) sendConfigAndUpdate(entities []ehub.EHubEntityState) {
	f.sendInitialConfig()

	updateMsg := &ehub.EHubUpdateMsg{
		Universe: 0,
		Entities: entities,
	}

	f.updateOut <- updateMsg
}

func (f *Faker) sendInitialConfig() {
	if len(f.allEntityIDs) == 0 {
		return
	}

	configMsg := &ehub.EHubConfigMsg{
		Universe: 0,
		Ranges: []ehub.EHubConfigRange{
			{
				SextuorStart: 0,
				EntityStart:  uint16(f.allEntityIDs[0]),
				SextuorEnd:   uint16(len(f.allEntityIDs) - 1),
				EntityEnd:    uint16(f.allEntityIDs[len(f.allEntityIDs)-1]),
			},
		},
	}

	f.configOut <- configMsg
}

func (f *Faker) Stop() {
	f.active = false
	log.Println("Faker: Arrêté")
}

// ✅ Interface simple pour tous types d'installations
func (f *Faker) SendTestPattern(pattern string) {
	switch pattern {
	case "white":
		f.SendSolidColor(255, 255, 255)
	case "red":
		f.SendSolidColor(255, 0, 0)
	log.Println("  white, red, green, blue, yellow, cyan, magenta, black")
	log.Println("Gradients:")
	log.Println("  gradient, gradient-rainbow")
	log.Println("Patterns partiels:")
	log.Println("  half (50%), quarter (25%), three-quarter (75%)")
	log.Println("Vagues:")
	log.Println("  wave, wave-blue")
		f.SendSolidColor(0, 255, 255)
	case "magenta":
		f.SendSolidColor(255, 0, 255)
	case "black":
		f.SendSolidColor(0, 0, 0)
	case "gradient":
		f.SendGradient(255, 0, 0, 0, 0, 255) // Rouge vers bleu
	case "gradient-rainbow":
		f.SendGradient(255, 0, 0, 255, 255, 0) // Rouge vers jaune
	case "half":
		f.SendPartialPattern(0.5, 255, 255, 0) // 50% jaune
	case "quarter":
		f.SendPartialPattern(0.25, 0, 255, 255) // 25% cyan
	case "three-quarter":
		f.SendPartialPattern(0.75, 255, 0, 255) // 75% magenta
	case "wave":
		f.SendWavePattern(0.5, 0.3, 255, 100, 0) // Vague au centre
	case "wave-blue":
		f.SendWavePattern(0.3, 0.2, 0, 100, 255) // Vague bleue
	case "animation":
		f.StartWaveAnimation()
	case "stop":
		f.Stop()
	default:
		log.Printf("Faker: Pattern '%s' non reconnu", pattern)
		f.ShowHelp()
	}
}

func (f *Faker) ShowHelp() {
	log.Println("=== FAKER - PATTERNS DISPONIBLES ===")
	log.Println("Couleurs unies:")
	log.Println("  white, red, green, blue, yellow, cyan, magenta, black")
	log.Println("Gradients:")
	log.Println("  gradient, gradient-rainbow")
	log.Println("Patterns partiels:")
	log.Println("  half (50%), quarter (25%), three-quarter (75%)")
	log.Println("Vagues:")
	log.Println("  wave, wave-blue")
	log.Println("Animations:")
	log.Println("  animation (démarre), stop (arrête)")
	log.Println("=====================================")
}

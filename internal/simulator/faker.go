// internal/simulator/faker.go
package simulator

import (
	"bufio"
	"fmt"
	"guitarHetic/internal/config"
	"guitarHetic/internal/domain/ehub"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

type Faker struct {
	updateOut    chan<- *ehub.EHubUpdateMsg
	configOut    chan<- *ehub.EHubConfigMsg
	config       *config.Config
	allEntityIDs []uint16 // Changed to uint16 to match EHubEntityState
	active       bool
}

func NewFaker(updateOut chan<- *ehub.EHubUpdateMsg, configOut chan<- *ehub.EHubConfigMsg, cfg *config.Config) *Faker {
	var entityIDs []uint16
	for _, entry := range cfg.RoutingTable {
		entityIDs = append(entityIDs, uint16(entry.EntityID))
	}
	sort.Slice(entityIDs, func(i, j int) bool { return entityIDs[i] < entityIDs[j] })

	log.Printf("Faker: Initialized with %d entities.", len(entityIDs))

	return &Faker{
		updateOut:    updateOut,
		configOut:    configOut,
		config:       cfg,
		allEntityIDs: entityIDs,
		active:       false,
	}
}

// StartInteractive a blocking method to read commands from stdin.
func (f *Faker) StartInteractive() {
	reader := bufio.NewReader(os.Stdin)
	f.ShowHelp()
	for {
		fmt.Print("faker> ")
		input, _ := reader.ReadString('\n')
		command := strings.TrimSpace(input)

		if command == "" {
			continue
		}

		if command == "exit" {
			log.Println("Faker: Exiting interactive mode.")
			break
		}

		f.SendTestPattern(command)
	}
}

// SendSolidColor sends a solid color to all entities.
func (f *Faker) SendSolidColor(r, g, b, w byte) {
	var entities []ehub.EHubEntityState
	for _, entityID := range f.allEntityIDs {
		entities = append(entities, ehub.EHubEntityState{
			ID:    entityID,
			Red:   r,
			Green: g,
			Blue:  b,
			White: w,
		})
	}

	f.sendConfigAndUpdate(entities)
}

// SendGradient sends a gradient across all entities.
func (f *Faker) SendGradient(startR, startG, startB, endR, endG, endB byte) {


	var entities []ehub.EHubEntityState
	totalEntities := len(f.allEntityIDs)

	for i, entityID := range f.allEntityIDs {
		// Interpolation linéaire basée sur la position dans la liste
		progress := float64(i) / float64(totalEntities-1)

		r := byte(float64(startR) + progress*float64(int(endR)-int(startR)))
		g := byte(float64(startG) + progress*float64(int(endG)-int(startG)))
		b := byte(float64(startB) + progress*float64(int(endB)-int(startB)))

		entities = append(entities, ehub.EHubEntityState{
			ID:    entityID,
			Red:   r,
			Green: g,
			Blue:  b,
			White: 0,
		})
	}

	f.sendConfigAndUpdate(entities)
}

// SendPartialPattern sends a color to a percentage of entities.
func (f *Faker) SendPartialPattern(percentage float64, r, g, b byte) {
	if percentage < 0 || percentage > 1 {
		fmt.Printf("Invalid percentage: %.2f\n", percentage)
		return
	}

	count := int(float64(len(f.allEntityIDs)) * percentage)
	fmt.Printf("Sending pattern to %d entities (%.1f%% of %d)\n", count, percentage*100, len(f.allEntityIDs))

	var entities []ehub.EHubEntityState

	for i, entityID := range f.allEntityIDs {
		if i < count {
			entities = append(entities, ehub.EHubEntityState{
				ID:    entityID,
				Red:   r,
				Green: g,
				Blue:  b,
				White: 0,
			})
		} else {
			entities = append(entities, ehub.EHubEntityState{
				ID:    entityID,
				Red:   0,
				Green: 0,
				Blue:  0,
				White: 0,
			})
		}
	}

	f.sendConfigAndUpdate(entities)
}

// SendWavePattern sends a wave of color.
func (f *Faker) SendWavePattern(wavePosition float64, waveWidth float64, r, g, b byte) {


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
			ID:    entityID,
			Red:   byte(float64(r) * intensity),
			Green: byte(float64(g) * intensity),
			Blue:  byte(float64(b) * intensity),
			White: 0,
		})
	}

	f.sendConfigAndUpdate(entities)
}

// StartWaveAnimation starts a continuous wave animation.
func (f *Faker) StartWaveAnimation() {
	if f.active {
		fmt.Println("Animation is already running.")
		return
	}
	f.active = true
	go func() {
		f.sendInitialConfig()

		ticker := time.NewTicker(50 * time.Millisecond) // 20 FPS
		defer ticker.Stop()

		position := 0.0
		direction := 0.02

		for f.active {
			select {
			case <-ticker.C:
				f.SendWavePattern(position, 0.2, 255, 100, 0)

				position += direction
				if position >= 1.0 {
					position = 0.0
				}
			default:
				// Allows the loop to exit if f.active becomes false
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	fmt.Println("Wave animation started. Type 'stop' to end.")
}

// sendConfigAndUpdate sends the configuration and then the entity states.
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
				EntityStart:  f.allEntityIDs[0],
				SextuorEnd:   uint16(len(f.allEntityIDs) - 1),
				EntityEnd:    f.allEntityIDs[len(f.allEntityIDs)-1],
			},
		},
	}

	f.configOut <- configMsg
}

// Stop stops any active animation.
func (f *Faker) Stop() {
	if f.active {
		f.active = false
		fmt.Println("Animation stopped.")
	} else {
		fmt.Println("No animation is currently running.")
	}
}

// SendTestPattern interprets a string command to generate a pattern.
func (f *Faker) SendTestPattern(pattern string) {
	switch pattern {
	case "white":
		f.SendSolidColor(255, 255, 255, 0)
	case "pure-white":
		f.SendSolidColor(0, 0, 0, 255)
	case "red":
		f.SendSolidColor(255, 0, 0, 0)
	case "green":
		f.SendSolidColor(0, 255, 0, 0)
	case "blue":
		f.SendSolidColor(0, 0, 255, 0)
	case "yellow":
		f.SendSolidColor(255, 255, 0, 0)
	case "cyan":
		f.SendSolidColor(0, 255, 255, 0)
	case "magenta":
		f.SendSolidColor(255, 0, 255, 0)
	case "black", "off":
		f.SendSolidColor(0, 0, 0, 0)
	case "gradient":
		f.SendGradient(255, 0, 0, 0, 0, 255) // Red to Blue
	case "gradient-rainbow":
		f.SendGradient(255, 0, 0, 255, 255, 0) // Red to Yellow
	case "half":
		f.SendPartialPattern(0.5, 255, 255, 0) // 50% Yellow
	case "quarter":
		f.SendPartialPattern(0.25, 0, 255, 255) // 25% Cyan
	case "three-quarter":
		f.SendPartialPattern(0.75, 255, 0, 255) // 75% Magenta
	case "wave":
		f.SendWavePattern(0.5, 0.3, 255, 100, 0) // Wave in the middle
	case "wave-blue":
		f.SendWavePattern(0.3, 0.2, 0, 100, 255) // Blue wave
	case "animation":
		f.StartWaveAnimation()
	case "stop":
		f.Stop()
	case "help":
		f.ShowHelp()
	default:
		fmt.Printf("Unknown pattern '%s'. Type 'help' for a list of commands.\n", pattern)
	}
}

// ShowHelp displays available commands.
func (f *Faker) ShowHelp() {
	fmt.Println("=== Faker Commands ===")
	fmt.Println("Solid Colors: white, pure-white, red, green, blue, yellow, cyan, magenta, black/off")
	fmt.Println("Gradients:    gradient, gradient-rainbow")
	fmt.Println("Patterns:     half, quarter, three-quarter")
	fmt.Println("Waves:        wave, wave-blue")
	fmt.Println("Animations:   animation (start), stop (end)")
	fmt.Println("Other:        help (show this message), exit (quit)")
	fmt.Println("====================")
}

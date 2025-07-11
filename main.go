// File: main.go
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	app_ehub "guitarHetic/internal/application/ehub"
	app_processor "guitarHetic/internal/application/processor"
	"guitarHetic/internal/config"
	domain_artnet "guitarHetic/internal/domain/artnet"
	"guitarHetic/internal/domain/ehub"
	infra_artnet "guitarHetic/internal/infrastructure/artnet"
	infra_ehub "guitarHetic/internal/infrastructure/ehub"
	"guitarHetic/internal/simulator"
)

func main() {
	log.Println("Démarrage du système de routage eHuB -> ArtNet...")

	// --- ÉTAPE 1 : CHARGEMENT DE LA CONFIGURATION (UNE SEULE FOIS) ---
	log.Println("Main: Chargement de la configuration depuis routing.csv...")
	appConfig, err := config.Load("internal/config/routing.xlsx")
	if err != nil {
		// Si la config de base ne peut pas être chargée, l'application ne peut pas fonctionner.
		// C'est une erreur fatale.
		log.Fatalf("Erreur fatale: Impossible de charger routing.csv: %v", err)
	}
	log.Println("Main: Configuration chargée avec succès.")


	// --- ÉTAPE 2 : CRÉATION DES CANAUX DE COMMUNICATION (OPTIMISÉS) ---
	rawPacketChannel := make(chan ehub.RawPacket, 1000)       // Augmenté pour éviter blocage UDP
	configChannel := make(chan *ehub.EHubConfigMsg, 50)       // Augmenté mais reste petit (configs rares)
	updateChannel := make(chan *ehub.EHubUpdateMsg, 1000)     // Augmenté pour 40 FPS
	artnetQueue := make(chan domain_artnet.LEDMessage, 10000) // Déjà optimisé

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()


	// --- ÉTAPE 3 : CONSTRUCTION DES COMPOSANTS ---
	
	// a) Infrastructure (couche externe)
	const eHubPort = 8765
	listener, err := infra_ehub.NewListener(eHubPort, rawPacketChannel)
	if err != nil {
		log.Fatalf("Erreur Listener: %v", err)
	}
	
	// Le Sender a besoin de la map des IPs des univers.
	sender, err := infra_artnet.NewSender(appConfig.UniverseIP)
	if err != nil {
		log.Fatalf("Erreur Sender: %v", err)
	}

	// b) Application (logique métier)
	parser := app_ehub.NewParser()
	eHubService := app_ehub.NewService(rawPacketChannel, parser, configChannel, updateChannel)
	
	// Le Processor nous donne un canal pour lui envoyer des configs plus tard.
	processorService, physicalConfigOut := app_processor.NewService(configChannel, updateChannel, artnetQueue)

	// c) Faker pour tests (E10)
	faker := simulator.NewFaker(updateChannel, configChannel, appConfig)


	// --- ÉTAPE 4 : DÉMARRAGE DES GOROUTINES ---
	
	listener.Start() // On ajoutera le contexte ici plus tard
	eHubService.Start()
	processorService.Start()
	go sender.Run(ctx, artnetQueue)


	// --- ÉTAPE 5 : INJECTION DE LA CONFIGURATION INITIALE ---
	// On n'a plus besoin de recharger le fichier. On utilise l'objet `appConfig` déjà chargé.
	log.Println("Main: Envoi de la configuration initiale au Processor.")
	physicalConfigOut <- appConfig


	// --- ÉTAPE 6 : ATTENTE AVEC INTERFACE FAKER ---
	log.Println("Système entièrement démarré.")
	log.Println("=== FAKER ACTIVÉ (E10) ===")
	log.Println("Commandes disponibles :")
	log.Println("  help    - Affiche l'aide complète")
	log.Println("  [color] - Écran [color]")
	log.Println("  gradient - Affiche un gradient")
	log.Println("  animation - Animation de vague")
	log.Println("  stop    - Arrête l'animation")
	log.Println("  quit    - Quitte le programme")
	log.Println("===========================")
	
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("faker> ")
		if !scanner.Scan() {
			break
		}
		
		command := strings.TrimSpace(scanner.Text())
		if command == "" {
			continue
		}
		
		switch command {
		case "quit", "exit", "q":
			log.Println("Arrêt du système...")
			faker.Stop()
			cancel()
			return
		case "help":
			faker.ShowHelp()
		default:
			faker.SendTestPattern(command)
		}
	}
}
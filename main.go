// File: main.go
package main

import (
	"context"
	"log"
	"time"
	
	app_ehub "guitarHetic/internal/application/ehub"
	app_processor "guitarHetic/internal/application/processor"
	"guitarHetic/internal/config"
	domain_artnet "guitarHetic/internal/domain/artnet"
	"guitarHetic/internal/domain/ehub"
	infra_artnet "guitarHetic/internal/infrastructure/artnet"
	infra_ehub "guitarHetic/internal/infrastructure/ehub"
)

func main() {
	log.Println("Démarrage du système de routage eHuB -> ArtNet...")

	// --- 1. CHARGEMENT & CRÉATION DES CANAUX ---
	
	// Charger la config physique une fois au démarrage pour le Sender.
	appConfig, err := config.Load("internal/config/routing.csv")
	if err != nil {
		log.Fatalf("Erreur fatale: Impossible de charger routing.csv: %v", err)
	}

	// Canaux de communication
	rawPacketChannel := make(chan ehub.RawPacket, 100)
	configChannel := make(chan *ehub.EHubConfigMsg, 10)
	updateChannel := make(chan *ehub.EHubUpdateMsg, 100)
	artnetQueue := make(chan domain_artnet.LEDMessage, 500) // Canal vers le sender

	// Contexte pour gérer l'arrêt propre de l'application
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // S'assurer que l'annulation est appelée à la fin

	// --- 2. CONSTRUCTION DES COMPOSANTS ---
	
	// a) Infrastructure (couche externe)
	const eHubPort = 8765
	listener, err := infra_ehub.NewListener(eHubPort, rawPacketChannel)
	if err != nil {
		log.Fatalf("Erreur Listener: %v", err)
	}
	
	sender, err := infra_artnet.NewSender(appConfig.UniverseIP)
	if err != nil {
		log.Fatalf("Erreur Sender: %v", err)
	}

	// b) Application (logique métier)
	parser := app_ehub.NewParser()
	eHubService := app_ehub.NewService(rawPacketChannel, parser, configChannel, updateChannel)
	processorService := app_processor.NewService(configChannel, updateChannel, artnetQueue)


	// --- 3. DÉMARRAGE DES GOROUTINES ---
	
	listener.Start() // On ajoutera le contexte plus tard
	eHubService.Start()
	processorService.Start()
	go sender.Run(ctx, artnetQueue) // Le sender prend le contexte pour s'arrêter

	// --- 4. ATTENTE ---
	log.Println("Système entièrement démarré. En attente de données eHuB de Unity...")
	for {
		time.Sleep(1 * time.Hour)
	}
}
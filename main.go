package main

import (
	"context"
	"log"

	app_ehub "guitarHetic/internal/application/ehub"
	app_processor "guitarHetic/internal/application/processor"
	"guitarHetic/internal/config"
	domain_artnet "guitarHetic/internal/domain/artnet"
	"guitarHetic/internal/domain/ehub"
	infra_artnet "guitarHetic/internal/infrastructure/artnet"
	infra_ehub "guitarHetic/internal/infrastructure/ehub"
	"guitarHetic/internal/simulator"
	"guitarHetic/internal/ui"
)

func main() {
    log.Println("Démarrage du système de routage eHuB -> ArtNet...")

	// --- 1. CONFIGURATION ---
	appConfig, err := config.Load("internal/config/routing.xlsx")
	if err != nil {
		log.Fatalf("Erreur fatale: Impossible de charger la configuration: %v", err)
	}

    // --- CRÉATION DES CANAUX ET CONTEXTE ---
    rawPacketChannel := make(chan ehub.RawPacket, 1000)
    configChannel := make(chan *ehub.EHubConfigMsg, 50)
    updateChannel := make(chan *ehub.EHubUpdateMsg, 1000)
    artnetQueue := make(chan domain_artnet.LEDMessage, 10000)
    monitorChan := make(chan *ui.UniverseMonitorData, 100)
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

	// Canaux pour l'aiguilleur
	eHubUpdateChannel := make(chan *ehub.EHubUpdateMsg, 1000)
	fakerUpdateChannel := make(chan *ehub.EHubUpdateMsg, 1000)
	fakerModeSwitch := make(chan bool)

	// --- 3. CONSTRUCTION DES COMPOSANTS ---
	listener, _ := infra_ehub.NewListener(8765, rawPacketChannel)
	sender, _ := infra_artnet.NewSender()
	parser := app_ehub.NewParser()
	eHubService := app_ehub.NewService(rawPacketChannel, parser, configChannel, eHubUpdateChannel)
	processorService, physicalConfigOut := app_processor.NewService(configChannel, updateChannel, artnetQueue, monitorChan)
	faker := simulator.NewFaker(fakerUpdateChannel, configChannel, fakerModeSwitch, appConfig)

	// --- 4. DÉMARRAGE DES SERVICES BACKEND DANS DES GOROUTINES ---
	log.Println("Démarrage des services backend en arrière-plan...")

	// Lancement de l'aiguilleur
	go func() {
		isFakerActive := false
		log.Println("Aiguilleur: Démarré en mode LIVE (écoute eHub).")
		for {
			select {
			case <-ctx.Done(): return // Permet d'arrêter proprement la goroutine
			case mode := <-fakerModeSwitch:
				if mode != isFakerActive {
					isFakerActive = mode
					if isFakerActive { log.Println("Aiguilleur: Passage en mode FAKER.")
					} else { log.Println("Aiguilleur: Retour au mode LIVE.") }
				}
			case msg := <-fakerUpdateChannel:
				if isFakerActive { updateChannel <- msg }
			case msg := <-eHubUpdateChannel:
				if !isFakerActive { updateChannel <- msg }
			}
		}
	}()

	listener.Start()
	eHubService.Start()
	processorService.Start()
	go sender.Run(ctx, artnetQueue)

	// --- 5. INJECTION DE LA CONFIG & DÉMARRAGE DE L'UI SUR LE THREAD PRINCIPAL ---
	physicalConfigOut <- appConfig
	log.Println("Système backend démarré. Lancement de l'interface graphique...")

	// L'appel à RunUI est maintenant la dernière action de la fonction main.
	// Il va bloquer l'exécution jusqu'à ce que la fenêtre soit fermée.
	// C'est la manière correcte de lancer une application Fyne.
    ui.RunUI(appConfig, physicalConfigOut, faker, monitorChan)

	// Ce log ne s'affichera que lorsque la fenêtre UI aura été fermée.
	log.Println("Arrêt complet de l'application.")
}

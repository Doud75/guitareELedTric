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

	// --- ÉTAPE 1 : CHARGEMENT DE LA CONFIGURATION (UNE SEULE FOIS) ---
	log.Println("Main: Chargement de la configuration depuis routing.csv...")
	appConfig, err := config.Load("internal/config/routing.xlsx")
	if err != nil {
		// Si la config de base ne peut pas être chargée, l'application ne peut pas fonctionner.
		// C'est une erreur fatale.
		log.Fatalf("Erreur fatale: Impossible de charger routing.csv: %v", err)
	}
	log.Println("Main: Configuration chargée avec succès.")


	// --- ÉTAPE 2 : CRÉATION DES CANAUX DE COMMUNICATION ---
	rawPacketChannel := make(chan ehub.RawPacket, 100)
	configChannel := make(chan *ehub.EHubConfigMsg, 10)
	updateChannel := make(chan *ehub.EHubUpdateMsg, 100)
	artnetQueue := make(chan domain_artnet.LEDMessage, 500)

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


	// --- ÉTAPE 4 : DÉMARRAGE DES GOROUTINES ---
	
	listener.Start() // On ajoutera le contexte ici plus tard
	eHubService.Start()
	processorService.Start()
	go sender.Run(ctx, artnetQueue)


	// --- ÉTAPE 5 : INJECTION DE LA CONFIGURATION INITIALE ---
	// On n'a plus besoin de recharger le fichier. On utilise l'objet `appConfig` déjà chargé.
	log.Println("Main: Envoi de la configuration initiale au Processor.")
	physicalConfigOut <- appConfig


	// --- ÉTAPE 6 : ATTENTE ---
	log.Println("Système entièrement démarré. En attente de données eHuB de Unity...")
	for {
		time.Sleep(1 * time.Hour)
	}
}
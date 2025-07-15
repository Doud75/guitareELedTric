// File: main.go
package main

import (
    "context"
    app_ehub "guitarHetic/internal/application/ehub"
    app_processor "guitarHetic/internal/application/processor"
    "guitarHetic/internal/config"
    domain_artnet "guitarHetic/internal/domain/artnet"
    "guitarHetic/internal/domain/ehub"
    infra_artnet "guitarHetic/internal/infrastructure/artnet"
    infra_ehub "guitarHetic/internal/infrastructure/ehub"
    "guitarHetic/internal/simulator"
    "guitarHetic/internal/ui"
    "log"
)

func main() {
    log.Println("Démarrage du système de routage eHuB -> ArtNet...")

    // --- CHARGEMENT DE LA CONFIGURATION ---
    appConfig, err := config.Load("internal/config/routing.xlsx")
    if err != nil {
        log.Fatalf("Erreur fatale: Impossible de charger la configuration: %v", err)
    }

    // --- CRÉATION DES CANAUX ET CONTEXTE ---
    rawPacketChannel := make(chan ehub.RawPacket, 1000)
    configChannel := make(chan *ehub.EHubConfigMsg, 50)
    updateChannel := make(chan *ehub.EHubUpdateMsg, 1000)
    artnetQueue := make(chan domain_artnet.LEDMessage, 10000)
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
    sender, err := infra_artnet.NewSender()
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

    // --- DÉMARRAGE DES SERVICES BACKEND ---
    listener.Start()
    eHubService.Start()
    processorService.Start()
    go sender.Run(ctx, artnetQueue)

    // --- INJECTION DE LA CONFIG & DÉMARRAGE DE L'UI ---
    physicalConfigOut <- appConfig
    log.Println("Système backend démarré. Lancement de l'interface graphique...")

    ui.RunUI(appConfig, physicalConfigOut, faker)
    
    log.Println("Arrêt complet de l'application.")
}

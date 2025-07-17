// main.go
package main

import (
	"context"
	"log"

	"fyne.io/fyne/v2/app"
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
	log.Println("Démarrage du système...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- CANAUX DE COMMUNICATION ---
	// Ces canaux sont créés une seule fois et vivent aussi longtemps que l'application.
	configRequestChannel := make(chan ui.ConfigUpdateRequest, 1)
	fakerModeSwitch := make(chan bool)
	eHubUpdateChannel := make(chan *ehub.EHubUpdateMsg, 1000)
	fakerUpdateChannel := make(chan *ehub.EHubUpdateMsg, 1000)
	fakerConfigOut := make(chan *ehub.EHubConfigMsg, 50)
	monitorChan := make(chan *ui.UniverseMonitorData, 100) // LE CANAL DE MONITORING EST ICI

	var faker *simulator.Faker = nil

	a := app.New()
	a.Settings().SetTheme(&ui.ArtHeticTheme{})
	w := a.NewWindow("Guitare Hetic - Inspecteur ArtNet")
	uiController := ui.NewUIController(a, faker, monitorChan, func(req ui.ConfigUpdateRequest) {
		configRequestChannel <- req
	})
	ui.RunUI(uiController, w)

	go func() {
		var currentConfig *config.Config
		var cancelPipeline context.CancelFunc = func() {}

		stopPipeline := func() {
			log.Println("Gestionnaire de Config: Arrêt du pipeline de traitement...")
			cancelPipeline()
		}
		
		for {
			select {
			case req := <-configRequestChannel:
				stopPipeline()

				if req.FilePath != "" {
					log.Printf("Gestionnaire de Config: Chargement du fichier %s", req.FilePath)
					newConfig, err := config.Load(req.FilePath)
					if err != nil {
						log.Printf("ERREUR: Impossible de charger le fichier de configuration: %v", err)
						currentConfig = nil
					} else {
						currentConfig = newConfig
					}
				}

				if req.IPChanges != nil && currentConfig != nil {
					log.Printf("Gestionnaire de Config: Application des changements d'IP: %v", req.IPChanges)
					for oldIP, newIP := range req.IPChanges {
						for i, entry := range currentConfig.RoutingTable {
							if entry.IP == oldIP {
								currentConfig.RoutingTable[i].IP = newIP
							}
						}
						for u, ip := range currentConfig.UniverseIP {
							if ip == oldIP {
								currentConfig.UniverseIP[u] = newIP
							}
						}
					}
				}

				if req.ExportPath != "" && currentConfig != nil {
					log.Printf("Gestionnaire de Config: Exportation de la configuration vers %s", req.ExportPath)
					if err := config.Save(currentConfig, req.ExportPath); err != nil {
						log.Printf("ERREUR: Impossible de sauvegarder la configuration: %v", err)
					} else {
						log.Println("Gestionnaire de Config: Sauvegarde réussie.")
					}
					continue // On ne relance pas le pipeline pour une sauvegarde
				}
				
				faker = simulator.NewFaker(fakerUpdateChannel, fakerConfigOut, fakerModeSwitch, currentConfig)
				uiController.SetFaker(faker)
				
				uiController.UpdateWithNewConfig(currentConfig)

				if currentConfig != nil {
					pipelineCtx, cancelFunc := context.WithCancel(ctx)
					cancelPipeline = cancelFunc
					// ON PASSE LE CANAL DE MONITORING GLOBAL À CHAQUE NOUVEAU PIPELINE
					startPipeline(pipelineCtx, currentConfig, monitorChan, eHubUpdateChannel, fakerUpdateChannel, fakerConfigOut, fakerModeSwitch)
				}

			case <-ctx.Done():
				stopPipeline()
				return
			}
		}
	}()

	log.Println("Système démarré. En attente du chargement d'une configuration via l'UI...")
	w.ShowAndRun()

	log.Println("Arrêt complet de l'application.")
}

// startPipeline PREND MAINTENANT LE CANAL DE MONITORING EN ARGUMENT
func startPipeline(ctx context.Context, cfg *config.Config, monitorChan chan *ui.UniverseMonitorData, eHubUpdateOut, fakerUpdateOut chan *ehub.EHubUpdateMsg, fakerConfigOut chan *ehub.EHubConfigMsg, fakerModeSwitch chan bool) {
	log.Println("Pipeline: Démarrage des services...")
	
	// Canaux internes au pipeline
	rawPacketChannel := make(chan ehub.RawPacket, 1000)
	eHubConfigOut := make(chan *ehub.EHubConfigMsg, 50)
	artnetQueue := make(chan domain_artnet.LEDMessage, 10000)
	finalConfigIn := make(chan *ehub.EHubConfigMsg, 50)
	finalUpdateIn := make(chan *ehub.EHubUpdateMsg, 1000)

	// Création des services
	listener, _ := infra_ehub.NewListener(8765, rawPacketChannel)
	parser := app_ehub.NewParser()
	eHubService := app_ehub.NewService(rawPacketChannel, parser, eHubConfigOut, eHubUpdateOut)
	// LE PROCESSOR UTILISE LE CANAL DE MONITORING PASSÉ EN ARGUMENT
	processorService, physicalConfigOut := app_processor.NewService(finalConfigIn, finalUpdateIn, artnetQueue, monitorChan)
	sender, _ := infra_artnet.NewSender()

	go func() {
		isFakerActive := false
		log.Println("Aiguilleur: Démarré en mode LIVE.")
		for {
			select {
			case <-ctx.Done():
				log.Println("Aiguilleur: Arrêt.")
				return
			case mode := <-fakerModeSwitch:
				if mode != isFakerActive {
					isFakerActive = mode
					if isFakerActive { log.Println("Aiguilleur: Passage en mode FAKER.") } else { log.Println("Aiguilleur: Retour au mode LIVE.") }
				}
			case msg := <-fakerUpdateOut:
				if isFakerActive { finalUpdateIn <- msg }
			case msg := <-eHubUpdateOut:
				if !isFakerActive { finalUpdateIn <- msg }
			case msg := <-fakerConfigOut:
				if isFakerActive { finalConfigIn <- msg }
			case msg := <-eHubConfigOut:
				if !isFakerActive { finalConfigIn <- msg }
			}
		}
	}()

	listener.Start(ctx)
	eHubService.Start()
	processorService.Start()
	go sender.Run(ctx, artnetQueue)

	physicalConfigOut <- cfg
}
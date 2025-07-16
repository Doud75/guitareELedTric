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

type ConfigUpdateRequest struct {
	FilePath  string
	IPChanges map[string]string
}

func main() {
	log.Println("Démarrage du système...")

	// --- CONTEXTE GLOBAL ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- CANAUX DE COMMUNICATION ---
	configRequestChannel := make(chan ConfigUpdateRequest, 1)
	fakerModeSwitch := make(chan bool)
	eHubUpdateChannel := make(chan *ehub.EHubUpdateMsg, 1000)
	fakerUpdateChannel := make(chan *ehub.EHubUpdateMsg, 1000)
	fakerConfigOut := make(chan *ehub.EHubConfigMsg, 50)
	monitorChan := make(chan *ui.UniverseMonitorData, 100)

	// --- INITIALISATION DES COMPOSANTS SANS CONFIG ---
	faker := simulator.NewFaker(fakerUpdateChannel, fakerConfigOut, fakerModeSwitch, nil)

	// --- INITIALISATION DE L'UI ---
	a := app.New()
	a.Settings().SetTheme(&ui.ArtHeticTheme{})
	w := a.NewWindow("Guitare Hetic - Inspecteur ArtNet")
	uiController := ui.NewUIController(a, faker, monitorChan, func(filePath string, ipChanges map[string]string) {
		configRequestChannel <- ConfigUpdateRequest{
			FilePath:  filePath,
			IPChanges: ipChanges,
		}
	})
	ui.RunUI(uiController, w)

	// --- GOROUTINE : GESTIONNAIRE DE CONFIGURATION ---
	go func() {
		var currentConfig *config.Config
		var cancelPipeline context.CancelFunc = func() {} // Initialise avec une fonction vide

		// Fonction pour arrêter proprement le pipeline de traitement
		stopPipeline := func() {
			log.Println("Gestionnaire de Config: Arrêt du pipeline de traitement...")
			cancelPipeline()
		}
		
		// Boucle principale du gestionnaire
		for {
			select {
			case req := <-configRequestChannel:
				stopPipeline() // On arrête toujours l'ancien pipeline

				// CAS 1: Chargement depuis un fichier
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

				// CAS 2: Modification d'IP en mémoire
				if req.IPChanges != nil && currentConfig != nil {
					log.Printf("Gestionnaire de Config: Application des changements d'IP: %v", req.IPChanges)
					for oldIP, newIP := range req.IPChanges {
						// On modifie la table de routage
						for i, entry := range currentConfig.RoutingTable {
							if entry.IP == oldIP {
								currentConfig.RoutingTable[i].IP = newIP
							}
						}
						// On modifie la map des univers
						for u, ip := range currentConfig.UniverseIP {
							if ip == oldIP {
								currentConfig.UniverseIP[u] = newIP
							}
						}
					}
				}

				// Mise à jour des composants dépendants
				faker = simulator.NewFaker(fakerUpdateChannel, fakerConfigOut, fakerModeSwitch, currentConfig)
				uiController.UpdateWithNewConfig(currentConfig)

				// Redémarrage du pipeline si la config est valide
				if currentConfig != nil {
					pipelineCtx, cancelFunc := context.WithCancel(ctx)
					cancelPipeline = cancelFunc
					startPipeline(pipelineCtx, currentConfig, monitorChan, eHubUpdateChannel, fakerUpdateChannel, fakerConfigOut, fakerModeSwitch)
				}

			case <-ctx.Done():
				stopPipeline()
				return
			}
		}
	}()

	// --- DÉMARRAGE DE L'UI SUR LE THREAD PRINCIPAL ---
	log.Println("Système démarré. En attente du chargement d'une configuration via l'UI...")
	w.ShowAndRun()

	log.Println("Arrêt complet de l'application.")
}

// startPipeline initialise et lance tous les services de traitement de données.
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
	processorService, physicalConfigOut := app_processor.NewService(finalConfigIn, finalUpdateIn, artnetQueue, monitorChan)
	sender, _ := infra_artnet.NewSender()

	// Aiguilleur
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

	// Démarrage des services
	listener.Start(ctx)
	eHubService.Start()
	processorService.Start()
	go sender.Run(ctx, artnetQueue)

	// Injection de la config physique initiale
	physicalConfigOut <- cfg
}
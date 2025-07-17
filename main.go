package main

import (
	"context"
	app_ehub "guitarHetic/internal/application/ehub"
	app_processor "guitarHetic/internal/application/processor"
	app_artnet_monitor "guitarHetic/internal/application/artnet_monitor"
	infra_artnet_monitor "guitarHetic/internal/infrastructure/artnet_monitor"
	"fyne.io/fyne/v2/app"
	"guitarHetic/internal/config"
	domain_artnet "guitarHetic/internal/domain/artnet"
	"guitarHetic/internal/domain/ehub"
	infra_artnet "guitarHetic/internal/infrastructure/artnet"
	infra_ehub "guitarHetic/internal/infrastructure/ehub"
	"guitarHetic/internal/simulator"
	"guitarHetic/internal/ui"
	"log"
	"strconv"
	"strings"
	"sync"
)

var (
    isArtNetMonitoringActive bool
    monitoringMutex          sync.Mutex
)


func main() {
    log.Println("Démarrage du système...")

 toggleCallback := func(active bool) {
        monitoringMutex.Lock()
        isArtNetMonitoringActive = active
        monitoringMutex.Unlock()
        if active {
            log.Println("MAIN: Monitoring ArtNet ACTIVÉ.")
        } else {
            log.Println("MAIN: Monitoring ArtNet DÉSACTIVÉ.")
        }
    }
	
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    configRequestChannel := make(chan ui.ConfigUpdateRequest, 1)
    fakerModeSwitch := make(chan bool)
    eHubUpdateChannel := make(chan *ehub.EHubUpdateMsg, 1000)
    fakerUpdateChannel := make(chan *ehub.EHubUpdateMsg, 1000)
    fakerConfigOut := make(chan *ehub.EHubConfigMsg, 50)
    monitorChan := make(chan *ui.UniverseMonitorData, 100)

    var faker *simulator.Faker = nil

    a := app.New()
    a.Settings().SetTheme(&ui.ArtHeticTheme{})
    w := a.NewWindow("Guitare Hetic - Inspecteur ArtNet")
    uiController := ui.NewUIController(a, faker, monitorChan, func(req ui.ConfigUpdateRequest) {
        configRequestChannel <- req
    }, toggleCallback)
    ui.RunUI(uiController, w)

    go func() {
        var currentConfig *config.Config
        var processorService *app_processor.Service
        var cancelPipeline context.CancelFunc = func() {}

        stopPipeline := func() {
            log.Println("Gestionnaire de Config: Arrêt du pipeline de traitement...")
            cancelPipeline()
        }

        for {
            select {
            case req := <-configRequestChannel:
                if req.PatchFilePath != "" {
                    if processorService != nil {
                        patchMap, err := config.LoadPatchMapFromExcel(req.PatchFilePath)
                        if err != nil {
                            log.Printf("ERREUR: Impossible de charger le fichier de patch: %v", err)
                        } else {
                            processorService.SetPatchMap(patchMap)
                        }
                    }
                    continue
                }
                if req.ClearPatch {
                    if processorService != nil {
                        processorService.SetPatchMap(nil)
                    }
                    continue
                }
                if req.SetPatchingActive != nil {
                    if processorService != nil {
                        processorService.SetPatchingActive(*req.SetPatchingActive)
                    }
                    continue
                }

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
                    for key, newIP := range req.IPChanges {
                        if strings.HasPrefix(key, "universe:") {
                            universeIDStr := strings.TrimPrefix(key, "universe:")
                            universeID, err := strconv.Atoi(universeIDStr)
                            if err != nil {
                                log.Printf("ERREUR: Clé d'univers invalide: %s", key)
                                continue
                            }
                            log.Printf("  -> Changement spécifique pour l'univers %d vers l'IP %s", universeID, newIP)
                            if _, ok := currentConfig.UniverseIP[universeID]; ok {
                                currentConfig.UniverseIP[universeID] = newIP
                            }
                            for i, entry := range currentConfig.RoutingTable {
                                if entry.Universe == universeID {
                                    currentConfig.RoutingTable[i].IP = newIP
                                }
                            }
                        } else {
                            oldIP := key
                            log.Printf("  -> Changement global de l'IP %s vers %s", oldIP, newIP)
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
                }

                if req.ExportPath != "" && currentConfig != nil {
                    log.Printf("Gestionnaire de Config: Exportation de la configuration vers %s", req.ExportPath)
                    if err := config.Save(currentConfig, req.ExportPath); err != nil {
                        log.Printf("ERREUR: Impossible de sauvegarder la configuration: %v", err)
                    } else {
                        log.Println("Gestionnaire de Config: Sauvegarde réussie.")
                    }
                    continue
                }

                faker = simulator.NewFaker(fakerUpdateChannel, fakerConfigOut, fakerModeSwitch, currentConfig)
                uiController.SetFaker(faker)

                uiController.UpdateWithNewConfig(currentConfig)

                if currentConfig != nil {
                    pipelineCtx, cancelFunc := context.WithCancel(ctx)
                    cancelPipeline = cancelFunc
                    processorService = startPipeline(pipelineCtx, currentConfig, monitorChan, eHubUpdateChannel, fakerUpdateChannel, fakerConfigOut, fakerModeSwitch)
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

func startPipeline(ctx context.Context, cfg *config.Config, monitorChan chan *ui.UniverseMonitorData, eHubUpdateOut, fakerUpdateOut chan *ehub.EHubUpdateMsg, fakerConfigOut chan *ehub.EHubConfigMsg, fakerModeSwitch chan bool) *app_processor.Service {
    log.Println("Pipeline: Démarrage des services...")

    // --- Canaux existants ---
    rawPacketChannel := make(chan ehub.RawPacket, 1000)
    eHubConfigOut := make(chan *ehub.EHubConfigMsg, 50)
    artnetQueue := make(chan domain_artnet.LEDMessage, 10000)
    finalConfigIn := make(chan *ehub.EHubConfigMsg, 50)
    finalUpdateIn := make(chan *ehub.EHubUpdateMsg, 1000)

    // --- Pipeline eHub -> Processor -> ArtNet (existant) ---
    listener, _ := infra_ehub.NewListener(8765, rawPacketChannel)
    parser := app_ehub.NewParser()
    eHubService := app_ehub.NewService(rawPacketChannel, parser, eHubConfigOut, eHubUpdateOut)
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
                    if isFakerActive {
                        log.Println("Aiguilleur: Passage en mode FAKER.")
                    } else {
                        log.Println("Aiguilleur: Retour au mode LIVE.")
                    }
                }
            case msg := <-fakerUpdateOut:
                if isFakerActive {
                    finalUpdateIn <- msg
                }
            case msg := <-eHubUpdateOut:
                if !isFakerActive {
                    finalUpdateIn <- msg
                }
            case msg := <-fakerConfigOut:
                if isFakerActive {
                    finalConfigIn <- msg
                }
            case msg := <-eHubConfigOut:
                if !isFakerActive {
                    finalConfigIn <- msg
                }
            }
        }
    }()

    listener.Start(ctx)
    eHubService.Start()
    processorService.Start()
    go sender.Run(ctx, artnetQueue)
    physicalConfigOut <- cfg

    // --- NOUVEAU : PIPELINE DE MONITORING ARTNET ---
    log.Println("Pipeline: Démarrage du moniteur ArtNet (E9)...")
    artnetMonitorRawChan := make(chan infra_artnet_monitor.RawArtNetPacket, 100)
    artnetMonitorParsedChan := make(chan *domain_artnet.ArtNetDMXPacket, 100)

    artnetListener, err := infra_artnet_monitor.NewListener(6454, artnetMonitorRawChan)
    if err != nil {
        log.Printf("ERREUR: Impossible de démarrer le listener ArtNet: %v", err)
    } else {
        monitorService := app_artnet_monitor.NewService(artnetMonitorRawChan, artnetMonitorParsedChan)

        artnetListener.Start(ctx)
        monitorService.Start()

        // Goroutine qui consomme les paquets parsés et les logue
        go func() {
            for packet := range artnetMonitorParsedChan {
                monitoringMutex.Lock()
                isActive := isArtNetMonitoringActive
                monitoringMutex.Unlock()

                if isActive {
                    log.Println(packet.String())
                }
            }
        }()
    }
    // --- FIN DU NOUVEAU BLOC ---

    return processorService
}


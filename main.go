// File: main.go
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
    "guitarHetic/internal/ui"
)

func main() {
    log.Println("Démarrage du système eHuB → ArtNet…")

    cfg, err := config.Load("internal/config/routing.csv")
    if err != nil {
        log.Fatalf("Impossible de charger routing.csv : %v", err)
    }

    rawCh := make(chan ehub.RawPacket, 100)
    cfgCh := make(chan *ehub.EHubConfigMsg, 10)
    updCh := make(chan *ehub.EHubUpdateMsg, 100)
    artnetCh := make(chan domain_artnet.LEDMessage, 500)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    listener, err := infra_ehub.NewListener(8765, rawCh)
    if err != nil {
        log.Fatalf("Listener: %v", err)
    }
    sender, err := infra_artnet.NewSender(cfg.UniverseIP)
    if err != nil {
        log.Fatalf("Sender: %v", err)
    }

    parser := app_ehub.NewParser()
    ehubSvc := app_ehub.NewService(rawCh, parser, cfgCh, updCh)
    procSvc := app_processor.NewService(cfgCh, updCh, artnetCh)

    listener.Start()
    ehubSvc.Start()
    procSvc.Start()
    go sender.Run(ctx, artnetCh)

    ui.RunUI(ctx, cfg)
    cancel()
    log.Println("Arrêt complet.")
}

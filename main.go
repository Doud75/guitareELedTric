package main

import (
	"context"
	"fmt"
	app_ehub "guitarHetic/internal/application/ehub"
	"guitarHetic/internal/application/simulator"
	"guitarHetic/internal/config"
	domainArtnet "guitarHetic/internal/domain/artnet"
	"guitarHetic/internal/domain/ehub"
	artnetinfra "guitarHetic/internal/infrastructure/artnet"
	infra_ehub "guitarHetic/internal/infrastructure/ehub"
	"log"
	"time"
)

func main() {

	fmt.Println("Démarrage du programme (version simple)...")
	rawPacketChannel := make(chan ehub.RawPacket, 100)
	//@TODO move port to a config file
	const eHubPort = 8765

	cfg, _ := config.Load("internal/config/routing.csv")
    queue := make(chan domainArtnet.LEDMessage, len(cfg.RoutingTable))
    sender, _ := artnetinfra.NewSender(cfg.UniverseIP)

    ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	listener, err := infra_ehub.NewListener(eHubPort, rawPacketChannel)
	if err != nil {
		log.Fatalf("Erreur fatale: Impossible de créer le listener: %v", err)
	}
	
	parser := app_ehub.NewParser()
	parseService := app_ehub.NewService(rawPacketChannel, parser)
	
	listener.Start()
	parseService.Start()
    go sender.Run(ctx, queue)
    simulator.RunMovement(ctx, queue, cfg)

	for {
		time.Sleep(1 * time.Hour) 
	}
}
package main

import (
    "context"
    "fmt"

    "guitarHetic/internal/application/simulator"
    "guitarHetic/internal/config"
    domainArtnet "guitarHetic/internal/domain/artnet"
    artnetinfra "guitarHetic/internal/infrastructure/artnet"
)

func main() {
    fmt.Println("Starting Movement")
    cfg, _ := config.Load("internal/config/routing.csv")
    queue := make(chan domainArtnet.LEDMessage, len(cfg.RoutingTable))
    sender, _ := artnetinfra.NewSender(cfg.UniverseIP)
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go sender.Run(ctx, queue)
    simulator.RunMovement(ctx, queue, cfg)
    select {}
}

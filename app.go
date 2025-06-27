package main

import (
    "context"
    "log"
    "sort"
    "strconv"

    app_ehub "guitarHetic/internal/application/ehub"
    app_processor "guitarHetic/internal/application/processor"
    "guitarHetic/internal/config"
    domain_artnet "guitarHetic/internal/domain/artnet"
    "guitarHetic/internal/domain/ehub"
    infra_artnet "guitarHetic/internal/infrastructure/artnet"
    infra_ehub "guitarHetic/internal/infrastructure/ehub"
)

type UniverseDetail struct {
    Universe int      `json:"universe"`
    Ranges   []string `json:"ranges"`
}

type App struct {
    ctx      context.Context
    cancel   context.CancelFunc
    config   *config.Config
    sender   *infra_artnet.Sender
    listener *infra_ehub.Listener
}

func NewApp() *App {
    return &App{}
}

func (a *App) onStartup(ctx context.Context) {
    a.ctx, a.cancel = context.WithCancel(ctx)
    log.Println("Wails application starting up. Initializing background services.")

    if err := a.initializeServices(); err != nil {
        log.Fatalf("Failed to initialize services: %v", err)
    }

    log.Println("All background services are running.")
}

func (a *App) onShutdown(ctx context.Context) {
    log.Println("Wails application shutting down. Stopping services.")
    a.cancel()
    if a.sender != nil {
        a.sender.Close()
    }
    if a.listener != nil {
        // Assuming listener's connection is closed when its goroutine exits.
        // If it has a Close() method, call it here.
    }
    log.Println("Shutdown complete.")
}

func (a *App) initializeServices() error {
    appConfig, err := config.Load("internal/config/routing.csv")
    if err != nil {
        return err
    }
    a.config = appConfig

    rawPacketChannel := make(chan ehub.RawPacket, 100)
    configChannel := make(chan *ehub.EHubConfigMsg, 10)
    updateChannel := make(chan *ehub.EHubUpdateMsg, 100)
    artnetQueue := make(chan domain_artnet.LEDMessage, 500)

    const eHubPort = 8765
    listener, err := infra_ehub.NewListener(eHubPort, rawPacketChannel)
    if err != nil {
        return err
    }
    a.listener = listener

    sender, err := infra_artnet.NewSender(a.config.UniverseIP)
    if err != nil {
        return err
    }
    a.sender = sender

    parser := app_ehub.NewParser()
    eHubService := app_ehub.NewService(rawPacketChannel, parser, configChannel, updateChannel)
    processorService := app_processor.NewService(configChannel, updateChannel, artnetQueue)

    listener.Start()
    eHubService.Start()
    processorService.Start()
    go a.sender.Run(a.ctx, artnetQueue)

    return nil
}

func (a *App) GetControllers() []string {
    if a.config == nil {
        return []string{}
    }

    ipMap := make(map[string]struct{})
    for _, ip := range a.config.UniverseIP {
        ipMap[ip] = struct{}{}
    }

    ips := make([]string, 0, len(ipMap))
    for ip := range ipMap {
        ips = append(ips, ip)
    }
    sort.Strings(ips)
    return ips
}

func (a *App) GetDetails(ip string) []*UniverseDetail {
    if a.config == nil {
        return []*UniverseDetail{}
    }

    type entityRange struct {
        start, end int
    }
    universeRanges := make(map[int][]entityRange)

    for _, entry := range a.config.RoutingTable {
        if entry.IP == ip {
            universeRanges[entry.Universe] = append(universeRanges[entry.Universe], entityRange{start: entry.EntityID, end: entry.EntityID})
        }
    }

    details := make([]*UniverseDetail, 0)
    for u, ranges := range universeRanges {
        if len(ranges) == 0 {
            continue
        }

        sort.Slice(ranges, func(i, j int) bool {
            return ranges[i].start < ranges[j].start
        })

        merged := []string{}
        currentStart := ranges[0].start
        currentEnd := ranges[0].end

        for i := 1; i < len(ranges); i++ {
            if ranges[i].start == currentEnd+1 {
                currentEnd = ranges[i].end
            } else {
                if currentStart == currentEnd {
                    merged = append(merged, strconv.Itoa(currentStart))
                } else {
                    merged = append(merged, strconv.Itoa(currentStart)+"-"+strconv.Itoa(currentEnd))
                }
                currentStart = ranges[i].start
                currentEnd = ranges[i].end
            }
        }

        if currentStart == currentEnd {
            merged = append(merged, strconv.Itoa(currentStart))
        } else {
            merged = append(merged, strconv.Itoa(currentStart)+"-"+strconv.Itoa(currentEnd))
        }

        details = append(details, &UniverseDetail{
            Universe: u,
            Ranges:   merged,
        })
    }

    sort.Slice(details, func(i, j int) bool {
        return details[i].Universe < details[j].Universe
    })

    return details
}

func (a *App) Reload() error {
    log.Println("Frontend requested a configuration reload.")

    // This simple implementation reloads the config file.
    // The processor service will pick up the changes on the next eHub config message.
    // For an immediate effect, the processor service would need a dedicated method to be triggered here.
    newConfig, err := config.Load("internal/config/routing.csv")
    if err != nil {
        log.Printf("Error reloading config: %v", err)
        return err
    }
    a.config = newConfig
    log.Println("Configuration reloaded successfully from routing.csv.")
    return nil
}

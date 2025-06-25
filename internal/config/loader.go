package config

import "guitarHetic/internal/domain/artnet"

func Load() *artnet.ArtNetConfig {
    return &artnet.ArtNetConfig{
        UniverseToMapping: staticMapping(),
    }
}

func staticMapping() map[int]artnet.UniverseMapping {
    controllers := []string{
        "192.168.1.45",
        "192.168.1.46",
        "192.168.1.47",
        "192.168.1.48",
    }

    mapping := make(map[int]artnet.UniverseMapping)
    for u := 0; u < 128; u++ {
        ip := controllers[u/32]
        mapping[u] = artnet.UniverseMapping{
            IP:      ip,
            Indexes: nil,
        }
    }
    return mapping
}

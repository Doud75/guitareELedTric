package artnet

type LEDMessage struct {
    Universe int
    Index    int
    Color    [3]byte
}

type UniverseMapping struct {
    IP      string
    Indexes []int
}

type ArtNetConfig struct {
    UniverseToMapping map[int]UniverseMapping
}

// internal/config/loader.go
package config

import (
    "encoding/csv"
    "io"
    "os"
    "strconv"
    "strings"
)

type RawEntry struct {
    Start    int
    End      int
    IP       string
    Universe int
}

type RoutingEntry struct {
    EntityID  int
    IP        string
    Universe  int
    DMXOffset int
}

type Config struct {
    UniverseIP   map[int]string
    RoutingTable []RoutingEntry
}

func Load(path string) (*Config, error) {
    raws, err := loadRawEntries(path)
    if err != nil {
        return nil, err
    }

    universeIP := make(map[int]string)
    table := make([]RoutingEntry, 0)
    for _, e := range raws {
        for id := e.Start; id <= e.End; id++ {
            offset := (id - e.Start) * 3
            table = append(table, RoutingEntry{EntityID: id, IP: e.IP, Universe: e.Universe, DMXOffset: offset})
            universeIP[e.Universe] = e.IP
        }
    }

    return &Config{UniverseIP: universeIP, RoutingTable: table}, nil
}

func loadRawEntries(path string) ([]RawEntry, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    r := csv.NewReader(f)
    r.Comma = ';'
    if _, err := r.Read(); err != nil {
        return nil, err
    }

    var raws []RawEntry
    for {
        rec, err := r.Read()
        if err == io.EOF {
            break
        }
        if err != nil || len(rec) < 5 {
            continue
        }

        start, err := strconv.Atoi(strings.TrimSpace(rec[1]))
        if err != nil {
            continue
        }
        end, err := strconv.Atoi(strings.TrimSpace(rec[2]))
        if err != nil {
            continue
        }
        ip := strings.TrimSpace(rec[3])
        uni, err := strconv.Atoi(strings.TrimSpace(rec[4]))
        if err != nil {
            continue
        }

        raws = append(raws, RawEntry{Start: start, End: end, IP: ip, Universe: uni})
    }

    return raws, nil
}

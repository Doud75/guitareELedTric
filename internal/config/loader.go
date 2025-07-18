package config

import (
    "fmt"
    "github.com/xuri/excelize/v2"
    "log"
    "strconv"
    "strings"
)

type RawEntry struct {
    Name     string
    Start    int
    End      int
    IP       string
    Universe int
}

type RoutingEntry struct {
    Name      string
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
    raws, err := loadRawEntriesFromExcel(path)
    if err != nil {
        return nil, fmt.Errorf("impossible de charger les entrées depuis le fichier Excel: %w", err)
    }

    universeIP := make(map[int]string)
    table := make([]RoutingEntry, 0)

    for _, e := range raws {
        if e.Start == e.End {
            table = append(table, RoutingEntry{Name: e.Name, EntityID: e.Start, IP: e.IP, Universe: e.Universe, DMXOffset: 0})
            universeIP[e.Universe] = e.IP
            continue
        }

        for id := e.Start; id <= e.End; id++ {
            offset := (id - e.Start) * 3
            if offset < 512 {
                table = append(table, RoutingEntry{Name: e.Name, EntityID: id, IP: e.IP, Universe: e.Universe, DMXOffset: offset})
                universeIP[e.Universe] = e.IP
            }
        }
    }

    return &Config{UniverseIP: universeIP, RoutingTable: table}, nil
}

func loadRawEntriesFromExcel(path string) ([]RawEntry, error) {
    f, err := excelize.OpenFile(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    sheetName := f.GetSheetName(0)
    if sheetName == "" {
        return nil, fmt.Errorf("le fichier Excel ne contient aucune feuille")
    }

    rows, err := f.GetRows(sheetName)
    if err != nil {
        return nil, err
    }

    var raws []RawEntry
    for i, row := range rows {
        if i == 0 {
            continue
        }

        if len(row) < 5 {
            log.Printf("Config Loader: Ligne %d ignorée (pas assez de colonnes)", i+1)
            continue
        }

        name := strings.TrimSpace(row[0])

        start, err := strconv.Atoi(strings.TrimSpace(row[1]))
        if err != nil {
            log.Printf("Config Loader: Ligne %d ignorée (Entity Start invalide: '%s')", i+1, row[1])
            continue
        }

        end, err := strconv.Atoi(strings.TrimSpace(row[2]))
        if err != nil {
            log.Printf("Config Loader: Ligne %d ignorée (Entity End invalide: '%s')", i+1, row[2])
            continue
        }

        ip := strings.TrimSpace(row[3])
        if ip == "" {
            log.Printf("Config Loader: Ligne %d ignorée (IP manquante)", i+1)
            continue
        }

        uni, err := strconv.Atoi(strings.TrimSpace(row[4]))
        if err != nil {
            log.Printf("Config Loader: Ligne %d ignorée (ArtNet Universe invalide: '%s')", i+1, row[4])
            continue
        }

        raws = append(raws, RawEntry{Name: name, Start: start, End: end, IP: ip, Universe: uni})
    }

    return raws, nil
}

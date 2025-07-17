// internal/config/loader.go
package config

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"github.com/xuri/excelize/v2"
)

// MODIFICATION: Ajout du champ Name
type RawEntry struct {
	Name     string
	Start    int
	End      int
	IP       string
	Universe int
}

// MODIFICATION: Ajout du champ Name
type RoutingEntry struct {
	Name      string // Le nom de l'équipement ou de la ligne
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
		// Cas pour les entrées qui ne représentent qu'une seule entité (ex: projecteur)
		if e.Start == e.End {
			// MODIFICATION: On passe le nom de l'entrée brute
			table = append(table, RoutingEntry{Name: e.Name, EntityID: e.Start, IP: e.IP, Universe: e.Universe, DMXOffset: 0})
			universeIP[e.Universe] = e.IP
			continue
		}

		// Logique pour les bandes de LEDs (plages)
		for id := e.Start; id <= e.End; id++ {
			offset := (id - e.Start) * 3
			if offset < 512 {
				// MODIFICATION: On passe aussi le nom ici. Toutes les entités d'une même ligne auront le même nom.
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

        // MODIFICATION: On lit la première colonne pour le nom
		// Colonne A (index 0) : Name
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
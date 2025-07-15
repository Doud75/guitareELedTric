// internal/config/loader.go
package config

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"github.com/xuri/excelize/v2" // <-- Importer la nouvelle bibliothèque
)

// Les structs RawEntry, RoutingEntry, et Config ne changent pas.
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

// La fonction Load est la seule qui change son implémentation interne.
// Elle prendra maintenant le chemin vers un fichier .xlsx
func Load(path string) (*Config, error) {
	raws, err := loadRawEntriesFromExcel(path) // On appelle notre nouvelle fonction
	if err != nil {
		return nil, fmt.Errorf("impossible de charger les entrées depuis le fichier Excel: %w", err)
	}

	universeIP := make(map[int]string)
	table := make([]RoutingEntry, 0)
	
    // Cette partie de la logique ne change pas, car elle traite les RawEntry
    // de la même manière, qu'ils viennent d'un CSV ou d'un Excel.
	for _, e := range raws {
		// Le projecteur est un cas spécial avec une seule entité, Start et End sont identiques
		if e.Start == e.End {
			// Pour le projecteur, l'offset est 0 car il est au début de son univers.
			// Les 3 canaux sont R, G, B.
			table = append(table, RoutingEntry{EntityID: e.Start, IP: e.IP, Universe: e.Universe, DMXOffset: 0})
			universeIP[e.Universe] = e.IP
			continue
		}

		// Logique pour les bandes de LEDs
		for id := e.Start; id <= e.End; id++ {
			offset := (id - e.Start) * 3
			if offset < 512 { // Sécurité pour ne pas dépasser la taille d'un univers DMX
				table = append(table, RoutingEntry{EntityID: id, IP: e.IP, Universe: e.Universe, DMXOffset: offset})
				universeIP[e.Universe] = e.IP
			}
		}
	}

	return &Config{UniverseIP: universeIP, RoutingTable: table}, nil
}

// loadRawEntriesFromExcel est la nouvelle fonction de lecture.
func loadRawEntriesFromExcel(path string) ([]RawEntry, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// On suppose que les données sont sur la première feuille du classeur.
	// Si le nom est fixe (ex: "Feuil1"), vous pouvez l'utiliser.
	// GetSheetName(0) est plus robuste.
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("le fichier Excel ne contient aucune feuille")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}

	var raws []RawEntry
	// On commence à la ligne 2 pour sauter l'en-tête (index 1 dans la slice)
	for i, row := range rows {
		if i == 0 { 
			continue 
		}

		// On s'attend à au moins 5 colonnes: Name, Entity Start, Entity End, ArtNet IP, ArtNet Universe
		if len(row) < 5 {
			log.Printf("Config Loader: Ligne %d ignorée (pas assez de colonnes)", i+1)
			continue
		}

		// On suppose la structure suivante :
		// Colonne B (index 1) : Entity Start
		// Colonne C (index 2) : Entity End
		// Colonne D (index 3) : ArtNet IP
		// Colonne E (index 4) : ArtNet Universe

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

		raws = append(raws, RawEntry{Start: start, End: end, IP: ip, Universe: uni})
	}

	return raws, nil
}
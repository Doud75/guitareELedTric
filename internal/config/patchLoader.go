// internal/config/patch_loader.go
package config

import (
    "fmt"
    "github.com/xuri/excelize/v2"
    "log"
    "strconv"
)

// LoadPatchMapFromExcel lit un fichier de patch au format Excel.
// VERSION AMÉLIORÉE : Lit la PREMIÈRE feuille du classeur, quel que soit son nom.
func LoadPatchMapFromExcel(path string) (map[int]map[int][]int, error) {
    f, err := excelize.OpenFile(path)
    if err != nil {
        return nil, fmt.Errorf("impossible d'ouvrir le fichier de patch '%s': %w", path, err)
    }
    defer f.Close()

    // --- MODIFICATION CLÉ ---
    // On récupère la liste de toutes les feuilles.
    sheetList := f.GetSheetList()
    if len(sheetList) == 0 {
        return nil, fmt.Errorf("le fichier Excel ne contient aucune feuille de calcul")
    }
    // On prend la première feuille de la liste.
    sheetName := sheetList[0]
    log.Printf("Patch Loader: Lecture de la première feuille trouvée : '%s'", sheetName)
    // --- FIN DE LA MODIFICATION ---

    rows, err := f.GetRows(sheetName)
    if err != nil {
        // Cette erreur est maintenant beaucoup moins probable, mais on la garde par sécurité.
        return nil, fmt.Errorf("impossible de lire les lignes de la feuille '%s': %w", sheetName, err)
    }

    patchMap := make(map[int]map[int][]int)

    // On parcourt les lignes, en sautant l'en-tête (i=0)
    for i, row := range rows {
        if i == 0 {
            continue // On ignore la ligne d'en-tête
        }
        if len(row) < 3 {
            log.Printf("Patch Loader: Ligne %d ignorée (pas assez de colonnes)", i+1)
            continue
        }

        // Conversion des valeurs texte en nombres
        universe, errU := strconv.Atoi(row[0])
        source, errS := strconv.Atoi(row[1])
        destination, errD := strconv.Atoi(row[2])

        if errU != nil || errS != nil || errD != nil {
            log.Printf("Patch Loader: Ligne %d ignorée (format de nombre invalide)", i+1)
            continue
        }

        if source < 1 || source > 512 || destination < 1 || destination > 512 {
            log.Printf("Patch Loader: Ligne %d ignorée (canal hors de la plage 1-512)", i+1)
            continue
        }

        if _, ok := patchMap[universe]; !ok {
            patchMap[universe] = make(map[int][]int)
        }
        patchMap[universe][source] = append(patchMap[universe][source], destination)
    }

    log.Printf("Patch Loader: Fichier de patch chargé avec succès depuis la feuille '%s'. %d univers affectés.", sheetName, len(patchMap))
    return patchMap, nil
}

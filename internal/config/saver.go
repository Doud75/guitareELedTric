package config

import (
    "github.com/xuri/excelize/v2"
    "sort"
)

func Save(cfg *Config, path string) error {
    type groupKey struct {
        Name     string
        IP       string
        Universe int
    }
    groups := make(map[groupKey][]int)
    for _, entry := range cfg.RoutingTable {
        key := groupKey{Name: entry.Name, IP: entry.IP, Universe: entry.Universe}
        groups[key] = append(groups[key], entry.EntityID)
    }

    type outputRow struct {
        Name     string
        Start    int
        End      int
        IP       string
        Universe int
    }
    var outputRows []outputRow

    for key, ids := range groups {
        if len(ids) == 0 {
            continue
        }
        sort.Ints(ids)

        start, end := ids[0], ids[0]
        for i := 1; i < len(ids); i++ {
            if ids[i] == end+1 {
                end = ids[i]
            } else {
                outputRows = append(outputRows, outputRow{Name: key.Name, Start: start, End: end, IP: key.IP, Universe: key.Universe})
                start, end = ids[i], ids[i]
            }
        }
        outputRows = append(outputRows, outputRow{Name: key.Name, Start: start, End: end, IP: key.IP, Universe: key.Universe})
    }

    sort.Slice(outputRows, func(i, j int) bool {
        if outputRows[i].Universe != outputRows[j].Universe {
            return outputRows[i].Universe < outputRows[j].Universe
        }
        return outputRows[i].Start < outputRows[j].Start
    })

    f := excelize.NewFile()
    sheetName := "Feuil1"
    index, _ := f.NewSheet(sheetName)
    f.SetActiveSheet(index)

    headers := []string{"Name", "Entity Start", "Entity End", "ArtNet IP", "ArtNet Universe"}
    f.SetSheetRow(sheetName, "A1", &headers)

    for i, rowData := range outputRows {
        row := []interface{}{
            rowData.Name,
            rowData.Start,
            rowData.End,
            rowData.IP,
            rowData.Universe,
        }
        cell, _ := excelize.CoordinatesToCellName(1, i+2)
        f.SetSheetRow(sheetName, cell, &row)
    }

    f.SetColWidth(sheetName, "A", "E", 20)
    f.DeleteSheet("Sheet1")

    return f.SaveAs(path)
}

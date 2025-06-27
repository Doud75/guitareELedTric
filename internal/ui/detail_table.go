// File: internal/ui/detail_table.go
package ui

import (
    "fmt"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/widget"
    "strings"
)

func NewDetailTable() (*widget.Table, *[]UniRange) {
    details := make([]UniRange, 0)
    table := widget.NewTable(
        func() (int, int) { return len(details), 2 },
        func() fyne.CanvasObject { return widget.NewLabel("") },
        func(ci widget.TableCellID, o fyne.CanvasObject) {
            l := o.(*widget.Label)
            if ci.Col == 0 {
                l.SetText(fmt.Sprint(details[ci.Row].Universe))
            } else {
                parts := make([]string, len(details[ci.Row].Ranges))
                for i, rg := range details[ci.Row].Ranges {
                    parts[i] = fmt.Sprintf("%d–%d", rg[0], rg[1])
                }
                l.SetText(strings.Join(parts, ", "))
            }
        },
    )
    return table, &details
}

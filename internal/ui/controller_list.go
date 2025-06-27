// File: internal/ui/controller_list.go
package ui

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/widget"
)

func NewControllerList(ips []string, onSelect func(string)) *widget.List {
    list := widget.NewList(
        func() int { return len(ips) },
        func() fyne.CanvasObject { return widget.NewLabel("") },
        func(i widget.ListItemID, o fyne.CanvasObject) {
            o.(*widget.Label).SetText(ips[i])
        },
    )
    list.OnSelected = func(id widget.ListItemID) {
        onSelect(ips[id])
    }
    return list
}

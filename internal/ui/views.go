// internal/ui/views.go
package ui

import (
    "fmt"
    "strings"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
)

// buildIPListView construit la page affichant la liste des contrôleurs.
func buildIPListView(state *UIState, controller *UIController) fyne.CanvasObject {
    list := widget.NewList(
        func() int { return len(state.controllerIPs) },
        func() fyne.CanvasObject { return widget.NewLabel("") },
        func(i widget.ListItemID, o fyne.CanvasObject) {
            o.(*widget.Label).SetText(state.controllerIPs[i])
        },
    )

    // On connecte l'action de sélection au contrôleur.
    list.OnSelected = func(id widget.ListItemID) {
        controller.SelectIPAndShowDetails(state.controllerIPs[id])
    }

    return container.NewScroll(list)
}

// buildDetailView construit la page affichant les détails d'un contrôleur.
func buildDetailView(state *UIState, controller *UIController) fyne.CanvasObject {
    // Bouton pour revenir en arrière.
    backButton := widget.NewButton("Retour à la liste", func() {
        controller.GoBackToIPList()
    })

    // Titre indiquant quelle IP on visualise.
    title := widget.NewLabel(fmt.Sprintf("Détails pour le contrôleur : %s", state.selectedIP))
    title.TextStyle.Bold = true

    // La table des univers (similaire à l'ancien code).
    table := widget.NewTable(
        func() (int, int) { return len(state.selectedDetails), 2 },
        func() fyne.CanvasObject { return widget.NewLabel("") },
        func(ci widget.TableCellID, o fyne.CanvasObject) {
            l := o.(*widget.Label)
            if ci.Col == 0 {
                l.SetText(fmt.Sprintf("Univers %d", state.selectedDetails[ci.Row].Universe))
            } else {
                parts := make([]string, len(state.selectedDetails[ci.Row].Ranges))
                for i, rg := range state.selectedDetails[ci.Row].Ranges {
                    parts[i] = fmt.Sprintf("%d–%d", rg[0], rg[1])
                }
                l.SetText(strings.Join(parts, ", "))
            }
        },
    )
    table.SetColumnWidth(0, 120) // Ajuster la largeur de la colonne univers

    // On assemble la page : titre en haut, puis la table. Le bouton est dans la barre du haut.
    content := container.NewBorder(container.NewVBox(title, widget.NewSeparator()), nil, nil, nil, table)

    // On retourne la vue complète avec le bouton "Retour" en haut.
    return container.NewBorder(backButton, nil, nil, nil, content)
}

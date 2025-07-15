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

func buildDetailView(state *UIState, controller *UIController) fyne.CanvasObject {
    backButton := widget.NewButton("Retour à la liste", func() {
        controller.GoBackToIPList()
    })

    title := widget.NewLabel(fmt.Sprintf("Détails pour le contrôleur : %s", state.selectedIP))
    title.TextStyle.Bold = true

    // --- NOUVEAUX WIDGETS ---
    // 1. Le champ de saisie (input).
    // On le pré-remplit avec l'IP actuelle.
    ipInput := widget.NewEntry()
    ipInput.SetText(state.selectedIP)

    // 2. Le bouton de validation.
    validateButton := widget.NewButton("Valider le changement", func() {
        // Au clic, on lit la valeur ACTUELLE de l'input...
        currentInputValue := ipInput.Text
        // ...et on la passe au contrôleur pour qu'il la traite.
        controller.ValidateNewIP(currentInputValue)
    })
    // --- FIN DES NOUVEAUX WIDGETS ---

    // On crée un petit formulaire pour l'édition.
    editForm := container.NewVBox(
        widget.NewLabel("Modifier l'adresse IP :"),
        ipInput,
        validateButton,
    )

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
    table.SetColumnWidth(0, 120)

    // On assemble la page :
    // - Le titre et le formulaire d'édition en haut.
    // - La table des univers en dessous.
    topContent := container.NewVBox(title, widget.NewSeparator(), editForm, widget.NewSeparator())
    mainContent := container.NewBorder(topContent, nil, nil, nil, table)

    // On retourne la vue complète avec le bouton "Retour" tout en haut.
    return container.NewBorder(backButton, nil, nil, nil, mainContent)
}

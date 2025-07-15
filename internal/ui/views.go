// internal/ui/views.go
package ui

import (
    "fmt"
    "strings"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/theme" // Important pour les icônes !
    "fyne.io/fyne/v2/widget"
)

// *** NOUVELLE FONCTION ***
// buildHeader construit la barre de titre qui change en fonction du contexte.
func buildHeader(state *UIState, controller *UIController) fyne.CanvasObject {
    title := widget.NewLabel("Contrôleurs ArtNet")
    title.TextStyle.Bold = true

    // Si nous sommes dans la vue de détail, on ajoute un bouton "Retour".
    if state.CurrentView == DetailView {
        backButton := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
            controller.GoBackToIPList()
        })

        // On met le bouton à gauche et le titre à droite.
        return container.NewBorder(nil, nil, backButton, nil, title)
    }

    // Sinon, on affiche juste le titre.
    return container.NewPadded(title)
}

// *** VUE DE LA LISTE, VERSION 2.0 ***
func buildIPListView(state *UIState, controller *UIController) fyne.CanvasObject {
    // On utilise une List, mais chaque élément sera une carte interactive.
    list := widget.NewList(
        func() int {
            return len(state.controllerIPs)
        },
        // Le template pour chaque item : une carte avec une icône et un label.
        func() fyne.CanvasObject {
            return widget.NewCard("", "", container.NewHBox(
                widget.NewIcon(theme.ComputerIcon()),
                widget.NewLabel("template ip"),
            ))
        },
        // La fonction qui met à jour un item avec les bonnes données.
        func(i widget.ListItemID, o fyne.CanvasObject) {
            card := o.(*widget.Card)
            card.SetTitle(state.controllerIPs[i])
            card.SetSubTitle(fmt.Sprintf("%d univers configurés", len(state.allControllers[state.controllerIPs[i]])))
        },
    )

    // Quand un item est sélectionné, on navigue vers les détails.
    list.OnSelected = func(id widget.ListItemID) {
        controller.SelectIPAndShowDetails(state.controllerIPs[id])
        list.UnselectAll() // Pour un effet visuel plus propre
    }

    return list
}

// *** VUE DE DÉTAIL, VERSION 2.0 ***
func buildDetailView(state *UIState, controller *UIController) fyne.CanvasObject {
    // --- CARTE 1: Formulaire d'édition ---
    ipInput := widget.NewEntry()
    ipInput.SetText(state.selectedIP)

    // On ajoute une icône au bouton "Valider"
    validateButton := widget.NewButtonWithIcon("Sauvegarder", theme.ConfirmIcon(), func() {
        controller.ValidateNewIP(ipInput.Text)
    })
    validateButton.Importance = widget.HighImportance // Le rend plus visible (souvent en couleur)

    editForm := container.NewVBox(
        widget.NewLabel("Modifier l'adresse IP du contrôleur :"),
        ipInput,
        validateButton,
    )

    // On met le formulaire dans une carte pour le grouper visuellement.
    editCard := widget.NewCard("Configuration", "", editForm)

    // --- CARTE 2: Table des univers ---
    table := widget.NewTable(
        func() (int, int) { return len(state.selectedDetails), 2 },
        func() fyne.CanvasObject { return widget.NewLabel("") },
        func(ci widget.TableCellID, o fyne.CanvasObject) {
            l := o.(*widget.Label)
            detail := state.selectedDetails[ci.Row]
            if ci.Col == 0 {
                l.SetText(fmt.Sprintf("Univers %d", detail.Universe))
            } else {
                parts := make([]string, len(detail.Ranges))
                for i, rg := range detail.Ranges {
                    parts[i] = fmt.Sprintf("%d à %d", rg[0], rg[1])
                }
                l.SetText(strings.Join(parts, ", "))
            }
        },
    )
    table.SetColumnWidth(0, 120)

    // On met la table dans une carte pour la cohérence visuelle.
    // On utilise un conteneur scrollable au cas où la table serait très grande.
    tableCard := widget.NewCard("Univers & Plages d'Entités", "", container.NewScroll(table))

    // On retourne une boîte verticale contenant nos deux cartes.
    // Le container.Scroll englobe le tout pour s'adapter aux petites fenêtres.
    return container.NewScroll(container.NewVBox(editCard, tableCard))
}

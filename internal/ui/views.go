package ui

import (
    "fmt"
    "fyne.io/fyne/v2/layout"
    "strings"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
)

type SizedEntry struct {
    widget.Entry
    DesiredWidth float32
}

// NewSizedEntry est le constructeur pour notre widget.
func NewSizedEntry(width float32) *SizedEntry {
    entry := &SizedEntry{DesiredWidth: width}
    // C'est une étape essentielle pour lier notre logique custom au widget de base.
    entry.ExtendBaseWidget(entry)
    return entry
}

// MinSize est la méthode que Fyne appelle pour connaître la taille minimale.
// C'est ici que nous imposons notre largeur.
func (e *SizedEntry) MinSize() fyne.Size {
    // On récupère la taille minimale d'origine pour connaître la hauteur idéale.
    originalMin := e.Entry.MinSize()
    // On retourne une nouvelle taille avec NOTRE largeur et la hauteur d'origine.
    return fyne.NewSize(e.DesiredWidth, originalMin.Height)
}

// *** NOUVELLE FONCTION (RÉINTRODUITE ET CORRECTE) ***
// buildHeader construit la barre de navigation en haut de la fenêtre.
func buildHeader(state *UIState, controller *UIController) fyne.CanvasObject {
    title := widget.NewLabel("Inspecteur Art'Hetic")
    title.TextStyle.Bold = true

    var headerContent fyne.CanvasObject
    if state.CurrentView == DetailView {
        // On affiche un bouton "Retour" uniquement sur la page de détail.
        backButton := widget.NewButtonWithIcon("Retour", theme.NavigateBackIcon(), func() {
            controller.GoBackToIPList()
        })
        // On place le bouton à gauche et on laisse le titre prendre le reste de la place.
        headerContent = container.NewBorder(nil, nil, backButton, nil, title)
    } else {
        // Sur la page d'accueil, on affiche juste le titre.
        headerContent = title
    }

    // On ajoute un padding et un séparateur pour un meilleur look.
    return container.NewVBox(container.NewPadded(headerContent), widget.NewSeparator())
}

// *** VUE DE LA LISTE, VERSION 4.0 (ÉPURÉE) ***
func buildIPListView(state *UIState, controller *UIController) fyne.CanvasObject {
    list := widget.NewList(
        func() int {
            return len(state.controllerIPs)
        },
        // Le template pour chaque item : simple, propre.
        func() fyne.CanvasObject {
            // Un HBox pour le texte et une icône "flèche" pour indiquer l'action.
            return container.NewBorder(
                nil, nil, nil, widget.NewIcon(theme.NavigateNextIcon()),
                widget.NewLabel("Template IP"),
            )
        },
        // Mise à jour de l'item.
        func(i widget.ListItemID, o fyne.CanvasObject) {
            ip := state.controllerIPs[i]
            label := o.(*fyne.Container).Objects[0].(*widget.Label)
            label.SetText(ip)
        },
    )

    list.OnSelected = func(id widget.ListItemID) {
        controller.SelectIPAndShowDetails(state.controllerIPs[id])
        list.UnselectAll()
    }

    // La liste elle-même gère son scroll, pas besoin d'en ajouter un autre.
    return list
}

// *** VUE DE DÉTAIL, VERSION 4.0 (CORRIGÉE SELON VOS DEMANDES) ***
func buildDetailView(state *UIState, controller *UIController) fyne.CanvasObject {

    // --- SECTION ÉDITION : avec notre input à taille contrôlée ---

    // On utilise notre nouveau widget au lieu de widget.NewEntry().
    // 200.0 est une bonne largeur de départ pour une IP, facilement ajustable ici.
    ipInput := NewSizedEntry(200.0)
    ipInput.SetText(state.selectedIP)

    validateButton := widget.NewButtonWithIcon("Sauvegarder", theme.ConfirmIcon(), func() {
        controller.ValidateNewIP(ipInput.Text)
    })
    validateButton.Importance = widget.HighImportance

    // On garde le layout HBox + Spacer, qui est correct.
    // Maintenant que ipInput a une MinSize correcte, le layout fonctionnera comme prévu.
    editLine := container.NewHBox(
        widget.NewLabel("Nouvelle IP :"),
        ipInput, // Notre widget personnalisé
        validateButton,
        layout.NewSpacer(), // Le spacer prendra toujours l'espace restant
    )

    // --- SECTION DONNÉES : On utilise la technique de boucle 'for' qui fonctionne ---
    universeItems := []fyne.CanvasObject{
        widget.NewLabelWithStyle("Univers & Plages d'Entités", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
    }

    for _, detail := range state.selectedDetails {
        labelUnivers := widget.NewLabel(fmt.Sprintf("Univers ArtNet %d", detail.Universe))

        parts := make([]string, len(detail.Ranges))
        for i, rg := range detail.Ranges {
            parts[i] = fmt.Sprintf("%d à %d", rg[0], rg[1])
        }
        labelRanges := widget.NewLabel(strings.Join(parts, ", "))

        row := container.NewBorder(nil, nil, labelUnivers, nil, labelRanges)
        universeItems = append(universeItems, row)
    }

    universeList := container.NewVBox(universeItems...)

    // --- ASSEMBLAGE FINAL DE LA VUE ---
    return container.NewScroll(
        container.NewVBox(
            container.NewPadded(editLine),
            widget.NewSeparator(),
            container.NewPadded(universeList),
        ),
    )
}

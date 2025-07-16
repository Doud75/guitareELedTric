// internal/ui/state.go
package ui

import (
    "fyne.io/fyne/v2" // Importer fyne pour fyne.CanvasObject
    "guitarHetic/internal/config"
    "sync"
)

// ViewName est un type pour définir nos vues de manière sûre.
type ViewName string

const (
    // Nos trois vues possibles dans l'application.
    IPListView   ViewName = "ip_list"
    DetailView   ViewName = "detail"
    UniverseView ViewName = "universe_view"
)

// UIState contient l'état complet de l'interface utilisateur.
type UIState struct {
    // --- Données de base ---
    allControllers map[string]map[int][][2]int

    // --- État dynamique ---
    CurrentView      ViewName
    controllerIPs    []string
    selectedIP       string
    selectedDetails  []UniRange
    selectedUniverse int

    // --- Données et widgets pour la vue de monitoring ---
    // Le Mutex protège l'accès aux slices de widgets lors de leur création/destruction.
    ledStateMutex sync.RWMutex

    // CHANGEMENT: On stocke les widgets eux-mêmes, et non plus les couleurs.
    ledInputWidgets  []*LedWidget
    ledOutputWidgets []*LedWidget

    // CHANGEMENT: On garde une référence au contenu de la vue pour pouvoir le rafraîchir.
    universeViewContent fyne.CanvasObject
}

// NewUIState initialise l'état de l'application.
func NewUIState(cfg *config.Config) *UIState {
    ips, ctrlMap := BuildModel(cfg)
    return &UIState{
        allControllers: ctrlMap,
        controllerIPs:  ips,
        CurrentView:    IPListView,
    }
}

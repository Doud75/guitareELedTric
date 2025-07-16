// internal/ui/state.go
package ui

import (
    "fyne.io/fyne/v2"
    "guitarHetic/internal/config"
    "sync"
)

// ViewName est un type pour définir nos vues de manière sûre.
type ViewName string

const (
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

    // CHANGEMENT: Ajout d'une pile pour gérer l'historique de navigation.
    viewStack []ViewName

    // --- Données et widgets pour la vue de monitoring ---
    ledStateMutex       sync.RWMutex
    ledInputWidgets     []*LedWidget
    ledOutputWidgets    []*LedWidget
    universeViewContent fyne.CanvasObject
}

// NewUIState initialise l'état de l'application.
func NewUIState(cfg *config.Config) *UIState {
    ips, ctrlMap := BuildModel(cfg)
    return &UIState{
        allControllers: ctrlMap,
        controllerIPs:  ips,
        CurrentView:    IPListView,          // La vue de départ
        viewStack:      make([]ViewName, 0), // Initialisation de la pile vide
    }
}

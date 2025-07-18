package ui

import (
    "fyne.io/fyne/v2"
    "guitarHetic/internal/config"
    "sync"
)

type ViewName string

const (
    IPListView   ViewName = "ip_list"
    DetailView   ViewName = "detail"
    UniverseView ViewName = "universe_view"
)

type UIState struct {
    allControllers      map[string]map[int][][2]int
    CurrentView         ViewName
    controllerIPs       []string
    selectedIP          string
    selectedDetails     []UniRange
    selectedUniverse    int
    viewStack           []ViewName
    ledStateMutex       sync.RWMutex
    ledInputWidgets     []*LedWidget
    ledOutputWidgets    []*LedWidget
    universeViewContent fyne.CanvasObject
    lastOpenedFolder    fyne.ListableURI
}

func NewUIState(cfg *config.Config) *UIState {
    ips, ctrlMap := BuildModel(cfg)
    return &UIState{
        allControllers: ctrlMap,
        controllerIPs:  ips,
        CurrentView:    IPListView,
        viewStack:      make([]ViewName, 0),
    }
}

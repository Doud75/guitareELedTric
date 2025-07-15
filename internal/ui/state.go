package ui

import "guitarHetic/internal/config"

// ViewName est un type pour définir nos vues de manière sûre.
type ViewName string

const (
    // Nos deux vues possibles dans l'application.
    IPListView ViewName = "ip_list"
    DetailView ViewName = "detail"
)

// UIState contient l'état complet de l'interface utilisateur.
// C'est la source de vérité unique.
type UIState struct {
    // --- Données de base (chargées une seule fois) ---
    allControllers map[string]map[int][][2]int

    // --- État dynamique (change avec les interactions) ---
    CurrentView     ViewName   // La vue actuellement affichée.
    controllerIPs   []string   // La liste des IPs à afficher dans la vue principale.
    selectedIP      string     // L'IP qui a été sélectionnée par l'utilisateur.
    selectedDetails []UniRange // Les détails de l'IP sélectionnée, prêts à être affichés.
}

// NewUIState initialise l'état de l'application.
func NewUIState(cfg *config.Config) *UIState {
    ips, ctrlMap := BuildModel(cfg) // On réutilise la logique de préparation existante.
    return &UIState{
        allControllers: ctrlMap,
        controllerIPs:  ips,
        CurrentView:    IPListView, // L'application démarre sur la liste des IPs.
    }
}

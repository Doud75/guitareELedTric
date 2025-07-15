package ui

import (
    "sort"
)

// UIController est le cerveau qui manipule l'état en réponse aux actions.
type UIController struct {
    state *UIState
    // Ce callback sera appelé pour notifier à la vue principale de se redessiner.
    onStateChange func()
}

func NewUIController(state *UIState) *UIController {
    return &UIController{
        state:         state,
        onStateChange: func() {}, // Initialisation avec un callback vide pour éviter les panics.
    }
}

// SetUpdateCallback permet à la vue de s'enregistrer pour les mises à jour.
func (c *UIController) SetUpdateCallback(callback func()) {
    c.onStateChange = callback
}

// SelectIPAndShowDetails est appelée lorsqu'un utilisateur clique sur une IP.
func (c *UIController) SelectIPAndShowDetails(ip string) {
    // 1. Mettre à jour l'état avec les informations de la sélection.
    c.state.selectedIP = ip

    // Calculer les détails pour la vue suivante.
    entries := c.state.allControllers[ip]
    details := make([]UniRange, 0, len(entries))
    for u, ranges := range entries {
        details = append(details, UniRange{Universe: u, Ranges: ranges})
    }
    sort.Slice(details, func(i, j int) bool {
        return details[i].Universe < details[j].Universe
    })
    c.state.selectedDetails = details

    // 2. Changer la vue actuelle pour la vue de détails.
    c.state.CurrentView = DetailView

    // 3. Notifier l'interface qu'elle doit se redessiner complètement.
    c.onStateChange()
}

// GoBackToIPList est appelée par le bouton "Retour".
func (c *UIController) GoBackToIPList() {
    // 1. Changer la vue pour revenir à la liste.
    c.state.CurrentView = IPListView

    // (Optionnel mais propre) Réinitialiser la sélection.
    c.state.selectedIP = ""
    c.state.selectedDetails = nil

    // 2. Notifier l'interface de se redessiner.
    c.onStateChange()
}

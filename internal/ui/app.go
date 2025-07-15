// internal/ui/view.go
package ui

import (
    "context"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/widget"
    "guitarHetic/internal/config"
)

// RunUI initialise et lance l'application graphique.
func RunUI(ctx context.Context, cfg *config.Config) {
    a := app.New()
    w := a.NewWindow("Inspecteur de Contrôleurs ArtNet")

    // 1. Créer le modèle d'état et le contrôleur.
    state := NewUIState(cfg)
    controller := NewUIController(state)

    // 2. Définir la fonction de routage qui reconstruit l'interface.
    //    Cette fonction sera appelée à chaque changement d'état.
    buildAndUpdateView := func() {
        var viewContent fyne.CanvasObject

        // C'est le cœur du routeur : il choisit quelle vue construire.
        switch state.CurrentView {
        case IPListView:
            viewContent = buildIPListView(state, controller)
        case DetailView:
            viewContent = buildDetailView(state, controller)
        default:
            viewContent = widget.NewLabel("Erreur : Vue inconnue")
        }

        // On met à jour le contenu de la fenêtre avec la nouvelle vue.
        w.SetContent(viewContent)
    }

    // 3. Connecter le contrôleur à la fonction de mise à jour.
    controller.SetUpdateCallback(buildAndUpdateView)

    // 4. Construire la vue initiale.
    buildAndUpdateView()

    w.Resize(fyne.NewSize(800, 600))
    w.ShowAndRun()
}

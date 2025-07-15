package ui

import (
    "context"
    "fyne.io/fyne/v2/widget"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container" // On aura besoin de container
    "guitarHetic/internal/config"
)

func RunUI(ctx context.Context, cfg *config.Config, physicalConfigOut chan<- *config.Config) {
    a := app.New()
    // On applique notre thème personnalisé à toute l'application.
    a.Settings().SetTheme(&myTheme{})

    w := a.NewWindow("Guitare Hetic - Inspecteur ArtNet")

    state := NewUIState(cfg)
    controller := NewUIController(state, cfg, physicalConfigOut)

    buildAndUpdateView := func() {
        var viewContent fyne.CanvasObject

        // Le routeur est le même, il choisit le contenu principal
        switch state.CurrentView {
        case IPListView:
            viewContent = buildIPListView(state, controller)
        case DetailView:
            viewContent = buildDetailView(state, controller)
        default:
            viewContent = widget.NewLabel("Erreur : Vue inconnue")
        }

        // *** NOUVEAU : On enveloppe le contenu dans une mise en page globale ***
        // On crée une coquille avec une barre de titre/navigation en haut.
        header := buildHeader(state, controller)

        // Le contenu principal est placé au centre, avec un léger padding.
        paddedContent := container.NewPadded(viewContent)

        // On assemble la vue finale et on la place dans la fenêtre.
        w.SetContent(container.NewBorder(header, nil, nil, nil, paddedContent))
    }

    controller.SetUpdateCallback(buildAndUpdateView)
    buildAndUpdateView()

    w.Resize(fyne.NewSize(900, 700)) // Un peu plus grand
    w.ShowAndRun()
}

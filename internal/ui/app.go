// internal/ui/app.go
package ui

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
    "guitarHetic/internal/config"
    "guitarHetic/internal/simulator"
)

// RunUI initialise et lance l'application graphique Fyne.
// C'est le point d'entrée principal de l'interface utilisateur.
func RunUI(
    cfg *config.Config,
    physicalConfigOut chan<- *config.Config,
    faker *simulator.Faker,
// Le canal de monitoring est maintenant passé en argument depuis main.go
    monitorChanIn <-chan *UniverseMonitorData,
) {
    // Initialisation standard de l'application Fyne
    a := app.New()
    a.Settings().SetTheme(&ArtHeticTheme{})
    w := a.NewWindow("Guitare Hetic - Inspecteur ArtNet")

    state := NewUIState(cfg)
    controller := NewUIController(state, cfg, physicalConfigOut, a, faker, monitorChanIn)

    mainMenu := buildMainMenu(controller)
    w.SetMainMenu(mainMenu)

    buildAndUpdateView := func() {
        var viewContent fyne.CanvasObject

        switch state.CurrentView {
        case IPListView:
            viewContent = buildIPListView(state, controller)
        case DetailView:
            viewContent = buildDetailView(state, controller)
        case UniverseView:
            viewContent = state.universeViewContent

        default:
            viewContent = widget.NewLabel("Erreur : Vue inconnue")
        }

        // *** CORRECTION CRUCIALE : On réintroduit le Header ***
        // On crée une coquille d'application qui place notre contenu de vue
        // sous une barre de navigation persistante.
        header := buildHeader(state, controller)

        // Le contenu principal est placé sous le header.
        fullContent := container.NewBorder(header, nil, nil, nil, viewContent)

        w.SetContent(fullContent)
    }

    controller.SetUpdateCallback(buildAndUpdateView)
    buildAndUpdateView()

    w.Resize(fyne.NewSize(1024, 768))
    w.SetCloseIntercept(func() {
        controller.QuitApp()
    })

    w.ShowAndRun()
}

func buildMainMenu(controller *UIController) *fyne.MainMenu {
    fileMenu := fyne.NewMenu("Art'hetic",
        fyne.NewMenuItem("Quitter", func() {
            controller.QuitApp()
        }),
    )

    // Sous-menu pour les couleurs unies du Faker
    solidColorItem := fyne.NewMenuItem("Couleur Unie", nil)
    solidColorItem.ChildMenu = fyne.NewMenu("",
        fyne.NewMenuItem("Blanc", func() { controller.RunFakerCommand("white") }),
        fyne.NewMenuItem("Rouge", func() { controller.RunFakerCommand("red") }),
        fyne.NewMenuItem("Vert", func() { controller.RunFakerCommand("green") }),
        fyne.NewMenuItem("Bleu", func() { controller.RunFakerCommand("blue") }),
        fyne.NewMenuItemSeparator(),
        fyne.NewMenuItem("Noir (Éteindre)", func() { controller.RunFakerCommand("black") }),
    )

    animationsItem := fyne.NewMenuItem("Animations", nil)
    animationsItem.ChildMenu = fyne.NewMenu("",
        fyne.NewMenuItem("Vague Animée", func() { controller.RunFakerCommand("animation") }),
        fyne.NewMenuItem("Arrêter l'animation", func() { controller.RunFakerCommand("stop") }),
    )

	// On assemble le menu "Faker".
	fakerMenu := fyne.NewMenu("Faker",
		solidColorItem,
		animationsItem,
		// --- AJOUT DE L'OPTION DE DÉSACTIVATION ---
		fyne.NewMenuItemSeparator(), // Un séparateur pour la clarté visuelle
		fyne.NewMenuItem("Retour au mode LIVE (eHub)", func() {
			controller.SwitchToLiveMode()
		}),
	)

	return fyne.NewMainMenu(fileMenu, fakerMenu)
}
// internal/ui/app.go
package ui

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/widget"
    "guitarHetic/internal/config"
    "guitarHetic/internal/simulator"
)

// RunUI initialise et lance l'application graphique.
func RunUI(cfg *config.Config, physicalConfigOut chan<- *config.Config, faker *simulator.Faker) {
    a := app.New()
    w := a.NewWindow("Inspecteur de Contrôleurs ArtNet")

    // 1. Créer le modèle d'état et le contrôleur.
    state := NewUIState(cfg)
    controller := NewUIController(state, cfg, physicalConfigOut, a, faker)

    // 2. Construire le menu principal.
    mainMenu := buildMainMenu(controller)
    w.SetMainMenu(mainMenu)

    // 3. Définir la fonction de routage qui reconstruit l'interface.
    buildAndUpdateView := func() {
        var viewContent fyne.CanvasObject

        // Ce routeur simple choisit quelle vue construire en fonction de l'état.
        switch state.CurrentView {
        case IPListView:
            viewContent = buildIPListView(state, controller)
        case DetailView:
            viewContent = buildDetailView(state, controller)
        default:
            viewContent = widget.NewLabel("Erreur : Vue inconnue")
        }

        w.SetContent(viewContent)
    }

    // 4. Connecter le contrôleur à la fonction de mise à jour.
    controller.SetUpdateCallback(buildAndUpdateView)

    // 5. Construire la vue initiale.
    buildAndUpdateView()

    w.Resize(fyne.NewSize(800, 600))
    // S'assurer que cliquer sur la croix de la fenêtre appelle notre logique de "quit".
    w.SetCloseIntercept(func() {
        controller.QuitApp()
    })

    w.ShowAndRun()
}

// buildMainMenu est une fonction d'aide pour garder RunUI propre.
func buildMainMenu(controller *UIController) *fyne.MainMenu {
    // Menu "Fichier"
    fileMenu := fyne.NewMenu("Art'hetic",
        fyne.NewMenuItem("Quitter", func() {
            controller.QuitApp()
        }),
    )

    // --- Menu "Faker" avec sous-menus ---

    // On crée l'élément de menu parent. Son action est `nil`.
    solidColorItem := fyne.NewMenuItem("Couleur Unie", nil)
    // On attache le sous-menu à sa propriété .ChildMenu.
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

    // On assemble le menu "Faker" avec les éléments contenant les sous-menus.
    fakerMenu := fyne.NewMenu("Faker", solidColorItem, animationsItem)

    return fyne.NewMainMenu(fileMenu, fakerMenu)
}

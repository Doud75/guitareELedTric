// internal/ui/app.go
package ui

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/canvas"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/data/binding"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/storage"
    "fyne.io/fyne/v2/widget"
    "image/color"
    "log"
)

func RunUI(
    controller *UIController, // On reçoit directement le contrôleur pré-configuré
    w fyne.Window,            // On reçoit la fenêtre de l'extérieur
) {
    mainMenu := buildMainMenu(controller, w)
    w.SetMainMenu(mainMenu)

    buildAndUpdateView := func() {
        var viewContent fyne.CanvasObject

        if !controller.IsConfigLoaded() {
            viewContent = container.NewCenter(
                widget.NewLabel("Veuillez charger un fichier de configuration via le menu 'Art'hetic' -> 'Charger...'"),
            )
        } else {
            switch controller.state.CurrentView {
            case IPListView:
                viewContent = buildIPListView(controller.state, controller)
            case DetailView:
                viewContent = buildDetailView(controller.state, controller)
            case UniverseView:
                viewContent = controller.state.universeViewContent
            default:
                viewContent = widget.NewLabel("Erreur : Vue inconnue")
            }
        }

        header := buildHeader(controller.state, controller)
        fullContent := container.NewBorder(header, nil, nil, nil, viewContent)
        w.SetContent(fullContent)
    }

    controller.SetUpdateCallback(buildAndUpdateView)
    buildAndUpdateView()

    w.Resize(fyne.NewSize(1324, 768))
}

// MODIFICATION: Version la plus sûre de buildMainMenu
func buildMainMenu(controller *UIController, parentWindow fyne.Window) *fyne.MainMenu {
    xlsxFilter := storage.NewExtensionFileFilter([]string{".xlsx"})

    // --- Menu "Art'hetic" (inchangé) ---
    fileMenu := fyne.NewMenu("Art'hetic",
        fyne.NewMenuItem("Charger configuration...", func() {
            fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
                if err != nil || reader == nil {
                    return
                }
                controller.LoadNewConfigFile(reader.URI())
                reader.Close()
            }, parentWindow)
            if controller.state.lastOpenedFolder != nil {
                fileDialog.SetLocation(controller.state.lastOpenedFolder)
            }
            fileDialog.SetFilter(xlsxFilter)
            fileDialog.Show()
        }),
        fyne.NewMenuItem("Sauvegarder la configuration sous...", func() {
            fileDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
                if err != nil || writer == nil {
                    return
                }
                controller.SaveConfigFile(writer.URI().Path())
                writer.Close()
            }, parentWindow)
            fileDialog.SetFileName("routing_export.xlsx")
            fileDialog.SetFilter(xlsxFilter)
            fileDialog.Show()
        }),
        fyne.NewMenuItem("Quitter", func() {
            controller.QuitApp()
        }),
    )

    // --- Menu "Faker" (inchangé) ---
    showColorPicker := func() {
        r, g, b, w := binding.NewFloat(), binding.NewFloat(), binding.NewFloat(), binding.NewFloat()
        preview := canvas.NewRectangle(color.Black)
        preview.SetMinSize(fyne.NewSize(100, 50))
        updatePreview := func() {
            vr, _ := r.Get()
            vg, _ := g.Get()
            vb, _ := b.Get()
            preview.FillColor = color.NRGBA{R: uint8(vr), G: uint8(vg), B: uint8(vb), A: 0xFF}
            preview.Refresh()
        }
        r.AddListener(binding.NewDataListener(updatePreview))
        g.AddListener(binding.NewDataListener(updatePreview))
        b.AddListener(binding.NewDataListener(updatePreview))
        content := container.NewVBox(
            container.NewCenter(preview), widget.NewSeparator(),
            container.NewGridWithColumns(3, widget.NewLabel("Rouge"), widget.NewSliderWithData(0, 255, r), widget.NewLabelWithData(binding.FloatToStringWithFormat(r, "%.0f"))),
            container.NewGridWithColumns(3, widget.NewLabel("Vert"), widget.NewSliderWithData(0, 255, g), widget.NewLabelWithData(binding.FloatToStringWithFormat(g, "%.0f"))),
            container.NewGridWithColumns(3, widget.NewLabel("Bleu"), widget.NewSliderWithData(0, 255, b), widget.NewLabelWithData(binding.FloatToStringWithFormat(b, "%.0f"))),
            container.NewGridWithColumns(3, widget.NewLabel("Blanc"), widget.NewSliderWithData(0, 255, w), widget.NewLabelWithData(binding.FloatToStringWithFormat(w, "%.0f"))),
        )
        dialog.ShowCustomConfirm("Choisir une couleur personnalisée", "Valider", "Annuler", content,
            func(ok bool) {
                if !ok {
                    return
                }
                vr, _ := r.Get()
                vg, _ := g.Get()
                vb, _ := b.Get()
                vw, _ := w.Get()
                controller.RunFakerCustomColor(uint8(vr), uint8(vg), uint8(vb), uint8(vw))
            }, parentWindow)
    }
    solidColorItem := fyne.NewMenuItem("Couleur Unie", nil)
    solidColorItem.ChildMenu = fyne.NewMenu("",
        fyne.NewMenuItem("Blanc", func() { controller.RunFakerCommand("white") }),
        fyne.NewMenuItem("Rouge", func() { controller.RunFakerCommand("red") }),
        fyne.NewMenuItem("Vert", func() { controller.RunFakerCommand("green") }),
        fyne.NewMenuItem("Bleu", func() { controller.RunFakerCommand("blue") }),
        fyne.NewMenuItemSeparator(),
        fyne.NewMenuItem("Noir (Éteindre)", func() { controller.RunFakerCommand("black") }),
        fyne.NewMenuItemSeparator(),
        fyne.NewMenuItem("Couleur Personnalisée...", showColorPicker),
    )
    animationsItem := fyne.NewMenuItem("Animations", nil)
    animationsItem.ChildMenu = fyne.NewMenu("",
        fyne.NewMenuItem("Vague Animée", func() { controller.RunFakerCommand("animation") }),
        fyne.NewMenuItem("Arrêter l'animation", func() { controller.RunFakerCommand("stop") }),
    )
    fakerMenu := fyne.NewMenu("Faker",
        solidColorItem, animationsItem, fyne.NewMenuItemSeparator(),
        fyne.NewMenuItem("Retour au mode LIVE (eHub)", func() { controller.SwitchToLiveMode() }),
    )

    // --- Menu "Patching" (Version ultra-simple et stable) ---
    var isPatchingActive = false // Variable locale pour garder une trace de l'état

    patchMenu := fyne.NewMenu("Patching",
        fyne.NewMenuItem("Charger un fichier de Patch (.xlsx)...", func() {
            fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
                if err != nil || reader == nil {
                    return
                }
                controller.LoadPatchFile(reader.URI())
                reader.Close()
            }, parentWindow)
            fileDialog.SetFilter(xlsxFilter)
            fileDialog.Show()
        }),
        fyne.NewMenuItem("Vider le Patch actuel", func() {
            controller.ClearPatch()
        }),
        fyne.NewMenuItemSeparator(),
        fyne.NewMenuItem("Activer/Désactiver le Patching", func() {
            // On bascule l'état et on informe le contrôleur
            isPatchingActive = !isPatchingActive
            controller.SetPatchingActive(isPatchingActive)
            if isPatchingActive {
                log.Println("UI: Commande 'Activer Patching' envoyée.")
            } else {
                log.Println("UI: Commande 'Désactiver Patching' envoyée.")
            }
        }),
    )

    // --- ASSEMBLAGE DU MENU PRINCIPAL ---
    return fyne.NewMainMenu(fileMenu, fakerMenu, patchMenu)
}

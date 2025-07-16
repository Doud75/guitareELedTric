// internal/ui/app.go
package ui

import (
	"fyne.io/fyne/v2"
	// "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	// "guitarHetic/internal/config"
	// "guitarHetic/internal/simulator"
	"image/color"
)

func RunUI(
	controller *UIController, // On reçoit directement le contrôleur pré-configuré
	w fyne.Window,            // On reçoit la fenêtre de l'extérieur
) {
	// On ne crée plus l'app, la window, le state ou le controller ici.
	// On se contente de construire l'interface.
	
	mainMenu := buildMainMenu(controller, w)
	w.SetMainMenu(mainMenu)

	buildAndUpdateView := func() {
		var viewContent fyne.CanvasObject
		
		// Si aucune config n'est chargée, on affiche un message.
		if !controller.IsConfigLoaded() {
			viewContent = container.NewCenter(
				widget.NewLabel("Veuillez charger un fichier de configuration via le menu 'Art'hetic' -> 'Charger...'"),
			)
		} else {
			// Sinon, on affiche la vue normale
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
	buildAndUpdateView() // Premier rendu

	// Le w.ShowAndRun() sera fait dans main.go
}

func buildMainMenu(controller *UIController, parentWindow fyne.Window) *fyne.MainMenu {
		xlsxFilter := storage.NewExtensionFileFilter([]string{".xlsx"})

	fileMenu := fyne.NewMenu("Art'hetic",
		// NOUVEL ITEM DE MENU
		fyne.NewMenuItem("Charger configuration...", func() {
			// Ouvre une boîte de dialogue de sélection de fichier
			fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil || reader == nil {
					return // Erreur ou annulation
				}
				// On envoie le chemin (URI) au contrôleur pour qu'il gère le chargement.
				controller.LoadNewConfigFile(reader.URI().Path())
				reader.Close()
			}, parentWindow)
			
			fileDialog.SetFilter(xlsxFilter) // Appliquer le filtre
			fileDialog.Show()
		}),
		fyne.NewMenuItem("Quitter", func() {
			controller.QuitApp()
		}),
	)
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
			container.NewCenter(preview),
			widget.NewSeparator(),
			container.NewGridWithColumns(3, widget.NewLabel("Rouge"), widget.NewSliderWithData(0, 255, r), widget.NewLabelWithData(binding.FloatToStringWithFormat(r, "%.0f"))),
			container.NewGridWithColumns(3, widget.NewLabel("Vert"), widget.NewSliderWithData(0, 255, g), widget.NewLabelWithData(binding.FloatToStringWithFormat(g, "%.0f"))),
			container.NewGridWithColumns(3, widget.NewLabel("Bleu"), widget.NewSliderWithData(0, 255, b), widget.NewLabelWithData(binding.FloatToStringWithFormat(b, "%.0f"))),
			container.NewGridWithColumns(3, widget.NewLabel("Blanc"), widget.NewSliderWithData(0, 255, w), widget.NewLabelWithData(binding.FloatToStringWithFormat(w, "%.0f"))),
		)

		dialog.ShowCustomConfirm(
			"Choisir une couleur personnalisée", "Valider", "Annuler", content,
			func(ok bool) {
				if !ok {
					return
				}
				vr, _ := r.Get()
				vg, _ := g.Get()
				vb, _ := b.Get()
				vw, _ := w.Get()
				// Appel à la nouvelle fonction du contrôleur
				controller.RunFakerCustomColor(uint8(vr), uint8(vg), uint8(vb), uint8(vw))
			},
			parentWindow,
		)
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
		solidColorItem,
		animationsItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Retour au mode LIVE (eHub)", func() {
			controller.SwitchToLiveMode()
		}),
	)

	return fyne.NewMainMenu(fileMenu, fakerMenu)
}
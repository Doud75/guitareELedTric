// internal/ui/app.go
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"guitarHetic/internal/config"
	"guitarHetic/internal/simulator"
	"image/color"
)

func RunUI(
	cfg *config.Config,
	physicalConfigOut chan<- *config.Config,
	faker *simulator.Faker,
	monitorChanIn <-chan *UniverseMonitorData,
) {
	a := app.New()
	a.Settings().SetTheme(&ArtHeticTheme{})
	w := a.NewWindow("Guitare Hetic - Inspecteur ArtNet")

	state := NewUIState(cfg)
	controller := NewUIController(state, cfg, physicalConfigOut, a, faker, monitorChanIn)

	mainMenu := buildMainMenu(controller, w) // Passe la fenêtre w
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
		header := buildHeader(state, controller)
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

func buildMainMenu(controller *UIController, parentWindow fyne.Window) *fyne.MainMenu {
	fileMenu := fyne.NewMenu("Art'hetic",
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
// internal/ui/views.go
package ui

import (
    "fmt"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/canvas"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/layout"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
    "image/color"
    "strings"
)

// --- VUE DE MONITORING (AVEC SÉPARATEUR FIXE ET ESPACEMENT) ---
func buildUniverseView(state *UIState, ledCount int) fyne.CanvasObject {
    state.ledStateMutex.Lock()
    defer state.ledStateMutex.Unlock()

    ledWidgetSize := fyne.NewSize(14, 14)

    // --- Panneau de Gauche (Entrée eHub) ---
    inputLedObjects := make([]fyne.CanvasObject, ledCount)
    state.ledInputWidgets = make([]*LedWidget, 0, ledCount)
    for i := 0; i < ledCount; i++ {
        led := NewLedWidget()
        state.ledInputWidgets = append(state.ledInputWidgets, led)
        inputLedObjects[i] = led
    }
    inputGrid := container.New(layout.NewGridWrapLayout(ledWidgetSize), inputLedObjects...)
    inputScroll := container.NewScroll(inputGrid)
    inputColumn := container.NewBorder(
        widget.NewLabelWithStyle("Entrée eHub", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
        nil, nil, nil,
        inputScroll,
    )

    // --- Panneau de Droite (Sortie Art-Net) ---
    outputLedObjects := make([]fyne.CanvasObject, ledCount)
    state.ledOutputWidgets = make([]*LedWidget, 0, ledCount)
    for i := 0; i < ledCount; i++ {
        led := NewLedWidget()
        state.ledOutputWidgets = append(state.ledOutputWidgets, led)
        outputLedObjects[i] = led
    }
    outputGrid := container.New(layout.NewGridWrapLayout(ledWidgetSize), outputLedObjects...)
    outputScroll := container.NewScroll(outputGrid)
    outputColumn := container.NewBorder(
        widget.NewLabelWithStyle("Sortie Art-Net", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
        nil, nil, nil,
        outputScroll,
    )

    // --- Assemblage final ---
    leftSideWithSeparator := container.NewBorder(nil, nil, nil, widget.NewSeparator(), inputColumn)
    return container.NewPadded(
        container.New(layout.NewGridLayout(2),
            leftSideWithSeparator,
            outputColumn,
        ),
    )
}

// --- WIDGETS PERSONNALISÉS ET AUTRES VUES ---

type LedWidget struct {
    widget.BaseWidget
    Circle *canvas.Circle
}

func NewLedWidget() *LedWidget {
    w := &LedWidget{Circle: &canvas.Circle{StrokeWidth: 1, StrokeColor: color.Gray{Y: 60}}}
    w.ExtendBaseWidget(w)
    w.SetColor(color.Black)
    return w
}

func (w *LedWidget) MinSize() fyne.Size {
    return fyne.NewSize(14, 14)
}

func (w *LedWidget) SetColor(c color.Color) {
    w.Circle.FillColor = c
    w.Refresh()
}

func (w *LedWidget) CreateRenderer() fyne.WidgetRenderer {
    return widget.NewSimpleRenderer(w.Circle)
}

type SizedEntry struct {
    widget.Entry
    DesiredWidth float32
}

func NewSizedEntry(width float32) *SizedEntry {
    entry := &SizedEntry{DesiredWidth: width}
    entry.ExtendBaseWidget(entry)
    return entry
}

func (e *SizedEntry) MinSize() fyne.Size {
    originalMin := e.Entry.MinSize()
    return fyne.NewSize(e.DesiredWidth, originalMin.Height)
}

func buildHeader(state *UIState, controller *UIController) fyne.CanvasObject {
    title := widget.NewLabel("Inspecteur Art'Hetic")
    title.TextStyle.Bold = true
    var headerContent fyne.CanvasObject
    // La condition d'affichage du bouton reste la même.
    if state.CurrentView == DetailView || state.CurrentView == UniverseView {
        // Le bouton appelle maintenant la nouvelle fonction intelligente GoBack().
        backButton := widget.NewButtonWithIcon("Retour", theme.NavigateBackIcon(), func() {
            controller.GoBack()
        })
        headerContent = container.NewBorder(nil, nil, backButton, nil, title)
    } else {
        headerContent = title
    }
    return container.NewVBox(container.NewPadded(headerContent), widget.NewSeparator())
}

func buildIPListView(state *UIState, controller *UIController) fyne.CanvasObject {
    list := widget.NewList(
        func() int { return len(state.controllerIPs) },
        func() fyne.CanvasObject {
            return container.NewBorder(nil, nil, nil, widget.NewIcon(theme.NavigateNextIcon()), widget.NewLabel("Template IP"))
        },
        func(i widget.ListItemID, o fyne.CanvasObject) {
            o.(*fyne.Container).Objects[0].(*widget.Label).SetText(state.controllerIPs[i])
        },
    )
    list.OnSelected = func(id widget.ListItemID) {
        controller.SelectIPAndShowDetails(state.controllerIPs[id])
        list.UnselectAll()
    }
    return list
}

func buildDetailView(state *UIState, controller *UIController) fyne.CanvasObject {
    ipInput := NewSizedEntry(200.0)
    ipInput.SetText(state.selectedIP)
    validateButton := widget.NewButtonWithIcon("Sauvegarder", theme.ConfirmIcon(), func() { controller.ValidateNewIP(ipInput.Text) })
    validateButton.Importance = widget.HighImportance
    editLine := container.NewHBox(widget.NewLabel("Nouvelle IP :"), ipInput, validateButton, layout.NewSpacer())

    universeItems := []fyne.CanvasObject{
        widget.NewLabelWithStyle("Univers & Plages d'Entités", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
    }
    for _, detail := range state.selectedDetails {
        currentDetail := detail
        monitorButton := widget.NewButton(fmt.Sprintf("Monitorer l'Univers ArtNet %d", currentDetail.Universe), func() {
            controller.SelectUniverseAndShowDetails(currentDetail.Universe)
        })
        parts := make([]string, len(currentDetail.Ranges))
        for i, rg := range currentDetail.Ranges {
            parts[i] = fmt.Sprintf("%d à %d", rg[0], rg[1])
        }
        labelRanges := widget.NewLabel(strings.Join(parts, ", "))
        labelRanges.Alignment = fyne.TextAlignTrailing
        row := container.NewBorder(nil, nil, monitorButton, nil, labelRanges)
        universeItems = append(universeItems, row)
    }
    universeList := container.NewVBox(universeItems...)

    return container.NewScroll(container.NewVBox(container.NewPadded(editLine), widget.NewSeparator(), container.NewPadded(universeList)))
}

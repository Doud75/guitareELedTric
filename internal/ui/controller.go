package ui

import (
    "fmt"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/storage"
    "guitarHetic/internal/config"
    "guitarHetic/internal/simulator"
    "image/color"
    "log"
    "net"
    "sort"
)

type ConfigRequester func(request ConfigUpdateRequest)

type UIController struct {
    state           *UIState
    onStateChange   func()
    app             fyne.App
    faker           *simulator.Faker
    monitorIn       <-chan *UniverseMonitorData
    configRequester ConfigRequester
    isConfigLoaded  bool
}

func NewUIController(app fyne.App, faker *simulator.Faker, monitorIn <-chan *UniverseMonitorData, configRequester ConfigRequester) *UIController {
    c := &UIController{
        state:           NewUIState(nil),
        onStateChange:   func() {},
        app:             app,
        faker:           faker,
        monitorIn:       monitorIn,
        configRequester: configRequester,
        isConfigLoaded:  false,
    }
    go c.listenForMonitorUpdates()
    return c
}

func (c *UIController) LoadPatchFile(uri fyne.URI) {
    log.Printf("UI Controller: Demande de chargement du fichier de patch: %s", uri.Path())
    c.configRequester(ConfigUpdateRequest{PatchFilePath: uri.Path()})
}

func (c *UIController) SetPatchingActive(active bool) {
    log.Printf("UI Controller: Demande de changement d'état du patching à: %v", active)
    c.configRequester(ConfigUpdateRequest{SetPatchingActive: &active})
}

func (c *UIController) ClearPatch() {
    log.Printf("UI Controller: Demande de suppression du patch actuel.")
    c.configRequester(ConfigUpdateRequest{ClearPatch: true})
}

func (c *UIController) SetFaker(newFaker *simulator.Faker) {
    c.faker = newFaker
}

func (c *UIController) IsConfigLoaded() bool {
    return c.isConfigLoaded
}

func (c *UIController) UpdateWithNewConfig(cfg *config.Config) {
    fyne.Do(func() {
        if cfg == nil {
            c.isConfigLoaded = false
            c.state = NewUIState(nil)
        } else {
            newIPs, newCtrlMap := BuildModel(cfg)
            c.state.allControllers = newCtrlMap
            c.state.controllerIPs = newIPs
            c.state.viewStack = make([]ViewName, 0)
            c.state.CurrentView = IPListView
            c.isConfigLoaded = true
        }
        c.onStateChange()
    })
}

func (c *UIController) LoadNewConfigFile(uri fyne.URI) {
    log.Printf("UI Controller: Demande de chargement du fichier: %s", uri.Path())

    parent, err := storage.Parent(uri)
    if err == nil {
        if listableParent, ok := parent.(fyne.ListableURI); ok {
            c.state.lastOpenedFolder = listableParent
        }
    }

    c.configRequester(ConfigUpdateRequest{FilePath: uri.Path()})
}

func (c *UIController) ValidateNewIP(newIPStr string) {
    if net.ParseIP(newIPStr) == nil {
        log.Printf("UI ERROR: L'adresse IP '%s' est invalide. Abandon.", newIPStr)
        return
    }
    oldIP := c.state.selectedIP
    if oldIP == newIPStr {
        return
    }
    log.Printf("UI Controller: Demande de changement d'IP de '%s' vers '%s'", oldIP, newIPStr)
    ipChanges := make(map[string]string)
    ipChanges[oldIP] = newIPStr
    c.configRequester(ConfigUpdateRequest{IPChanges: ipChanges})
}

func (c *UIController) ValidateNewIPForUniverse(universeID int, newIPStr string) {
    if net.ParseIP(newIPStr) == nil {
        log.Printf("UI ERROR: L'adresse IP '%s' pour l'univers %d est invalide. Abandon.", newIPStr, universeID)
        return
    }

    log.Printf("UI Controller: Demande de changement d'IP pour l'univers %d vers '%s'", universeID, newIPStr)

    ipChanges := make(map[string]string)

    universeKey := fmt.Sprintf("universe:%d", universeID)
    ipChanges[universeKey] = newIPStr

    c.configRequester(ConfigUpdateRequest{IPChanges: ipChanges})
}

func (c *UIController) SaveConfigFile(path string) {
    log.Printf("UI Controller: Demande de sauvegarde vers: %s", path)
    c.configRequester(ConfigUpdateRequest{ExportPath: path})
}

func (c *UIController) SetUpdateCallback(callback func()) {
    c.onStateChange = callback
}

func (c *UIController) navigateTo(view ViewName) {
    c.state.viewStack = append(c.state.viewStack, c.state.CurrentView)
    c.state.CurrentView = view
    c.onStateChange()
}

func (c *UIController) GoBack() {
    stackLen := len(c.state.viewStack)
    if stackLen == 0 {
        return
    }
    previousView := c.state.viewStack[stackLen-1]
    c.state.viewStack = c.state.viewStack[:stackLen-1]
    if c.state.CurrentView == UniverseView {
        c.state.ledStateMutex.Lock()
        c.state.universeViewContent = nil
        c.state.ledInputWidgets = nil
        c.state.ledOutputWidgets = nil
        c.state.ledStateMutex.Unlock()
    }
    c.state.CurrentView = previousView
    c.onStateChange()
}

func (c *UIController) SelectUniverseAndShowDetails(universeID int) {
    entityCount := 0
    for _, ipData := range c.state.allControllers {
        if ranges, ok := ipData[universeID]; ok {
            for _, r := range ranges {
                entityCount += (r[1] - r[0] + 1)
            }
        }
    }

    if entityCount == 0 {
        log.Printf("UI CONTROLLER: Aucune entité trouvée pour l'univers %d. Affichage annulé.", universeID)
        return
    }
    log.Printf("UI CONTROLLER: Construction de la vue pour l'univers %d avec %d entités.", universeID, entityCount)
    c.state.selectedUniverse = universeID
    c.state.universeViewContent = buildUniverseView(c.state, entityCount)
    c.navigateTo(UniverseView)
}

func (c *UIController) SelectIPAndShowDetails(ip string) {
    c.state.selectedIP = ip
    entries := c.state.allControllers[ip]
    details := make([]UniRange, 0, len(entries))
    for u, ranges := range entries {
        details = append(details, UniRange{Universe: u, Ranges: ranges})
    }
    sort.Slice(details, func(i, j int) bool { return details[i].Universe < details[j].Universe })
    c.state.selectedDetails = details
    c.navigateTo(DetailView)
}

func (c *UIController) QuitApp() {
    if c.faker != nil {
        c.faker.Stop()
    }
    c.app.Quit()
}

func (c *UIController) RunFakerCommand(command string) {
    if c.faker != nil {
        go c.faker.SendTestPattern(command)
    }
}

func (c *UIController) RunFakerCustomColor(r, g, b, w byte) {
    if c.faker != nil {
        go c.faker.SendTestPattern("custom", r, g, b, w)
    }
}

func (c *UIController) SwitchToLiveMode() {
    if c.faker != nil {
        go c.faker.SwitchToLiveMode()
    }
}

func (c *UIController) listenForMonitorUpdates() {
    for data := range c.monitorIn {
        if c.state.CurrentView != UniverseView || c.state.selectedUniverse != data.UniverseID || c.state.universeViewContent == nil {
            continue
        }
        inputColors := make([]color.Color, len(c.state.ledInputWidgets))
        for i := range inputColors {
            if i < len(data.InputState) {
                entity := data.InputState[i]
                inputColors[i] = amplifyColor(entity.Red, entity.Green, entity.Blue)
            } else {
                inputColors[i] = color.Black
            }
        }
        outputColors := make([]color.Color, len(c.state.ledOutputWidgets))
        for i := range outputColors {
            offset := i * 3
            if offset+2 < len(data.OutputDMX) {
                r, g, b := data.OutputDMX[offset], data.OutputDMX[offset+1], data.OutputDMX[offset+2]
                outputColors[i] = amplifyColor(r, g, b)
            } else {
                outputColors[i] = color.Black
            }
        }
        fyne.Do(func() {
            c.state.ledStateMutex.RLock()
            defer c.state.ledStateMutex.RUnlock()
            if c.state.ledInputWidgets == nil || c.state.ledOutputWidgets == nil {
                return
            }
            for i, widget := range c.state.ledInputWidgets {
                widget.SetColor(inputColors[i])
            }
            for i, widget := range c.state.ledOutputWidgets {
                widget.SetColor(outputColors[i])
            }
        })
    }
}

func amplifyColor(r, g, b byte) color.Color {
    const minBrightness = 100
    if r == 0 && g == 0 && b == 0 {
        return color.Black
    }
    if r > 0 && r < minBrightness {
        r = minBrightness
    }
    if g > 0 && g < minBrightness {
        g = minBrightness
    }
    if b > 0 && b < minBrightness {
        b = minBrightness
    }
    return color.NRGBA{R: r, G: g, B: b, A: 255}
}

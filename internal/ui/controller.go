// internal/ui/controller.go
package ui

import (
    "fyne.io/fyne/v2"
    "guitarHetic/internal/config"
    "guitarHetic/internal/simulator"
    "image/color"
    "log"
    "net"
    "sort"
)

// UIController est le cerveau qui manipule l'état en réponse aux actions.
type UIController struct {
    state             *UIState
    onStateChange     func()
    currentConfig     *config.Config
    physicalConfigOut chan<- *config.Config
    app               fyne.App
    faker             *simulator.Faker
    monitorIn         <-chan *UniverseMonitorData
}

// NewUIController construit et initialise le contrôleur.
func NewUIController(state *UIState, initialConfig *config.Config, cfgOut chan<- *config.Config, app fyne.App, faker *simulator.Faker, monitorIn <-chan *UniverseMonitorData) *UIController {
    c := &UIController{
        state:             state,
        onStateChange:     func() {},
        currentConfig:     initialConfig,
        physicalConfigOut: cfgOut,
        app:               app,
        faker:             faker,
        monitorIn:         monitorIn,
    }
    go c.listenForMonitorUpdates()
    return c
}

// amplifyColor est une petite fonction d'aide pour rendre les couleurs plus visibles.
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

func (c *UIController) SetUpdateCallback(callback func()) {
    c.onStateChange = callback
}

// --- NOUVELLE LOGIQUE DE NAVIGATION ---

// navigateTo est la nouvelle fonction centralisée pour changer de vue.
func (c *UIController) navigateTo(view ViewName) {
    // On pousse la vue actuelle sur la pile d'historique.
    c.state.viewStack = append(c.state.viewStack, c.state.CurrentView)
    // On met à jour la vue actuelle.
    c.state.CurrentView = view
    // On rafraîchit l'interface.
    c.onStateChange()
}

// GoBack est la nouvelle fonction "intelligente" pour le bouton retour.
func (c *UIController) GoBack() {
    stackLen := len(c.state.viewStack)
    if stackLen == 0 {
        return // Sécurité: ne rien faire si la pile est vide.
    }

    // 1. On récupère la vue précédente depuis la pile.
    previousView := c.state.viewStack[stackLen-1]

    // 2. On retire la vue de la pile (on la "pop").
    c.state.viewStack = c.state.viewStack[:stackLen-1]

    // 3. On nettoie l'état de la vue que l'on quitte, si nécessaire.
    if c.state.CurrentView == UniverseView {
        c.state.ledStateMutex.Lock()
        c.state.universeViewContent = nil
        c.state.ledInputWidgets = nil
        c.state.ledOutputWidgets = nil
        c.state.ledStateMutex.Unlock()
    }

    // 4. On définit la nouvelle vue actuelle.
    c.state.CurrentView = previousView

    // 5. On rafraîchit l'interface.
    c.onStateChange()
}

// Les fonctions de navigation utilisent maintenant navigateTo.
func (c *UIController) SelectUniverseAndShowDetails(universeID int) {
    entityCount := 0
    for _, route := range c.currentConfig.RoutingTable {
        if route.Universe == universeID {
            entityCount++
        }
    }
    if entityCount == 0 {
        log.Printf("UI CONTROLLER: Aucune entité trouvée pour l'univers %d. Affichage annulé.", universeID)
        return
    }

    log.Printf("UI CONTROLLER: Construction de la vue pour l'univers %d avec %d entités.", universeID, entityCount)
    c.state.selectedUniverse = universeID
    c.state.universeViewContent = buildUniverseView(c.state, entityCount)

    c.navigateTo(UniverseView) // Utilise la nouvelle fonction
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

    c.navigateTo(DetailView) // Utilise la nouvelle fonction
}

// --- Fonctions restantes du contrôleur ---

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

func (c *UIController) ValidateNewIP(newIP string) {
    if net.ParseIP(newIP) == nil {
        log.Printf("UI ERROR: L'adresse IP '%s' est invalide. Abandon.", newIP)
        return
    }
    oldIP := c.state.selectedIP
    newConfig := deepCopyConfig(c.currentConfig)
    for i, entry := range newConfig.RoutingTable {
        if entry.IP == oldIP {
            newConfig.RoutingTable[i].IP = newIP
        }
    }
    for i, ip := range newConfig.UniverseIP {
        if ip == oldIP {
            newConfig.UniverseIP[i] = newIP
        }
    }
    c.physicalConfigOut <- newConfig
    c.currentConfig = newConfig
    newIPs, newCtrlMap := BuildModel(newConfig)
    c.state.allControllers = newCtrlMap
    c.state.controllerIPs = newIPs

    // Au lieu d'un retour direct, on vide la pile d'historique et on retourne à la liste
    c.state.viewStack = make([]ViewName, 0)
    c.state.CurrentView = IPListView
    c.onStateChange()
}

func deepCopyConfig(original *config.Config) *config.Config {
    if original == nil {
        return nil
    }
    newRoutingTable := make([]config.RoutingEntry, len(original.RoutingTable))
    copy(newRoutingTable, original.RoutingTable)
    newUniverseIP := make(map[int]string)
    for k, v := range original.UniverseIP {
        newUniverseIP[k] = v
    }
    return &config.Config{RoutingTable: newRoutingTable, UniverseIP: newUniverseIP}
}

// SwitchToLiveMode demande au faker de se désactiver et de repasser l'aiguilleur
// en mode eHub.
func (c *UIController) SwitchToLiveMode() {
    if c.faker != nil {
        // La logique est déjà dans le Faker, on n'a qu'à l'appeler.
        // On le fait dans une goroutine pour ne jamais bloquer l'UI, par principe.
        go c.faker.SwitchToLiveMode()
    }
}
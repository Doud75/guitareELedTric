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
    app               fyne.App // Conservé pour la fonction QuitApp
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
    // On lance l'écouteur en arrière-plan.
    go c.listenForMonitorUpdates()
    return c
}

// listenForMonitorUpdates est la goroutine qui met à jour les widgets et demande un rafraîchissement de l'UI.
// CETTE FONCTION EST LA CLÉ DE LA CORRECTION.
func (c *UIController) listenForMonitorUpdates() {
    for data := range c.monitorIn {
        // On s'assure que la vue est active et que les widgets ont bien été créés.
        if c.state.CurrentView != UniverseView || c.state.selectedUniverse != data.UniverseID || c.state.universeViewContent == nil {
            continue
        }

        // Préparation des données (on peut le faire en dehors du fyne.Do)
        inputColors := make([]color.Color, len(c.state.ledInputWidgets))
        for i := range inputColors {
            if i < len(data.InputState) {
                entity := data.InputState[i]
                inputColors[i] = color.NRGBA{R: entity.Red, G: entity.Green, B: entity.Blue, A: 255}
            } else {
                inputColors[i] = color.Black
            }
        }

        outputColors := make([]color.Color, len(c.state.ledOutputWidgets))
        for i := range outputColors {
            offset := i * 3
            if offset+2 < len(data.OutputDMX) {
                r, g, b := data.OutputDMX[offset], data.OutputDMX[offset+1], data.OutputDMX[offset+2]
                outputColors[i] = color.NRGBA{R: r, G: g, B: b, A: 255}
            } else {
                outputColors[i] = color.Black
            }
        }

        // CORRECTION: On exécute TOUTES les modifications de l'UI dans fyne.Do
        fyne.Do(func() {
            c.state.ledStateMutex.RLock()
            defer c.state.ledStateMutex.RUnlock()

            // On vérifie que les widgets n'ont pas été détruits entre-temps
            if c.state.ledInputWidgets == nil || c.state.ledOutputWidgets == nil {
                return
            }

            log.Printf("UI FYNE.DO: Mise à jour des couleurs pour l'univers %d", data.UniverseID)

            // Mise à jour des widgets d'entrée (eHub)
            for i, widget := range c.state.ledInputWidgets {
                widget.SetColor(inputColors[i])
            }

            // Mise à jour des widgets de sortie (Art-Net)
            for i, widget := range c.state.ledOutputWidgets {
                widget.SetColor(outputColors[i])
            }
        })
    }
}

// SetUpdateCallback permet à l'UI de s'enregistrer pour les changements de vue.
func (c *UIController) SetUpdateCallback(callback func()) {
    c.onStateChange = callback
}

// SelectUniverseAndShowDetails prépare le changement vers la vue de monitoring.
func (c *UIController) SelectUniverseAndShowDetails(universeID int) {
    c.state.selectedUniverse = universeID

    // CHANGEMENT: On construit la vue ici, UNE SEULE FOIS.
    // La fonction buildUniverseView va peupler l'état avec les widgets créés.
    c.state.universeViewContent = buildUniverseView(c.state)

    c.state.CurrentView = UniverseView
    // On déclenche le rafraîchissement global pour afficher la nouvelle vue.
    c.onStateChange()
}

// SelectIPAndShowDetails est appelée lorsqu'un utilisateur clique sur une IP.
func (c *UIController) SelectIPAndShowDetails(ip string) {
    c.state.selectedIP = ip
    entries := c.state.allControllers[ip]
    details := make([]UniRange, 0, len(entries))
    for u, ranges := range entries {
        details = append(details, UniRange{Universe: u, Ranges: ranges})
    }
    sort.Slice(details, func(i, j int) bool { return details[i].Universe < details[j].Universe })
    c.state.selectedDetails = details
    c.state.CurrentView = DetailView
    c.onStateChange()
}

// GoBackToIPList est appelée par le bouton "Retour".
func (c *UIController) GoBackToIPList() {
    // CHANGEMENT: On nettoie l'état pour libérer la mémoire des widgets.
    c.state.ledStateMutex.Lock()
    c.state.ledInputWidgets = nil
    c.state.ledOutputWidgets = nil
    c.state.universeViewContent = nil
    c.state.ledStateMutex.Unlock()

    c.state.CurrentView = IPListView
    c.state.selectedIP = ""
    c.state.selectedDetails = nil
    c.onStateChange()
}

// --- AUTRES FONCTIONS DU CONTRÔLEUR (inchangées) ---

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
    c.GoBackToIPList()
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
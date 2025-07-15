package ui

import (
    "fyne.io/fyne/v2"
    "guitarHetic/internal/config"
    "guitarHetic/internal/simulator"
    "log"
    "net"
    "sort"
)

// UIController est le cerveau qui manipule l'état en réponse aux actions.
type UIController struct {
    state             *UIState
    onStateChange     func()
    currentConfig     *config.Config        // Garde une référence à la config actuelle pour la copier
    physicalConfigOut chan<- *config.Config // Le canal pour envoyer la config mise à jour
    app               fyne.App
    faker             *simulator.Faker
}

func NewUIController(state *UIState, initialConfig *config.Config, cfgOut chan<- *config.Config, app fyne.App, faker *simulator.Faker) *UIController {
    return &UIController{
        state:             state,
        onStateChange:     func() {},
        currentConfig:     initialConfig,
        physicalConfigOut: cfgOut,
        app:               app,
        faker:             faker,
    }
}

// SetUpdateCallback permet à la vue de s'enregistrer pour les mises à jour.
func (c *UIController) SetUpdateCallback(callback func()) {
    c.onStateChange = callback
}

// SelectIPAndShowDetails est appelée lorsqu'un utilisateur clique sur une IP.
func (c *UIController) SelectIPAndShowDetails(ip string) {
    // 1. Mettre à jour l'état avec les informations de la sélection.
    c.state.selectedIP = ip

    // Calculer les détails pour la vue suivante.
    entries := c.state.allControllers[ip]
    details := make([]UniRange, 0, len(entries))
    for u, ranges := range entries {
        details = append(details, UniRange{Universe: u, Ranges: ranges})
    }
    sort.Slice(details, func(i, j int) bool {
        return details[i].Universe < details[j].Universe
    })
    c.state.selectedDetails = details

    // 2. Changer la vue actuelle pour la vue de détails.
    c.state.CurrentView = DetailView

    // 3. Notifier l'interface qu'elle doit se redessiner complètement.
    c.onStateChange()
}

// GoBackToIPList est appelée par le bouton "Retour".
func (c *UIController) GoBackToIPList() {
    // 1. Changer la vue pour revenir à la liste.
    c.state.CurrentView = IPListView

    // (Optionnel mais propre) Réinitialiser la sélection.
    c.state.selectedIP = ""
    c.state.selectedDetails = nil

    // 2. Notifier l'interface de se redessiner.
    c.onStateChange()
}

// --- ACTIONS DU MENU ---

// QuitApp gère l'action de quitter proprement.
func (c *UIController) QuitApp() {
    if c.faker != nil {
        c.faker.Stop() // Arrêter les animations du faker avant de quitter.
    }
    c.app.Quit()
}

// RunFakerCommand exécute une commande de test via le Faker.
func (c *UIController) RunFakerCommand(command string) {
    if c.faker != nil {
        // Lancer dans une goroutine pour ne pas bloquer l'UI.
        go c.faker.SendTestPattern(command)
    }
}

func deepCopyConfig(original *config.Config) *config.Config {
    if original == nil {
        return nil
    }

    // Copier la RoutingTable
    newRoutingTable := make([]config.RoutingEntry, len(original.RoutingTable))
    copy(newRoutingTable, original.RoutingTable)

    // Copier la map UniverseIP
    newUniverseIP := make(map[int]string)
    for k, v := range original.UniverseIP {
        newUniverseIP[k] = v
    }

    return &config.Config{
        RoutingTable: newRoutingTable,
        UniverseIP:   newUniverseIP,
    }
}

func (c *UIController) ValidateNewIP(newIP string) {
    // 1. Valider l'entrée
    if net.ParseIP(newIP) == nil {
        log.Printf("UI ERROR: L'adresse IP '%s' est invalide. Abandon.", newIP)
        // Ici, on pourrait mettre à jour l'état pour afficher une erreur à l'utilisateur.
        return
    }

    oldIP := c.state.selectedIP
    log.Printf("UI ACTION: Validation de la nouvelle IP '%s' pour l'ancienne '%s'", newIP, oldIP)

    // 2. Créer une copie profonde de la configuration actuelle
    newConfig := deepCopyConfig(c.currentConfig)

    // 3. Modifier la copie
    for i, entry := range newConfig.RoutingTable {
        if entry.IP == oldIP {
            newConfig.RoutingTable[i].IP = newIP
        }
    }
    // On reconstruit la map UniverseIP pour être certain de sa cohérence
    for i, ip := range newConfig.UniverseIP {
        if ip == oldIP {
            newConfig.UniverseIP[i] = newIP
        }
    }

    // 4. Envoyer la configuration mise à jour au processeur
    log.Println("UI ACTION: Envoi de la configuration mise à jour au processeur...")
    c.physicalConfigOut <- newConfig

    // 5. Mettre à jour l'état interne de l'UI
    // Le contrôleur considère maintenant la nouvelle config comme la version "actuelle".
    c.currentConfig = newConfig
    // On met à jour l'état qui pilote les vues.
    newIPs, newCtrlMap := BuildModel(newConfig)
    c.state.allControllers = newCtrlMap
    c.state.controllerIPs = newIPs

    log.Println("UI ACTION: Changement appliqué. Retour à la liste.")
    // 6. Naviguer en arrière
    c.GoBackToIPList() // Cette méthode déclenche déjà onStateChange, donc la vue sera rafraîchie.
}

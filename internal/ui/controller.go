package ui

import (
	"fyne.io/fyne/v2"
	"guitarHetic/internal/config"
	"guitarHetic/internal/simulator"
	"image/color"
	"net"
	"sort"
)

type UIController struct {
	state             *UIState
	onStateChange     func()
	currentConfig     *config.Config
	physicalConfigOut chan<- *config.Config
	app               fyne.App
	faker             *simulator.Faker
	monitorIn         <-chan *UniverseMonitorData
}

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
	for _, route := range c.currentConfig.RoutingTable {
		if route.Universe == universeID {
			entityCount++
		}
	}
	if entityCount == 0 {
		return
	}
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
		c.faker.Stop() // Appelle la nouvelle méthode Stop du Faker
	}
	c.app.Quit()
}

// Pour les commandes simples comme "red", "animation", etc.
func (c *UIController) RunFakerCommand(command string) {
	if c.faker != nil {
		go c.faker.SendTestPattern(command)
	}
}

// Pour la couleur personnalisée venant de la boîte de dialogue
func (c *UIController) RunFakerCustomColor(r, g, b, w byte) {
	if c.faker != nil {
		go c.faker.SendTestPattern("custom", r, g, b, w)
	}
}

func (c *UIController) ValidateNewIP(newIP string) {
	if net.ParseIP(newIP) == nil {
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
	c.state.viewStack = make([]ViewName, 0)
	c.state.CurrentView = IPListView
	c.onStateChange()
}

func (c *UIController) SwitchToLiveMode() {
	if c.faker != nil {
		go c.faker.SwitchToLiveMode()
	}
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
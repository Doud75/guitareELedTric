package ui

import (
    "guitarHetic/internal/domain/ehub"
    "image/color"
)

type UniverseMonitorData struct {
    UniverseID int
    InputState []ehub.EHubEntityState
    OutputDMX  [512]byte
}

type LedState struct {
    InputColors  []color.Color
    OutputColors []color.Color
}

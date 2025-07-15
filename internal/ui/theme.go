package ui

import (
    "image/color"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/theme"
)

// myTheme hérite du thème par défaut mais nous permet de surcharger des éléments.
type myTheme struct{}

// S'assurer que notre thème implémente bien l'interface fyne.Theme
var _ fyne.Theme = (*myTheme)(nil)

// On retourne le thème par défaut pour toutes les méthodes que nous ne surchargeons pas.
func (m *myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
    return theme.DefaultTheme().Color(name, variant)
}

func (m *myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
    return theme.DefaultTheme().Icon(name)
}

func (m *myTheme) Font(style fyne.TextStyle) fyne.Resource {
    return theme.DefaultTheme().Font(style)
}

// C'est ici que l'on peut surcharger les valeurs.
// Augmentons un peu l'espacement par défaut pour que l'UI respire mieux.
func (m *myTheme) Size(name fyne.ThemeSizeName) float32 {
    if name == theme.SizeNamePadding {
        return 8 // Le padding par défaut est 4, on le double.
    }
    return theme.DefaultTheme().Size(name)
}

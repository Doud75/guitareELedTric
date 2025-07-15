package ui

import (
    "image/color"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/theme"
)

// La palette de couleurs reste la même, elle est bien définie.
var (
    colorBackground = color.NRGBA{R: 0x1E, G: 0x2A, B: 0x3A, A: 0xFF} // Bleu très sombre
    colorInputBg    = color.NRGBA{R: 0x2C, G: 0x3E, B: 0x50, A: 0xFF} // Bleu-gris pour les fonds d'input/cartes
    colorPrimary    = color.NRGBA{R: 0x4A, G: 0x90, B: 0xE2, A: 0xFF} // Bleu "classe" pour les accents
    colorForeground = color.NRGBA{R: 0xE0, G: 0xE0, B: 0xE0, A: 0xFF} // Texte blanc cassé
    colorMuted      = color.NRGBA{R: 0x9E, G: 0x9E, B: 0x9E, A: 0xFF} // Texte plus discret
)

// ArtHeticTheme est notre thème personnalisé.
type ArtHeticTheme struct{}

var _ fyne.Theme = (*ArtHeticTheme)(nil)

// Color est la fonction corrigée avec les bonnes constantes.
func (t *ArtHeticTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
    switch name {
    // Fond principal de l'application
    case theme.ColorNameBackground:
        return colorBackground

    // Couleur pour le texte principal
    case theme.ColorNameForeground:
        return colorForeground

    // Couleur d'accentuation (boutons "HighImportance", focus, sélection)
    case theme.ColorNamePrimary:
        return colorPrimary

    // Couleur pour le texte désactivé et les placeholders
    case theme.ColorNameDisabled:
        return colorMuted

    // Couleur du fond des widgets de saisie (Entry, Select, etc.)
    // C'est cette couleur qui va donner l'apparence des "cartes" pour les inputs.
    case theme.ColorNameInputBackground:
        return colorInputBg

    // Couleur du focus autour des widgets
    case theme.ColorNameFocus:
        c := colorPrimary
        c.A = 0x99 // On la rend un peu plus visible
        return c

    // Couleur des séparateurs
    case theme.ColorNameSeparator:
        c := colorMuted
        c.A = 0x44 // Un peu plus visible pour bien délimiter
        return c

    // Couleur de l'ombre (subtil mais important pour l'effet de profondeur)
    case theme.ColorNameShadow:
        return color.NRGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x66}

    default:
        // La sécurité reste la même : on se rabat sur le thème sombre par défaut.
        return theme.DarkTheme().Color(name, variant)
    }
}

// Le reste du fichier est correct et n'a pas besoin de changement.
func (t *ArtHeticTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
    return theme.DarkTheme().Icon(name)
}

func (t *ArtHeticTheme) Font(style fyne.TextStyle) fyne.Resource {
    return theme.DarkTheme().Font(style)
}

func (t *ArtHeticTheme) Size(name fyne.ThemeSizeName) float32 {
    if name == theme.SizeNamePadding {
        return 8.0
    }
    return theme.DarkTheme().Size(name)
}

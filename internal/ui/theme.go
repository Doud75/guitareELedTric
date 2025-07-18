package ui

import (
    "image/color"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/theme"
)

var (
    colorBackground = color.NRGBA{R: 0x1E, G: 0x2A, B: 0x3A, A: 0xFF}
    colorInputBg    = color.NRGBA{R: 0x2C, G: 0x3E, B: 0x50, A: 0xFF}
    colorPrimary    = color.NRGBA{R: 0x4A, G: 0x90, B: 0xE2, A: 0xFF}
    colorForeground = color.NRGBA{R: 0xE0, G: 0xE0, B: 0xE0, A: 0xFF}
    colorMuted      = color.NRGBA{R: 0x9E, G: 0x9E, B: 0x9E, A: 0xFF}
)

type ArtHeticTheme struct{}

var _ fyne.Theme = (*ArtHeticTheme)(nil)

func (t *ArtHeticTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
    switch name {
    case theme.ColorNameBackground:
        return colorBackground

    case theme.ColorNameForeground:
        return colorForeground

    case theme.ColorNamePrimary:
        return colorPrimary

    case theme.ColorNameDisabled:
        return colorMuted

    case theme.ColorNameInputBackground:
        return colorInputBg

    case theme.ColorNameFocus:
        c := colorPrimary
        c.A = 0x99
        return c

    case theme.ColorNameSeparator:
        c := colorMuted
        c.A = 0x44
        return c

    case theme.ColorNameShadow:
        return color.NRGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x66}

    default:
        return theme.DarkTheme().Color(name, variant)
    }
}

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

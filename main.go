package main

import (
    "embed"
    "log"

    "github.com/wailsapp/wails/v2"
    "github.com/wailsapp/wails/v2/pkg/options"
    "github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

var assets embed.FS

func main() {
    app := NewApp()

    err := wails.Run(&options.App{
        Title:  "GuitareELedTric Controller",
        Width:  1280,
        Height: 800,
        AssetServer: &assetserver.Options{
            Assets: assets,
        },
        BackgroundColour: &options.RGBA{R: 30, G: 30, B: 40, A: 255},
        OnStartup:        app.onStartup,
        OnShutdown:       app.onShutdown,
        Bind: []any{
            app,
        },
    })

    if err != nil {
        log.Fatal(err)
    }
}

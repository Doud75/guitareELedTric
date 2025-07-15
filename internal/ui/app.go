package ui

import (
    "context"
    "fyne.io/fyne/v2"
    "sort"

    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
    "guitarHetic/internal/config"
)

func RunUI(ctx context.Context, cfg *config.Config) {
    a := app.New()
    w := a.NewWindow("Controllers")

    ips, ctrlMap := BuildModel(cfg)
    table, detailData := NewDetailTable()
    list := NewControllerList(ips, func(ip string) {
        entries := ctrlMap[ip]
        details := make([]UniRange, 0, len(entries))
        for u, ranges := range entries {
            details = append(details, UniRange{Universe: u, Ranges: ranges})
        }
        sort.Slice(details, func(i, j int) bool {
            return details[i].Universe < details[j].Universe
        })
        *detailData = details
        table.Refresh()
    })

    reloadBtn := widget.NewButton("Recharger", func() {
        ips, ctrlMap = BuildModel(cfg)
        list.Refresh()
        *detailData = (*detailData)[:0]
        table.Refresh()
    })

    content := container.NewBorder(
        reloadBtn, nil, nil, nil,
        container.NewHSplit(
            container.NewScroll(list),
            container.NewScroll(table),
        ),
    )

    w.SetContent(content)
    w.Resize(fyne.NewSize(800, 600))

    go func() {
        <-ctx.Done()
        a.Quit()
    }()

    w.ShowAndRun()
}

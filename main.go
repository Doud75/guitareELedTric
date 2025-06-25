package main

import (
    "fmt"
    "guitarHetic/internal/artnet"
    "guitarHetic/internal/config"
    "guitarHetic/internal/ehub"
    "time"
)

func main() {
    fmt.Println("LED Controller App démarrée")

    conf := config.Load()
    artnet.Init(conf.UniverseToMapping)
    artnet.Start()

    ehub.DrawRedSquareCenter(artnet.Instance())

    time.Sleep(2 * time.Second)
}

package main

import (
    "flag"
    "log"
    "sync"
    "okaroute/internal/client"
    "okaroute/internal/config"
    "okaroute/internal/porthop"
)

func main() {
    cfgPath := flag.String("config", "configs/client.json", "path to client config")
    flag.Parse()
    cfgs, err := config.LoadClientConfigs(*cfgPath)
    if err != nil { log.Fatal(err) }
    var wg sync.WaitGroup
    for _, cfg := range cfgs {
        sec, err := porthop.DecodeSecret(cfg.TOTPSecret)
        if err != nil { log.Fatal(err) }
        cl := client.New(cfg, sec)
        wg.Add(1)
        go func(c *client.Client) {
            defer wg.Done()
            if err := c.Start(); err != nil { log.Println(err) }
        }(cl)
    }
    wg.Wait()
}
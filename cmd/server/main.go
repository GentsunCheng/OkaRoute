package main

import (
    "context"
    "flag"
    "log"
    "sync"
    "time"
    "okaroute/internal/config"
    "okaroute/internal/porthop"
    "okaroute/internal/server"
)

func main() {
    cfgPath := flag.String("config", "configs/server.json", "path to server config")
    flag.Parse()
    cfgs, err := config.LoadServerConfigs(*cfgPath)
    if err != nil { log.Fatal(err) }
    var wg sync.WaitGroup
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    for _, cfg := range cfgs {
        sec, err := porthop.DecodeSecret(cfg.TOTPSecret)
        if err != nil { log.Fatal(err) }
        srv := server.New(cfg, sec)
        wg.Add(1)
        go func(s *server.Server) {
            defer wg.Done()
            if err := s.Start(ctx); err != nil { log.Println(err) }
        }(srv)
    }
    time.Sleep(time.Hour)
}
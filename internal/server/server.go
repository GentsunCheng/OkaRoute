package server

import (
    "context"
    "encoding/binary"
    "io"
    "log"
    "net"
    "strconv"
    "sync"
    "time"
    "okaroute/internal/auth"
    "okaroute/internal/config"
    "okaroute/internal/forward"
    "okaroute/internal/porthop"
)

type Server struct {
    cfg config.ServerConfig
    secret []byte
    target string
    mu sync.Mutex
    listeners map[int]net.Listener
    currentStep int64
    name string
}

func New(cfg config.ServerConfig, secret []byte) *Server {
    return &Server{cfg: cfg, secret: secret, target: net.JoinHostPort(cfg.TargetAddr, itoa(cfg.TargetPort)), listeners: map[int]net.Listener{}, name: cfg.Name}
}

func itoa(i int) string { return fmtInt(i) }

func fmtInt(i int) string { return strconv.FormatInt(int64(i), 10) }

func (s *Server) openPort(port int) error {
    if _, ok := s.listeners[port]; ok {
        return nil
    }
    l, err := net.Listen("tcp", net.JoinHostPort(s.cfg.ListenIP, fmtInt(port)))
    if err != nil {
        return err
    }
    s.listeners[port] = l
    if s.name != "" { log.Printf("[%s] 服务端开始监听端口: %d", s.name, port) } else { log.Printf("服务端开始监听端口: %d", port) }
    go s.acceptLoop(port, l)
    return nil
}

func (s *Server) closePort(port int) {
    if l, ok := s.listeners[port]; ok {
        l.Close()
        delete(s.listeners, port)
    }
}

func (s *Server) acceptLoop(port int, l net.Listener) {
    for {
        c, err := l.Accept()
        if err != nil {
            return
        }
        go s.handleConnOnPort(port, c)
    }
}

func (s *Server) handleConnOnPort(port int, c net.Conn) {
    var hdr [8 + 16 + 32]byte
    if _, err := ioReadFull(c, hdr[:]); err != nil {
        c.Close()
        return
    }
    step := int64(binary.BigEndian.Uint64(hdr[0:8]))
    nonce := hdr[8:24]
    token := hdr[24:56]
    nowStep := porthop.StepIndex(time.Now(), s.cfg.StepSeconds)
    if !porthop.ClampSkew(step, nowStep, s.cfg.SkewSteps) {
        if s.name != "" { log.Printf("[%s] 服务端握手失败: 步长超出容忍, 来自=%s 使用端口=%d 声明step=%d 当前step=%d", s.name, c.RemoteAddr().String(), port, step, nowStep) } else { log.Printf("服务端握手失败: 步长超出容忍, 来自=%s 使用端口=%d 声明step=%d 当前step=%d", c.RemoteAddr().String(), port, step, nowStep) }
        c.Close()
        return
    }
    if !auth.Verify(s.secret, step, nonce, token, "client") {
        if s.name != "" { log.Printf("[%s] 服务端握手失败: 鉴权无效, 来自=%s 使用端口=%d step=%d", s.name, c.RemoteAddr().String(), port, step) } else { log.Printf("服务端握手失败: 鉴权无效, 来自=%s 使用端口=%d step=%d", c.RemoteAddr().String(), port, step) }
        c.Close()
        return
    }
    if s.name != "" { log.Printf("[%s] 服务端接受连接: 来自=%s 转发端口=%d step=%d 目标=%s", s.name, c.RemoteAddr().String(), port, step, s.target) } else { log.Printf("服务端接受连接: 来自=%s 转发端口=%d step=%d 目标=%s", c.RemoteAddr().String(), port, step, s.target) }
    forward.HandleTCP(c, s.target)
}

func ioReadFull(c net.Conn, b []byte) (int, error) { return io.ReadFull(c, b) }

func (s *Server) Start(ctx context.Context) error {
    s.currentStep = porthop.StepIndex(time.Now(), s.cfg.StepSeconds)
    prev, curr, next := porthop.Triplet(s.secret, s.currentStep, s.cfg.PortRange.Min, s.cfg.PortRange.Max)
    ports := porthop.UniquePorts([]int{prev, curr, next})
    for _, p := range ports {
        if err := s.openPort(p); err != nil { return err }
    }
    if s.name != "" { log.Printf("[%s] 服务端启动: step=%d 监听端口 prev=%d curr=%d next=%d 目标=%s", s.name, s.currentStep, prev, curr, next, s.target) } else { log.Printf("服务端启动: step=%d 监听端口 prev=%d curr=%d next=%d 目标=%s", s.currentStep, prev, curr, next, s.target) }
    t := time.NewTicker(time.Duration(s.cfg.StepSeconds) * time.Second)
    defer t.Stop()
    for {
        select {
        case <-ctx.Done():
            s.mu.Lock()
            for p, l := range s.listeners { l.Close(); delete(s.listeners, p) }
            s.mu.Unlock()
            return nil
        case <-t.C:
            s.currentStep++
            p2, c2, n2 := porthop.Triplet(s.secret, s.currentStep, s.cfg.PortRange.Min, s.cfg.PortRange.Max)
            newSet := map[int]struct{}{}
            for _, p := range porthop.UniquePorts([]int{p2, c2, n2}) { newSet[p] = struct{}{} }
            for p := range newSet { s.openPort(p) }
            for p := range s.listeners { if _, ok := newSet[p]; !ok { s.closePort(p); if s.name != "" { log.Printf("[%s] 服务端关闭端口: %d", s.name, p) } else { log.Printf("服务端关闭端口: %d", p) } } }
            if s.name != "" { log.Printf("[%s] 服务端轮换: step=%d 监听端口 prev=%d curr=%d next=%d", s.name, s.currentStep, p2, c2, n2) } else { log.Printf("服务端轮换: step=%d 监听端口 prev=%d curr=%d next=%d", s.currentStep, p2, c2, n2) }
        }
    }
}
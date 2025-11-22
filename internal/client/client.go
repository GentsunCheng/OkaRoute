package client

import (
    "encoding/binary"
    "io"
    "log"
    "net"
    "strconv"
    "time"
    "okaroute/internal/auth"
    "okaroute/internal/config"
    "okaroute/internal/porthop"
)

type Client struct {
    cfg config.ClientConfig
    secret []byte
    name string
}

func New(cfg config.ClientConfig, secret []byte) *Client {
    return &Client{cfg: cfg, secret: secret, name: cfg.Name}
}

func (c *Client) Start() error {
    l, err := net.Listen("tcp", net.JoinHostPort(c.cfg.BindIP, itoa(c.cfg.BindPort)))
    if err != nil { return err }
    if c.name != "" { log.Printf("[%s] 客户端本地监听: %s:%d", c.name, c.cfg.BindIP, c.cfg.BindPort) } else { log.Printf("客户端本地监听: %s:%d", c.cfg.BindIP, c.cfg.BindPort) }
    for {
        conn, err := l.Accept()
        if err != nil { return err }
        go c.handleLocal(conn)
    }
}

func itoa(i int) string { return strconv.FormatInt(int64(i), 10) }

func (c *Client) dialServerPort(step int64, host string) (net.Conn, int, error) {
    prev, curr, next := porthop.Triplet(c.secret, step, c.cfg.PortRange.Min, c.cfg.PortRange.Max)
    ports := []int{curr, prev, next}
    for _, p := range porthop.UniquePorts(ports) {
        conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, itoa(p)), 3*time.Second)
        if err == nil { return conn, p, nil }
    }
    return nil, 0, net.ErrClosed
}

func (c *Client) handleLocal(local net.Conn) {
    step := porthop.StepIndex(time.Now(), c.cfg.StepSeconds)
    rc, sp, err := c.dialServerPort(step, c.cfg.ServerHost)
    if err != nil { local.Close(); return }
    nonce, token := auth.Issue(c.secret, step, c.cfg.ClientID)
    var hdr [8 + 16 + 32]byte
    binary.BigEndian.PutUint64(hdr[0:8], uint64(step))
    copy(hdr[8:24], nonce)
    copy(hdr[24:56], token)
    rc.Write(hdr[:])
    if c.name != "" { log.Printf("[%s] 客户端建立转发: 来源=%s 服务器=%s 使用端口=%d step=%d", c.name, local.RemoteAddr().String(), c.cfg.ServerHost, sp, step) } else { log.Printf("客户端建立转发: 来源=%s 服务器=%s 使用端口=%d step=%d", local.RemoteAddr().String(), c.cfg.ServerHost, sp, step) }
    done := make(chan struct{}, 2)
    go func() { io.Copy(local, rc); done <- struct{}{} }()
    go func() { io.Copy(rc, local); done <- struct{}{} }()
    <-done
    local.Close()
    rc.Close()
}
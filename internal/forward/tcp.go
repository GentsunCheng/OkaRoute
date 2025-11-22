package forward

import (
    "io"
    "net"
    "time"
)

func pipe(a, b net.Conn) {
    defer a.Close()
    defer b.Close()
    done := make(chan struct{}, 2)
    go func() {
        io.Copy(a, b)
        done <- struct{}{}
    }()
    go func() {
        io.Copy(b, a)
        done <- struct{}{}
    }()
    <-done
}

func HandleTCP(conn net.Conn, target string) {
    conn.SetDeadline(time.Now().Add(90 * time.Second))
    dst, err := net.Dial("tcp", target)
    if err != nil {
        conn.Close()
        return
    }
    conn.SetDeadline(time.Time{})
    dst.SetDeadline(time.Time{})
    pipe(conn, dst)
}
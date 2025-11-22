package porthop

import (
    "crypto/hmac"
    "crypto/sha1"
    "encoding/base32"
    "encoding/binary"
    "math"
    "strings"
    "time"
)

func DecodeSecret(base32Str string) ([]byte, error) {
    d := base32.StdEncoding.WithPadding(base32.NoPadding)
    return d.DecodeString(strings.ToUpper(base32Str))
}

func StepIndex(now time.Time, stepSeconds int) int64 {
    return now.Unix() / int64(stepSeconds)
}

func totp(secret []byte, step int64) uint32 {
    var b [8]byte
    binary.BigEndian.PutUint64(b[:], uint64(step))
    h := hmac.New(sha1.New, secret)
    h.Write(b[:])
    sum := h.Sum(nil)
    off := sum[len(sum)-1] & 0x0f
    code := (uint32(sum[off])&0x7f)<<24 | uint32(sum[off+1])<<16 | uint32(sum[off+2])<<8 | uint32(sum[off+3])
    return code
}

func PortForStep(secret []byte, step int64, minPort, maxPort int) int {
    r := maxPort - minPort + 1
    c := totp(secret, step)
    return minPort + int(c%uint32(r))
}

func Triplet(secret []byte, step int64, minPort, maxPort int) (int, int, int) {
    prev := PortForStep(secret, step-1, minPort, maxPort)
    curr := PortForStep(secret, step, minPort, maxPort)
    next := PortForStep(secret, step+1, minPort, maxPort)
    return prev, curr, next
}

func UniquePorts(ports []int) []int {
    m := map[int]struct{}{}
    res := make([]int, 0, len(ports))
    for _, p := range ports {
        if _, ok := m[p]; !ok {
            m[p] = struct{}{}
            res = append(res, p)
        }
    }
    return res
}

func NextRotation(now time.Time, stepSeconds int) time.Duration {
    s := StepIndex(now, stepSeconds)
    base := s * int64(stepSeconds)
    next := base + int64(stepSeconds)
    return time.Duration(next-now.Unix()) * time.Second
}

func ClampSkew(step int64, current int64, skew int) bool {
    d := math.Abs(float64(step - current))
    return d <= float64(skew)
}
package auth

import (
    "crypto/hmac"
    "crypto/rand"
    "crypto/sha256"
)

func Issue(secret []byte, step int64, clientID string) ([]byte, []byte) {
    nonce := make([]byte, 16)
    rand.Read(nonce)
    mac := hmac.New(sha256.New, secret)
    var b [8]byte
    for i := 0; i < 8; i++ {
        b[7-i] = byte(step >> (8 * uint(i)))
    }
    mac.Write(b[:])
    mac.Write(nonce)
    mac.Write([]byte(clientID))
    return nonce, mac.Sum(nil)
}

func Verify(secret []byte, step int64, nonce []byte, token []byte, clientID string) bool {
    mac := hmac.New(sha256.New, secret)
    var b [8]byte
    for i := 0; i < 8; i++ {
        b[7-i] = byte(step >> (8 * uint(i)))
    }
    mac.Write(b[:])
    mac.Write(nonce)
    mac.Write([]byte(clientID))
    expect := mac.Sum(nil)
    return hmac.Equal(expect, token)
}
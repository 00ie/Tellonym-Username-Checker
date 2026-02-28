package random

import (
	"math/rand"
	"sync"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

var seeded = rand.New(rand.NewSource(time.Now().UnixNano()))
var mu sync.Mutex

func String(length int) string {
	if length <= 0 {
		return ""
	}

	b := make([]byte, length)
	mu.Lock()
	for i := range b {
		b[i] = letters[seeded.Intn(len(letters))]
	}
	mu.Unlock()
	return string(b)
}

func StringFromCharset(length int, charset string) string {
	if length <= 0 || charset == "" {
		return ""
	}

	b := make([]byte, length)
	mu.Lock()
	for i := range b {
		b[i] = charset[seeded.Intn(len(charset))]
	}
	mu.Unlock()
	return string(b)
}

func ByteFromCharset(charset string) byte {
	if charset == "" {
		return 0
	}

	mu.Lock()
	v := charset[seeded.Intn(len(charset))]
	mu.Unlock()
	return v
}

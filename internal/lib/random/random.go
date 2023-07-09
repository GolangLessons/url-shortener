package random

import (
	"math/rand"
	"time"
)

// NewRandomString generates random string with given size.
func NewRandomString(size int) string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	chl := len(chars)

	b := make([]rune, size)
	for i := range b {
		b[i] = chars[rnd.Intn(chl)]
	}

	return string(b)
}

package utils

import (
	"math/rand"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

var r* rand.Rand

var nletter = len(letterRunes)

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func GenerateRandomStr(n int) string {
	b := make([]rune, n)

	for i := range b {
		b[i] = letterRunes[r.Intn(nletter)]
	}
	return string(b)
}

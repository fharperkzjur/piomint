package utils

import (
	"math/rand"
	"strconv"
	"time"
)

var letterRunes = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

var r* rand.Rand

var nletter = len(letterRunes)

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func GenerateRandomStr(n int) string {
	b := make([]byte, n)

	for i := range b {
		b[i] = letterRunes[r.Intn(nletter)]
	}
	return string(b)
}

func GenerateRandomPasswd(n int) []byte {
	b := make([]byte, n)

	for i := range b {
		b[i] = letterRunes[r.Intn(nletter)]
	}
	return b
}

func GenerateReqId() string{
	seqno := time.Now().UTC().UnixNano()/1000
	seqno = seqno*1000 + rand.Int63n(1000)
	return strconv.FormatInt(seqno,0)
}




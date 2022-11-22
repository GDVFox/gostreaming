package util

import (
	"math/rand"
	"time"
)

// FindStringIndex возвращает индекс первого вхождения строки s в массиве a.
func FindStringIndex(a []string, s string) int {
	for i, str := range a {
		if str == s {
			return i
		}
	}
	return -1
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// RandString возвращает строку длины n состоящую из случайных символов.
func RandString(n int) string {
	rand.Seed(time.Now().Unix())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

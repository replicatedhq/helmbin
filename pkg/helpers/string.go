package helpers

import "math/rand"

var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func RandString(l int) string {
	b := make([]rune, l)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}

	return string(b)
}

func UniqueStringSlice(input []string) []string {
	var unique []string
	seen := map[string]bool{}
	for _, element := range input {
		if !seen[element] {
			unique = append(unique, element)
			seen[element] = true
		}
	}
	return unique
}

func CleanErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) > 1024 {
		msg = msg[:1024]
	}
	return msg
}

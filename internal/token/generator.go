package token

import (
	"crypto/rand"
	"encoding/hex"
)

type Generator struct{}

func New() *Generator {
	return &Generator{}
}

const tokenBytes = 32

func (g *Generator) Generate() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

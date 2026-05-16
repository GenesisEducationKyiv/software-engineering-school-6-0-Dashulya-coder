package token

import (
	"crypto/rand"
	"encoding/hex"
)

type Generator struct{}

func New() *Generator {
	return &Generator{}
}

func (g *Generator) Generate() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

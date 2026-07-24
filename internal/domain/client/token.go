package client

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const (
	TokenMarker = "wm_live_"

	tokenPrefixBytes = 8
	tokenSecretBytes = 32
)

var ErrInvalidToken = errors.New("invalid client token")

type IssuedToken struct {
	Value  string
	Prefix string
	Hash   string
}

func IssueToken() (IssuedToken, error) {
	prefix, err := randomTokenSegment(tokenPrefixBytes)
	if err != nil {
		return IssuedToken{}, fmt.Errorf("generate token prefix: %w", err)
	}
	secret, err := randomTokenSegment(tokenSecretBytes)
	if err != nil {
		return IssuedToken{}, fmt.Errorf("generate token secret: %w", err)
	}

	value := TokenMarker + prefix + "_" + secret
	hash, err := DigestToken(value)
	if err != nil {
		return IssuedToken{}, err
	}
	return IssuedToken{Value: value, Prefix: prefix, Hash: hash}, nil
}

func ParseToken(value string) (string, error) {
	prefixLength := base64.RawURLEncoding.EncodedLen(tokenPrefixBytes)
	secretLength := base64.RawURLEncoding.EncodedLen(tokenSecretBytes)
	expectedLength := len(TokenMarker) + prefixLength + 1 + secretLength
	if len(value) != expectedLength || !strings.HasPrefix(value, TokenMarker) {
		return "", ErrInvalidToken
	}

	prefixStart := len(TokenMarker)
	prefixEnd := prefixStart + prefixLength
	if value[prefixEnd] != '_' {
		return "", ErrInvalidToken
	}
	prefix := value[prefixStart:prefixEnd]
	secret := value[prefixEnd+1:]
	if !validTokenSegment(prefix, tokenPrefixBytes) || !validTokenSegment(secret, tokenSecretBytes) {
		return "", ErrInvalidToken
	}

	return prefix, nil
}

func DigestToken(value string) (string, error) {
	if _, err := ParseToken(value); err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

func randomTokenSegment(length int) (string, error) {
	value := make([]byte, length)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func validTokenSegment(value string, expectedBytes int) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != expectedBytes {
		return false
	}
	return base64.RawURLEncoding.EncodeToString(decoded) == value
}

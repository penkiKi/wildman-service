package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

func verifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("invalid password hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, fmt.Errorf("unsupported password hash version")
	}
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, fmt.Errorf("invalid password hash parameters")
	}
	if memory == 0 || memory > 256*1024 || iterations == 0 || iterations > 10 || parallelism == 0 || parallelism > 16 {
		return false, fmt.Errorf("password hash parameters outside limits")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 16 || len(salt) > 64 {
		return false, fmt.Errorf("invalid password hash salt")
	}
	expectedKey, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expectedKey) < 16 || len(expectedKey) > 64 {
		return false, fmt.Errorf("invalid password hash key")
	}

	actualKey := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expectedKey)))
	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1, nil
}

func VerifyPassword(password, encodedHash string) (bool, error) {
	return verifyPassword(password, encodedHash)
}

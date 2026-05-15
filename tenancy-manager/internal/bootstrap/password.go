// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	uppercaseChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercaseChars = "abcdefghijklmnopqrstuvwxyz"
	digitChars     = "0123456789"
	specialChars   = "!@#$%^&*()_+|:<>?="
	passwordLength = 16
)

// GeneratePassword returns a cryptographically-random 16-character password
// containing at least one uppercase letter, one lowercase letter, one
// digit, and one special character. Mirrors the behavior of the original
// tenancy-init password generator.
func GeneratePassword() (string, error) {
	if passwordLength < 4 {
		return "", fmt.Errorf("password length must be at least 4")
	}

	pw := make([]byte, passwordLength)
	var err error

	if pw[0], err = randChar(uppercaseChars); err != nil {
		return "", err
	}
	if pw[1], err = randChar(lowercaseChars); err != nil {
		return "", err
	}
	if pw[2], err = randChar(digitChars); err != nil {
		return "", err
	}
	if pw[3], err = randChar(specialChars); err != nil {
		return "", err
	}

	allChars := uppercaseChars + lowercaseChars + digitChars + specialChars
	for i := 4; i < passwordLength; i++ {
		if pw[i], err = randChar(allChars); err != nil {
			return "", err
		}
	}

	if err := shuffle(pw); err != nil {
		return "", err
	}
	return string(pw), nil
}

func randChar(charset string) (byte, error) {
	max := big.NewInt(int64(len(charset)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return charset[n.Int64()], nil
}

func shuffle(data []byte) error {
	for i := len(data) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(jBig.Int64())
		data[i], data[j] = data[j], data[i]
	}
	return nil
}

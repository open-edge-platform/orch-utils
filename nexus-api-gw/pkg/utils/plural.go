// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

var consonants = "bcdfghjklmnpqrstvwxyz"

const intTwoConstant = 2

// ToPlural returns the plural form of the type's name. If the type's name is found
// in the exceptions map, the map value is returned.
func ToPlural(t string) string {
	singular := t

	if len(singular) < intTwoConstant {
		return singular
	}

	lastChar := rune(singular[len(singular)-1])
	secondLastChar := rune(singular[len(singular)-2])

	switch lastChar {
	case 's', 'x', 'z':
		return esPlural(singular)
	case 'y':
		return handleYPlural(singular, secondLastChar)
	case 'h':
		return handleHPlural(singular, secondLastChar)
	case 'e':
		return handleEPlural(singular, secondLastChar)
	case 'f':
		return vesPlural(singular)
	default:
		return sPlural(singular)
	}
}

func handleYPlural(singular string, secondLastChar rune) string {
	if isConsonant(secondLastChar) {
		return iesPlural(singular)
	}
	return sPlural(singular)
}

func handleHPlural(singular string, secondLastChar rune) string {
	if secondLastChar == 'c' || secondLastChar == 's' {
		return esPlural(singular)
	}
	return sPlural(singular)
}

func handleEPlural(singular string, secondLastChar rune) string {
	if secondLastChar == 'f' {
		return vesPlural(singular[:len(singular)-1])
	}
	return sPlural(singular)
}

func iesPlural(singular string) string {
	return singular[:len(singular)-1] + "ies"
}

func vesPlural(singular string) string {
	return singular[:len(singular)-1] + "ves"
}

func esPlural(singular string) string {
	return singular + "es"
}

func sPlural(singular string) string {
	return singular + "s"
}

func isConsonant(char rune) bool {
	for _, c := range consonants {
		if char == c {
			return true
		}
	}
	return false
}

package edu

import (
	"strings"
	"unicode"
)

// cyrLatinMap maps Cyrillic runes to their Latin transliteration (GOST 7.79-2000 simplified).
var cyrLatinMap = map[rune]string{
	// Uppercase
	'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D",
	'Е': "E", 'Ё': "Yo", 'Ж': "Zh", 'З': "Z", 'И': "I",
	'Й': "Y", 'К': "K", 'Л': "L", 'М': "M", 'Н': "N",
	'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T",
	'У': "U", 'Ф': "F", 'Х': "Kh", 'Ц': "Ts", 'Ч': "Ch",
	'Ш': "Sh", 'Щ': "Shch", 'Ъ': "", 'Ы': "Y", 'Ь': "",
	'Э': "E", 'Ю': "Yu", 'Я': "Ya",
	// Lowercase
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d",
	'е': "e", 'ё': "yo", 'ж': "zh", 'з': "z", 'и': "i",
	'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n",
	'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
	'у': "u", 'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch",
	'ш': "sh", 'щ': "shch", 'ъ': "", 'ы': "y", 'ь': "",
	'э': "e", 'ю': "yu", 'я': "ya",
}

// Transliterate converts Cyrillic characters to Latin (GOST 7.79-2000 simplified).
// Non-Cyrillic, non-Latin, non-digit characters are removed or replaced with "-".
func Transliterate(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		if lat, ok := cyrLatinMap[r]; ok {
			b.WriteString(lat)
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if r == '-' || r == ' ' {
			b.WriteRune(r)
		}
		// other characters are dropped
	}
	return b.String()
}

// cleanUsername removes invalid characters, collapses duplicate special chars,
// and strips leading/trailing special chars. Valid chars: [a-zA-Z0-9._-].
func cleanUsername(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	prevSpecial := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevSpecial = false
		case r == '-' || r == '.' || r == '_':
			if !prevSpecial {
				b.WriteRune(r)
			}
			prevSpecial = true
		default:
			// skip invalid characters
		}
	}

	result := b.String()

	// Trim leading/trailing special characters
	result = strings.TrimFunc(result, func(r rune) bool {
		return r == '-' || r == '.' || r == '_'
	})

	return result
}

// GenerateUsername creates a username from a full name (ФИО).
// Format: "ivanov-i" (transliterated last name + "-" + first letter of first name), all lowercase.
// If >= 3 parts (space-separated), only the first two are used (patronymic is discarded).
// The result contains only valid Forgejo username characters: [a-zA-Z0-9._-].
// Truncated to 40 chars.
func GenerateUsername(fullName string) string {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return ""
	}

	parts := strings.Fields(fullName)

	var username string

	switch len(parts) {
	case 1:
		// Only last name, no first name initial
		username = strings.ToLower(Transliterate(parts[0]))
	default:
		// >= 2 parts: last name + first letter of first name; ignore 3rd+ parts
		lastName := strings.ToLower(Transliterate(parts[0]))
		firstNameTranslit := strings.ToLower(Transliterate(parts[1]))
		firstInitial := ""
		for _, r := range firstNameTranslit {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				firstInitial = string(r)
				break
			}
		}
		if firstInitial != "" {
			username = lastName + "-" + firstInitial
		} else {
			username = lastName
		}
	}

	username = cleanUsername(username)

	// Truncate to 40 characters
	if len(username) > 40 {
		username = username[:40]
		// Re-trim trailing special chars after truncation
		username = strings.TrimRight(username, "-._")
	}

	return username
}

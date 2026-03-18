package edu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransliterate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple word",
			input:    "Привет",
			expected: "Privet",
		},
		{
			name:     "Shcha",
			input:    "Щука",
			expected: "Shchuka",
		},
		{
			name:     "Mixed Cyrillic and Latin",
			input:    "Hello Мир",
			expected: "Hello Mir",
		},
		{
			name:     "Yo letter",
			input:    "Ёлка",
			expected: "Yolka",
		},
		{
			name:     "Hard and soft signs skipped",
			input:    "Объём",
			expected: "Obyom",
		},
		{
			name:     "Full name",
			input:    "Иванов Иван Иванович",
			expected: "Ivanov Ivan Ivanovich",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Digits preserved",
			input:    "Тест123",
			expected: "Test123",
		},
		{
			name:     "Hyphen preserved",
			input:    "Мухаммад-Али",
			expected: "Mukhammad-Ali",
		},
		{
			name:     "Zhenya",
			input:    "Женя",
			expected: "Zhenya",
		},
		{
			name:     "Yu and Ya",
			input:    "Юля Яна",
			expected: "Yulya Yana",
		},
		{
			name:     "Lowercase",
			input:    "привет мир",
			expected: "privet mir",
		},
		{
			name:     "Kh sound",
			input:    "Хорошо",
			expected: "Khorosho",
		},
		{
			name:     "Ts sound",
			input:    "Цирк",
			expected: "Tsirk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Transliterate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard full name with patronymic",
			input:    "Иванов Иван Иванович",
			expected: "ivanov-i",
		},
		{
			name:     "Two-part name",
			input:    "Петрова Анна",
			expected: "petrova-a",
		},
		{
			name:     "Hyphenated parts with patronymic",
			input:    "Мухаммад-Али Аль-Хасан Ибрагимович",
			expected: "mukhammad-ali-a",
		},
		{
			name:     "Latin name",
			input:    "O'Brien James",
			expected: "obrien-j",
		},
		{
			name:     "Single name only",
			input:    "Сидоров",
			expected: "sidorov",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "Extra spaces between parts",
			input:    "  Козлов   Дмитрий  ",
			expected: "kozlov-d",
		},
		{
			name:     "Four-part name",
			input:    "Иванов Пётр Сергеевич Младший",
			expected: "ivanov-p",
		},
		{
			name:     "Name with Yo",
			input:    "Ёлкин Ёж",
			expected: "yolkin-y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateUsername(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateUsername_MaxLength(t *testing.T) {
	// Create a very long name that would exceed 40 chars
	longName := "Абвгдеёжзийклмнопрстуфхцчшщъыьэюя Тест"
	result := GenerateUsername(longName)
	assert.LessOrEqual(t, len(result), 40)
	assert.NotEmpty(t, result)

	// Check no trailing special chars
	last := result[len(result)-1]
	assert.True(t, (last >= 'a' && last <= 'z') || (last >= '0' && last <= '9'),
		"username should not end with special character, got: %s", result)
}

func TestCleanUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No change needed",
			input:    "ivanov-i",
			expected: "ivanov-i",
		},
		{
			name:     "Leading dash removed",
			input:    "-ivanov",
			expected: "ivanov",
		},
		{
			name:     "Trailing dash removed",
			input:    "ivanov-",
			expected: "ivanov",
		},
		{
			name:     "Double dashes collapsed",
			input:    "ivanov--i",
			expected: "ivanov-i",
		},
		{
			name:     "Invalid characters removed",
			input:    "ivan@ov#i",
			expected: "ivanovi",
		},
		{
			name:     "Dots and underscores preserved",
			input:    "ivan.ov_i",
			expected: "ivan.ov_i",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanUsername(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

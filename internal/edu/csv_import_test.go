package edu

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
)

func TestParseCSV_WithHeaderSemicolon(t *testing.T) {
	data := []byte("ФИО;Email;Группа\nИванов Иван Иванович;ivanov@example.com;ИУ7-11Б\nПетрова Анна;petrova@example.com;ИУ7-12Б\n")

	mapping := CSVColumnMapping{
		FullNameCol: 0,
		EmailCol:    1,
		GroupCol:    2,
		HasHeader:   true,
	}

	rows, err := ParseCSV(data, mapping)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, "Иванов Иван Иванович", rows[0].FullName)
	assert.Equal(t, "ivanov@example.com", rows[0].Email)
	assert.Equal(t, "ИУ7-11Б", rows[0].Group)
	assert.Equal(t, "ivanov-ii", rows[0].Username)

	assert.Equal(t, "Петрова Анна", rows[1].FullName)
	assert.Equal(t, "petrova@example.com", rows[1].Email)
	assert.Equal(t, "ИУ7-12Б", rows[1].Group)
	assert.Equal(t, "petrova-a", rows[1].Username)
}

func TestParseCSV_NoHeaderComma(t *testing.T) {
	data := []byte("Сидоров Пётр Петрович,sidorov@mail.ru\nКозлова Мария,kozlova@mail.ru\n")

	mapping := CSVColumnMapping{
		FullNameCol: 0,
		EmailCol:    1,
		GroupCol:    -1,
		HasHeader:   false,
	}

	rows, err := ParseCSV(data, mapping)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, "Сидоров Пётр Петрович", rows[0].FullName)
	assert.Equal(t, "sidorov@mail.ru", rows[0].Email)
	assert.Equal(t, "", rows[0].Group)
	assert.Equal(t, "sidorov-pp", rows[0].Username)

	assert.Equal(t, "Козлова Мария", rows[1].FullName)
	assert.Equal(t, "kozlova@mail.ru", rows[1].Email)
	assert.Equal(t, "kozlova-m", rows[1].Username)
}

func TestParseCSV_EmptyLinesSkipped(t *testing.T) {
	data := []byte("ФИО;Email\nИванов Иван;ivan@test.com\n\n  \n\nПетров Пётр;petr@test.com\n")

	mapping := CSVColumnMapping{
		FullNameCol: 0,
		EmailCol:    1,
		GroupCol:    -1,
		HasHeader:   true,
	}

	rows, err := ParseCSV(data, mapping)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, "Иванов Иван", rows[0].FullName)
	assert.Equal(t, "Петров Пётр", rows[1].FullName)
}

func TestParseCSV_Windows1251Encoding(t *testing.T) {
	// Encode test data as Windows-1251
	encoder := charmap.Windows1251.NewEncoder()

	utf8Data := "ФИО;Группа\nИванов Иван;ИУ7-11\nПетрова Анна;ИУ7-12\n"
	win1251Data, err := encoder.Bytes([]byte(utf8Data))
	require.NoError(t, err)

	mapping := CSVColumnMapping{
		FullNameCol: 0,
		EmailCol:    -1,
		GroupCol:    1,
		HasHeader:   true,
	}

	rows, err := ParseCSV(win1251Data, mapping)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, "Иванов Иван", rows[0].FullName)
	assert.Equal(t, "ИУ7-11", rows[0].Group)
	assert.Equal(t, "ivanov-i", rows[0].Username)

	assert.Equal(t, "Петрова Анна", rows[1].FullName)
	assert.Equal(t, "ИУ7-12", rows[1].Group)
	assert.Equal(t, "petrova-a", rows[1].Username)
}

func TestParseCSV_WithBOM(t *testing.T) {
	// UTF-8 BOM + CSV data
	bom := []byte{0xEF, 0xBB, 0xBF}
	csvData := []byte("ФИО,Email\nИванов Иван,ivan@test.com\n")
	data := append(bom, csvData...)

	mapping := CSVColumnMapping{
		FullNameCol: 0,
		EmailCol:    1,
		GroupCol:    -1,
		HasHeader:   true,
	}

	rows, err := ParseCSV(data, mapping)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	assert.Equal(t, "Иванов Иван", rows[0].FullName)
	assert.Equal(t, "ivan@test.com", rows[0].Email)
	assert.Equal(t, "ivanov-i", rows[0].Username)
}

func TestParseCSV_ColumnMappingWithAllFields(t *testing.T) {
	// Columns in non-standard order: Group, Email, FullName
	data := []byte("ИУ7-11;ivanov@mail.ru;Иванов Иван Иванович\nИУ7-12;petrova@mail.ru;Петрова Анна\n")

	mapping := CSVColumnMapping{
		FullNameCol: 2,
		EmailCol:    1,
		GroupCol:    0,
		HasHeader:   false,
	}

	rows, err := ParseCSV(data, mapping)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, "Иванов Иван Иванович", rows[0].FullName)
	assert.Equal(t, "ivanov@mail.ru", rows[0].Email)
	assert.Equal(t, "ИУ7-11", rows[0].Group)
	assert.Equal(t, "ivanov-ii", rows[0].Username)

	assert.Equal(t, "Петрова Анна", rows[1].FullName)
	assert.Equal(t, "petrova@mail.ru", rows[1].Email)
	assert.Equal(t, "ИУ7-12", rows[1].Group)
	assert.Equal(t, "petrova-a", rows[1].Username)
}

func TestParseCSV_EmptyData(t *testing.T) {
	mapping := CSVColumnMapping{
		FullNameCol: 0,
		EmailCol:    -1,
		GroupCol:    -1,
		HasHeader:   false,
	}

	rows, err := ParseCSV([]byte(""), mapping)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestParseCSV_OnlyHeader(t *testing.T) {
	data := []byte("ФИО;Email\n")

	mapping := CSVColumnMapping{
		FullNameCol: 0,
		EmailCol:    1,
		GroupCol:    -1,
		HasHeader:   true,
	}

	rows, err := ParseCSV(data, mapping)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestParseCSV_RowWithEmptyFullName(t *testing.T) {
	data := []byte("ФИО;Email\n;empty@test.com\nИванов Иван;ivan@test.com\n")

	mapping := CSVColumnMapping{
		FullNameCol: 0,
		EmailCol:    1,
		GroupCol:    -1,
		HasHeader:   true,
	}

	rows, err := ParseCSV(data, mapping)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	assert.Equal(t, "Иванов Иван", rows[0].FullName)
}

func TestDetectDelimiter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected rune
	}{
		{
			name:     "Semicolons",
			input:    "ФИО;Email;Группа\n",
			expected: ';',
		},
		{
			name:     "Commas",
			input:    "ФИО,Email,Группа\n",
			expected: ',',
		},
		{
			name:     "No delimiters defaults to comma",
			input:    "ФИО\n",
			expected: ',',
		},
		{
			name:     "More semicolons than commas",
			input:    "a;b;c,d\n",
			expected: ';',
		},
		{
			name:     "More commas than semicolons",
			input:    "a,b,c;d\n",
			expected: ',',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectDelimiter([]byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStripBOM(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}

	t.Run("With BOM", func(t *testing.T) {
		data := append(bom, []byte("hello")...)
		result := stripBOM(data)
		assert.Equal(t, []byte("hello"), result)
	})

	t.Run("Without BOM", func(t *testing.T) {
		data := []byte("hello")
		result := stripBOM(data)
		assert.Equal(t, []byte("hello"), result)
	})
}

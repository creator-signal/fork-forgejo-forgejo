package edu

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// ImportRow represents a single parsed row from a CSV import file.
type ImportRow struct {
	FullName string
	Email    string
	Group    string
	Username string
}

// CSVColumnMapping defines which CSV columns map to which fields.
type CSVColumnMapping struct {
	FullNameCol int  // required, column index for the full name
	EmailCol    int  // -1 if not present
	GroupCol    int  // -1 if not present
	HasHeader   bool // if true, the first row is skipped
}

// utf8BOM is the byte sequence for UTF-8 BOM.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// ParseCSV parses CSV data into ImportRow slices.
// It handles BOM stripping, Windows-1251 detection, and delimiter detection (comma vs semicolon).
func ParseCSV(data []byte, mapping CSVColumnMapping) ([]ImportRow, error) {
	// Strip UTF-8 BOM if present
	data = stripBOM(data)

	// Detect and convert encoding if needed
	data = ensureUTF8(data)

	// Detect delimiter
	delimiter := detectDelimiter(data)

	// Parse CSV
	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // allow variable number of fields

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	startIdx := 0
	if mapping.HasHeader {
		startIdx = 1
	}

	var rows []ImportRow
	for i := startIdx; i < len(records); i++ {
		record := records[i]

		// Get full name
		fullName := getField(record, mapping.FullNameCol)
		fullName = strings.TrimSpace(fullName)
		if fullName == "" {
			continue // skip rows with empty full name
		}

		row := ImportRow{
			FullName: fullName,
			Username: GenerateUsername(fullName),
		}

		// Get email if column is mapped
		if mapping.EmailCol >= 0 {
			row.Email = strings.TrimSpace(getField(record, mapping.EmailCol))
		}

		// Get group if column is mapped
		if mapping.GroupCol >= 0 {
			row.Group = strings.TrimSpace(getField(record, mapping.GroupCol))
		}

		rows = append(rows, row)
	}

	return rows, nil
}

// stripBOM removes the UTF-8 BOM from the beginning of data if present.
func stripBOM(data []byte) []byte {
	if bytes.HasPrefix(data, utf8BOM) {
		return data[len(utf8BOM):]
	}
	return data
}

// ensureUTF8 checks if data is valid UTF-8; if not, attempts to decode as Windows-1251.
func ensureUTF8(data []byte) []byte {
	if utf8.Valid(data) {
		return data
	}

	// Check if data likely contains Windows-1251 encoded Cyrillic (bytes 0xC0-0xFF)
	hasCyrillicRange := false
	for _, b := range data {
		if b >= 0xC0 && b <= 0xFF {
			hasCyrillicRange = true
			break
		}
	}

	if !hasCyrillicRange {
		return data // not Windows-1251 Cyrillic, return as-is
	}

	// Decode from Windows-1251
	decoder := charmap.Windows1251.NewDecoder()
	decoded, err := decoder.Bytes(data)
	if err != nil {
		return data // fallback to original
	}
	return decoded
}

// detectDelimiter counts semicolons vs commas in the first non-empty line
// and returns the more frequent one.
func detectDelimiter(data []byte) rune {
	// Find first non-empty line
	reader := bytes.NewReader(data)
	line := ""
	for {
		b, err := reader.ReadByte()
		if err == io.EOF {
			break
		}
		if b == '\n' {
			line = strings.TrimSpace(line)
			if line != "" {
				break
			}
			continue
		}
		if b != '\r' {
			line += string(b)
		}
	}

	semicolons := strings.Count(line, ";")
	commas := strings.Count(line, ",")

	if semicolons > commas {
		return ';'
	}
	return ','
}

// getField safely retrieves a field from a CSV record by index.
func getField(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}
	return record[index]
}

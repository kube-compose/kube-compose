package util

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

const chars = "abcdefghijklmnopqrstuvwxyz0123456789"

type HasSubexpNames interface {
	SubexpNames() []string
}

// BuildRegexpMatchMap creates a map from a regular expression, and a match slice obtained from r.FindStringSubmatch or
// r.FindAllStringSubmatch.
func BuildRegexpMatchMap(r HasSubexpNames, matches []string) map[string]string {
	subexpNames := r.SubexpNames()
	n := len(subexpNames)
	matchMap := map[string]string{}
	for i := 1; i < n; i++ {
		if len(subexpNames[i]) > 0 {
			matchMap[subexpNames[i]] = matches[i]
		}
	}
	return matchMap
}

// CloseAndLogError closes the closer and logs any error it returns.
func CloseAndLogError(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		fmt.Println(err)
	}
}

func decodeBase36(b int) int {
	if b <= '9' {
		if '0' <= b {
			return 26 - '0' + b
		}
	} else if 'a' <= b && b <= 'z' {
		return b - 'a'
	}
	return -1
}

// EscapeName takes an arbitrary string and maps it bijectively to the grammar '^[a-z0-9]([-a-z0-9]*[a-z0-9])?$'.
// This is useful when creating Kubernetes resources.
func EscapeName(input string) string {
	n := len(input)
	var sb strings.Builder
	for i := 0; i < n; i++ {
		b := input[i]
		if (b >= '0' && b <= '8') || (b >= 'a' && b <= 'z') {
			sb.WriteByte(b)
			continue
		} else if b == '-' && i > 0 && i < n-1 {
			sb.WriteByte(b)
			continue
		}
		sb.WriteByte('9')
		sb.WriteByte(chars[b/36])
		sb.WriteByte(chars[b%36])
	}
	return sb.String()
}

// TryParseInt64 is a convenience method to parse a string into an *int64, allowing only one or more ASCII digits and an optional sign
// prefix.
func TryParseInt64(s string) *int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &i
}

// UnescapeName performs the reverse transformation of EscapeName.
func UnescapeName(input string) (string, error) {
	var sb strings.Builder
	i := 0
	for i < len(input) {
		if input[i] == '9' {
			b, err := unescapeByte(input, i)
			if err != nil {
				return "", err
			}
			sb.WriteByte(b)
			i += 3
		} else {
			sb.WriteByte(input[i])
			i++
		}
	}
	return sb.String(), nil
}

func unescapeByte(input string, i int) (byte, error) {
	if len(input)-i >= 3 {
		d1 := decodeBase36(int(input[i+1]))
		if d1 >= 0 {
			d2 := decodeBase36(int(input[i+2]))
			if d2 >= 0 {
				b := d1*36 + d2
				if b < 256 {
					return byte(b), nil
				}
			}
		}

	}
	return 0, fmt.Errorf("invalid input")
}

func FormatTable(rows [][]string) string {
	maxValueWidthPerColumn := []int{}
	for _, row := range rows {
		for column, value := range row {
			for len(maxValueWidthPerColumn) <= column {
				maxValueWidthPerColumn = append(maxValueWidthPerColumn, 0)
			}
			if len(value) > maxValueWidthPerColumn[column] {
				maxValueWidthPerColumn[column] = len(value)
			}
		}
	}
	sb := strings.Builder{}
	for _, row := range rows {
		for column, value := range row {
			if column+1 >= len(maxValueWidthPerColumn) {
				_, _ = fmt.Fprintf(&sb, value)
			} else {
				_, _ = fmt.Fprintf(&sb, fmt.Sprintf("%%-%ds  ", maxValueWidthPerColumn[column]), value)
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

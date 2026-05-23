package core

import (
	"strings"
	"unicode/utf8"
)

const MaxQueryRunes = 160

func NormalizeQuery(query string) string {
	query = strings.ToLower(strings.TrimSpace(query))
	query = strings.Join(strings.Fields(query), " ")
	if utf8.RuneCountInString(query) <= MaxQueryRunes {
		return query
	}

	runes := []rune(query)
	return string(runes[:MaxQueryRunes])
}

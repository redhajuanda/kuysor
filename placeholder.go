package kuysor

import (
	"fmt"
	"strings"
)

type PlaceHolderType uint8

const (
	Question PlaceHolderType = iota
	Dollar
	At
	Colon
)

// findOrderOfInternalPlaceholders finds the indices of internal placeholders
// Returns the indices where internal placeholder appears in the order of all parameters
func findOrderOfInternalPlaceholders(query string) []int {
	tokens := tokenizeQuery(query)
	var indices []int
	currentParam := 0
	for _, token := range tokens {
		if token.tokenType == "placeholder" {
			if token.value == defaultInternalPlaceHolder {
				indices = append(indices, currentParam)
			}
			currentParam++
		}
	}
	return indices
}

// replacePlaceholders replaces internal placeholders with the appropriate placeholders
// based on the placeholder type
func replacePlaceholders(query string, placeholderType PlaceHolderType) string {
	tokens := tokenizeQuery(query)
	var placeholders []Token
	for _, token := range tokens {
		if token.tokenType == "placeholder" {
			placeholders = append(placeholders, token)
		}
	}
	if len(placeholders) == 0 {
		return query
	}
	if placeholderType == Question {
		result := query
		offset := 0
		for _, p := range placeholders {
			if p.value == defaultInternalPlaceHolder {
				originalPos := p.position
				replacement := "?"
				adjustedPos := originalPos + offset
				before := result[:adjustedPos]
				after := result[adjustedPos+2:]
				result = before + replacement + after
				offset += len(replacement) - 2
			}
		}
		return result
	}
	parameterNumber := 1
	paramMap := make(map[int]int)
	for i := range placeholders {
		paramMap[i] = parameterNumber
		parameterNumber++
	}
	result := query
	offset := 0
	for i, p := range placeholders {
		originalPos := p.position
		var replacement string
		switch placeholderType {
		case Dollar:
			replacement = fmt.Sprintf("$%d", paramMap[i])
		case At:
			replacement = fmt.Sprintf("@p%d", paramMap[i])
		case Colon:
			replacement = fmt.Sprintf(":%d", paramMap[i])
		}
		if p.value == defaultInternalPlaceHolder {
			adjustedPos := originalPos + offset
			before := result[:adjustedPos]
			after := result[adjustedPos+2:]
			result = before + replacement + after
			offset += len(replacement) - 2
		} else if placeholderType == Dollar && strings.HasPrefix(p.value, "$") {
			numLength := len(p.value) - 1
			adjustedPos := originalPos + offset
			before := result[:adjustedPos]
			after := result[adjustedPos+numLength+1:]
			result = before + replacement + after
			offset += len(replacement) - (numLength + 1)
		} else if placeholderType == At && strings.HasPrefix(p.value, "@p") {
			numLength := len(p.value) - 2
			adjustedPos := originalPos + offset
			before := result[:adjustedPos]
			after := result[adjustedPos+numLength+2:]
			result = before + replacement + after
			offset += len(replacement) - (numLength + 2)
		} else if placeholderType == Colon && strings.HasPrefix(p.value, ":") {
			numLength := len(p.value) - 1
			adjustedPos := originalPos + offset
			before := result[:adjustedPos]
			after := result[adjustedPos+numLength+1:]
			result = before + replacement + after
			offset += len(replacement) - (numLength + 1)
		}
	}
	return result
}

// extractNumber extracts a number from a string
func extractNumber(s string) int {
	var num int
	fmt.Sscanf(s, "%d", &num)
	return num
}

// Token represents a token in the SQL query
type Token struct {
	position  int
	value     string
	tokenType string
}

// tokenizeQuery breaks down a SQL query into tokens
func tokenizeQuery(query string) []Token {
	var tokens []Token
	i := 0
	for i < len(query) {
		if i < len(query) && (query[i] == '\'' || query[i] == '"' || query[i] == '`') {
			quoteChar := query[i]
			startPos := i
			i++
			escaped := false
			for i < len(query) && (query[i] != quoteChar || escaped) {
				escaped = query[i] == '\\' && !escaped
				i++
			}
			if i < len(query) {
				i++
			}
			tokens = append(tokens, Token{
				position:  startPos,
				value:     query[startPos:i],
				tokenType: "quoted",
			})
			continue
		}
		if i < len(query)-1 && query[i] == '$' && query[i+1] == '0' {
			tokens = append(tokens, Token{
				position:  i,
				value:     defaultInternalPlaceHolder,
				tokenType: "placeholder",
			})
			i += 2
			continue
		}
		if i < len(query) && query[i] == '?' {
			tokens = append(tokens, Token{
				position:  i,
				value:     "?",
				tokenType: "placeholder",
			})
			i++
			continue
		}
		if i < len(query)-1 && query[i] == '$' && isDigit(query[i+1]) {
			startPos := i
			i++
			for i < len(query) && isDigit(query[i]) {
				i++
			}
			tokens = append(tokens, Token{
				position:  startPos,
				value:     query[startPos:i],
				tokenType: "placeholder",
			})
			continue
		}
		if i < len(query)-2 && query[i] == '@' && query[i+1] == 'p' && i+2 < len(query) && isDigit(query[i+2]) {
			startPos := i
			i += 2
			for i < len(query) && isDigit(query[i]) {
				i++
			}
			tokens = append(tokens, Token{
				position:  startPos,
				value:     query[startPos:i],
				tokenType: "placeholder",
			})
			continue
		}
		// Colon placeholder support
		if i < len(query)-1 && query[i] == ':' && isDigit(query[i+1]) {
			startPos := i
			i++
			for i < len(query) && isDigit(query[i]) {
				i++
			}
			tokens = append(tokens, Token{
				position:  startPos,
				value:     query[startPos:i],
				tokenType: "placeholder",
			})
			continue
		}
		startPos := i
		for i < len(query) &&
			!(query[i] == '\'' || query[i] == '"' || query[i] == '`') &&
			!(i < len(query)-1 && query[i] == '$' && query[i+1] == '0') &&
			!(query[i] == '?') &&
			!(i < len(query)-1 && query[i] == '$' && isDigit(query[i+1])) &&
			!(i < len(query)-2 && query[i] == '@' && query[i+1] == 'p' && i+2 < len(query) && isDigit(query[i+2])) &&
			!(i < len(query)-1 && query[i] == ':' && isDigit(query[i+1])) {
			i++
		}
		if i > startPos {
			tokens = append(tokens, Token{
				position:  startPos,
				value:     query[startPos:i],
				tokenType: "text",
			})
		} else {
			i++
		}
	}
	return tokens
}

// Helper function to check if a character is a digit
func isDigit(c byte) bool {
	return c >= '1' && c <= '9'
}

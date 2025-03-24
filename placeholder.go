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
)

// findOrderOfInternalPlaceholders finds the indices of internal placeholders
// Returns the indices where internal placeholder appears in the order of all parameters
func findOrderOfInternalPlaceholders(query string) []int {

	// Parse through the query and find all placeholders
	tokens := tokenizeQuery(query)

	// Find the positions of &v placeholders
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

	// Parse through the query and find all placeholders
	tokens := tokenizeQuery(query)

	// Find all placeholders and their positions
	var placeholders []Token
	for _, token := range tokens {
		if token.tokenType == "placeholder" {
			placeholders = append(placeholders, token)
		}
	}

	if len(placeholders) == 0 {
		return query
	}

	// For Question type, no numbering needed
	if placeholderType == Question {
		result := query
		offset := 0
		for _, p := range placeholders {
			if p.value == defaultInternalPlaceHolder {
				originalPos := p.position
				replacement := "?"
				adjustedPos := originalPos + offset
				before := result[:adjustedPos]
				after := result[adjustedPos+2:] // +2 to skip "&v"
				result = before + replacement + after
				offset += len(replacement) - 2 // -2 for original "&v"
			}
		}
		return result
	}

	// For DOLLAR and AT types, we need to renumber all placeholders sequentially
	// based on their order of appearance in the query
	parameterNumber := 1
	paramMap := make(map[int]int) // Maps placeholder index to parameter number

	// First pass: assign sequential numbers to all placeholders
	for i := range placeholders {
		paramMap[i] = parameterNumber
		parameterNumber++
	}

	// Prepare for replacement
	result := query
	offset := 0

	// Replace all placeholders with their sequential numbers
	for i, p := range placeholders {
		originalPos := p.position
		var replacement string

		switch placeholderType {
		case Dollar:
			replacement = fmt.Sprintf("$%d", paramMap[i])
		case At:
			replacement = fmt.Sprintf("@p%d", paramMap[i])
		}

		// Only replace &v placeholders
		if p.value == defaultInternalPlaceHolder {
			// Apply replacement
			adjustedPos := originalPos + offset
			before := result[:adjustedPos]
			after := result[adjustedPos+2:] // +2 to skip "&v"
			result = before + replacement + after
			offset += len(replacement) - 2 // -2 for original "&v"
		} else if placeholderType == Dollar && strings.HasPrefix(p.value, "$") {
			// Replace existing $n placeholders
			numLength := len(p.value) - 1 // -1 for the $ character
			adjustedPos := originalPos + offset
			before := result[:adjustedPos]
			after := result[adjustedPos+numLength+1:] // +1 for the $ character
			result = before + replacement + after
			offset += len(replacement) - (numLength + 1)
		} else if placeholderType == At && strings.HasPrefix(p.value, "@p") {
			// Replace existing @pn placeholders
			numLength := len(p.value) - 2 // -2 for the @p prefix
			adjustedPos := originalPos + offset
			before := result[:adjustedPos]
			after := result[adjustedPos+numLength+2:] // +2 for the @p prefix
			result = before + replacement + after
			offset += len(replacement) - (numLength + 2)
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
	position  int    // Position in original string
	value     string // Token value
	tokenType string // Type: "text", "quoted", "placeholder"
}

// tokenizeQuery breaks down a SQL query into tokens
func tokenizeQuery(query string) []Token {
	var tokens []Token
	i := 0

	for i < len(query) {
		// Check for quoted strings
		if i < len(query) && (query[i] == '\'' || query[i] == '"' || query[i] == '`') {
			// Extract quoted string
			quoteChar := query[i]
			startPos := i
			i++ // Move past opening quote

			escaped := false
			for i < len(query) && (query[i] != quoteChar || escaped) {
				escaped = query[i] == '\\' && !escaped
				i++
			}

			if i < len(query) {
				i++ // Move past closing quote
			}

			tokens = append(tokens, Token{
				position:  startPos,
				value:     query[startPos:i],
				tokenType: "quoted",
			})
			continue
		}

		// Check for &v placeholder
		if i < len(query)-1 && query[i] == '$' && query[i+1] == '0' {
			tokens = append(tokens, Token{
				position:  i,
				value:     defaultInternalPlaceHolder,
				tokenType: "placeholder",
			})
			i += 2
			continue
		}

		// Check for ? placeholder
		if i < len(query) && query[i] == '?' {
			tokens = append(tokens, Token{
				position:  i,
				value:     "?",
				tokenType: "placeholder",
			})
			i++
			continue
		}

		// Check for $n placeholder
		if i < len(query)-1 && query[i] == '$' && isDigit(query[i+1]) {
			startPos := i
			i++ // Skip $

			// Read all digits
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

		// Check for @pn placeholder
		if i < len(query)-2 && query[i] == '@' && query[i+1] == 'p' && i+2 < len(query) && isDigit(query[i+2]) {
			startPos := i
			i += 2 // Skip @p

			// Read all digits
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

		// Regular text
		startPos := i
		for i < len(query) &&
			!(query[i] == '\'' || query[i] == '"' || query[i] == '`') &&
			!(i < len(query)-1 && query[i] == '$' && query[i+1] == '0') &&
			!(query[i] == '?') &&
			!(i < len(query)-1 && query[i] == '$' && isDigit(query[i+1])) &&
			!(i < len(query)-2 && query[i] == '@' && query[i+1] == 'p' && i+2 < len(query) && isDigit(query[i+2])) {
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

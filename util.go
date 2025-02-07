package kuysor

import "regexp"

// replaceBindVariables replaces all bind variables with format
// :v1, :v2, etc to ?
func replaceBindVariables(sql string) string {
	// replace all bind variables with format :v1, :v2, etc to ?
	re := regexp.MustCompile(`:\bv\d+\b`)

	// Ganti semua yang match dengan ?
	return re.ReplaceAllString(sql, "?")

}

// Function to reverse a slice of maps
func reverse(data *[]map[string]interface{}) {
	for i, j := 0, len(*data)-1; i < j; i, j = i+1, j-1 {
		(*data)[i], (*data)[j] = (*data)[j], (*data)[i]
	}
}

// Function to delete an element at index from a slice
func deleteElement(s *[]map[string]interface{}, index int) {
	if index < 0 || index >= len(*s) {
		panic("index out of range")
	}
	*s = append((*s)[:index], (*s)[index+1:]...)
}

package kuysor

import (
	"encoding/base64"
	"reflect"
	"regexp"
)

// replaceBindVariables replaces all bind variables with format
// :v1, :v2, etc to ?
func replaceBindVariables(sql string) string {
	// replace all bind variables with format :v1, :v2, etc to ?
	re := regexp.MustCompile(`:\bv\d+\b`)
	return re.ReplaceAllString(sql, "?")

}

func findParamOrder(query, param string) []int {
	var indexes []int
	// Regex to match :v0, :v1, :v2, ...
	re := regexp.MustCompile(`:(v\d+)`)
	matches := re.FindAllString(query, -1)

	// Loop through all matches and check which ones are equal to param
	for i, match := range matches {
		if match == param {
			indexes = append(indexes, i)
		}
	}

	return indexes
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

// base64Encode encodes the string into base64.
func base64Encode(cursor string) string {

	return base64.StdEncoding.EncodeToString([]byte(cursor))

}

// base64Decode decodes the base64 string.
func base64Decode(s string) (string, error) {

	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decoded), nil

}

// getFieldValueByTag gets a field value from a struct using the tag key
func getFieldValueByTag(item reflect.Value, columnName string, tagKey string) interface{} {
	// Get the type of the struct
	itemType := item.Type()
	if item.Kind() == reflect.Ptr {
		item = item.Elem()
		itemType = item.Type()
	}

	// Iterate through fields to find matching tag
	for i := 0; i < itemType.NumField(); i++ {
		field := itemType.Field(i)
		if tag := field.Tag.Get(tagKey); tag == columnName {
			return item.Field(i).Interface()
		}
	}
	return nil
}

// Helper function to reverse a slice using reflection
func reverseSlice(sliceVal reflect.Value) {
	for i := 0; i < sliceVal.Len()/2; i++ {
		j := sliceVal.Len() - i - 1
		tmp := sliceVal.Index(i).Interface()
		sliceVal.Index(i).Set(sliceVal.Index(j))
		sliceVal.Index(j).Set(reflect.ValueOf(tmp))
	}
}

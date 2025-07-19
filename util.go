package kuysor

import (
	"encoding/base64"
	"fmt"
	"reflect"
)

// Function to reverse a slice of maps
func reverse(data *[]map[string]any) {
	for i, j := 0, len(*data)-1; i < j; i, j = i+1, j-1 {
		(*data)[i], (*data)[j] = (*data)[j], (*data)[i]
	}
}

// Function to delete an element at index from a slice
func deleteElement(s *[]map[string]any, index int) error {
	if index < 0 || index >= len(*s) {
		return fmt.Errorf("index out of range: %d", index)
	}
	*s = append((*s)[:index], (*s)[index+1:]...)
	return nil
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
// now supports embedded structs
func getFieldValueByTag(item reflect.Value, columnName string, tagKey string) any {

	// Get the type of the struct
	itemType := item.Type()
	if item.Kind() == reflect.Ptr {
		item = item.Elem()
		itemType = item.Type()
	}

	// Iterate through fields to find matching tag
	for i := 0; i < itemType.NumField(); i++ {
		field := itemType.Field(i)
		fieldValue := item.Field(i)

		// Check if current field has the matching tag
		if tag := field.Tag.Get(tagKey); tag == columnName {
			return fieldValue.Interface()
		}

		// Check if this is an embedded struct (anonymous field)
		if field.Anonymous && fieldValue.Kind() == reflect.Struct {
			// Recursively search in the embedded struct
			if result := getFieldValueByTag(fieldValue, columnName, tagKey); result != nil {
				return result
			}
		}

		// Also handle embedded pointer to struct
		if field.Anonymous && fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil() && fieldValue.Elem().Kind() == reflect.Struct {
			if result := getFieldValueByTag(fieldValue.Elem(), columnName, tagKey); result != nil {
				return result
			}
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

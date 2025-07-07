package kuysor

import (
	"encoding/json"
	"fmt"
)

// vCursor is the cursor struct for internal use.
type vCursor struct {
	Prefix cursorPrefix   `json:"prefix"`
	Cols   map[string]any `json:"cols"`
	cursor cursorBase64   `json:"-"`
}

// generateCursorBase64 generates the cursor base64 from vCursor.
func (v *vCursor) generateCursorBase64() (cursorBase64, error) {

	item, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cursor: %v", err)
	}

	v.cursor = cursorBase64(base64Encode(string(item)))
	return v.cursor, nil

}

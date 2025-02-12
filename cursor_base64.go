package kuysor

import (
	"encoding/json"
	"fmt"
)

// cursorBase64 is the base64 encoded cursor.
type cursorBase64 string

// parse parses the cursor base64 into vCursor.
func (c cursorBase64) parse() (*vCursor, error) {

	var (
		vcursor vCursor
	)

	// decode cursor from base64
	decodedCursor, err := base64Decode(string(c))
	if err != nil {
		return nil, err
	}

	// unmarshal cursor
	err = json.Unmarshal([]byte(decodedCursor), &vcursor)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cursor: %v", err)
	}

	vcursor.cursor = c

	return &vcursor, nil
}

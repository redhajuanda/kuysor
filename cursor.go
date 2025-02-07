package kuysor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type CursorPrefix string

const (
	CursorPrefixNext CursorPrefix = "next"
	CursorPrefixPrev CursorPrefix = "prev"
)

type vCursor struct {
	Prefix CursorPrefix      `json:"prefix"`
	Cols   map[string]string `json:"cols"`
	Id     map[string]string `json:"id"`
	cursor string            `json:"-"`
}

func (v *vCursor) isNext() bool {
	return v.Prefix == CursorPrefixNext
}

func (v *vCursor) isPrev() bool {
	return v.Prefix == CursorPrefixPrev
}

// parseCursor parses the cursor.
func parseCursor(cursor string) (*vCursor, error) {

	var (
		vcursor vCursor
	)

	// decode cursor from base64
	decodedCursor, err := decodeCursor(cursor)
	if err != nil {
		return nil, err
	}

	// unmarshal cursor
	err = json.Unmarshal([]byte(decodedCursor), &vcursor)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal cursor: %v", err)
	}

	vcursor.cursor = cursor

	return &vcursor, nil
}

// generateCursor generates the cursor.
func generateCursor(vCursor vCursor) (string, error) {

	item, err := json.Marshal(vCursor)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cursor: %v", err)
	}

	return encodeCursor(string(item)), nil

}

// encodeCursor encodes the cursor.
func encodeCursor(cursor string) string {
	return base64.StdEncoding.EncodeToString([]byte(cursor))
}

// decodeCursor decodes the cursor.
func decodeCursor(cursor string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

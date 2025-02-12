package kuysor

// cursorPrefix is the cursor prefix.
type cursorPrefix string

const (
	cursorPrefixNext cursorPrefix = "next"
	cursorPrefixPrev cursorPrefix = "prev"
)

// isNext returns true if the prefix is next.
func (p cursorPrefix) isNext() bool {
	return p == cursorPrefixNext
}

// isPrev returns true if the prefix is prev.
func (p cursorPrefix) isPrev() bool {
	return p == cursorPrefixPrev
}

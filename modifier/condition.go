package modifier

import (
	"strings"
)

// SQLCondition represents a SQL condition with parenthesization support
type SQLCondition struct {
	Expression string
	IsNested   bool // Whether the condition is already wrapped in parentheses
}

// NewCondition creates a simple SQL condition
func NewCondition(expr string) SQLCondition {
	return SQLCondition{
		Expression: expr,
		IsNested:   false,
	}
}

// NewNestedCondition creates a nested condition by joining multiple conditions with an operator
func NewNestedCondition(operator string, conditions ...SQLCondition) SQLCondition {

	if len(conditions) == 0 {
		return SQLCondition{}
	}

	operator = strings.ToUpper(operator)
	if operator != "AND" && operator != "OR" {
		return SQLCondition{}
	}

	var result strings.Builder

	result.WriteString("(")
	for i, condition := range conditions {
		if i > 0 {
			result.WriteString(" ")
			result.WriteString(operator)
			result.WriteString(" ")
		}
		result.WriteString(condition.Expression)
	}
	result.WriteString(")")

	return SQLCondition{
		Expression: result.String(),
		IsNested:   true,
	}
}

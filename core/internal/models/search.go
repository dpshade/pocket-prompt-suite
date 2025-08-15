package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BooleanExpression represents a boolean search expression for tags
type BooleanExpression struct {
	Type  ExpressionType   `json:"type"`
	Value interface{}      `json:"value"` // string for Tag, []*BooleanExpression for operators
}

// ExpressionType defines the type of boolean expression
type ExpressionType string

const (
	ExpressionTag ExpressionType = "tag"
	ExpressionAnd ExpressionType = "and"
	ExpressionOr  ExpressionType = "or"
	ExpressionXor ExpressionType = "xor"
	ExpressionNot ExpressionType = "not"
)

// SavedSearch represents a named boolean search that can be reused
type SavedSearch struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Expression  *BooleanExpression `json:"expression"`
	TextQuery   string             `json:"text_query,omitempty"` // Optional text search filter
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

// Evaluate evaluates the boolean expression against a prompt's tags
func (be *BooleanExpression) Evaluate(tags []string) bool {
	if be == nil {
		return true
	}

	switch be.Type {
	case ExpressionTag:
		tagName, ok := be.Value.(string)
		if !ok {
			return false
		}
		return containsTag(tags, tagName)

	case ExpressionAnd:
		expressions, ok := be.Value.([]*BooleanExpression)
		if !ok || len(expressions) == 0 {
			return true
		}
		for _, expr := range expressions {
			if !expr.Evaluate(tags) {
				return false
			}
		}
		return true

	case ExpressionOr:
		expressions, ok := be.Value.([]*BooleanExpression)
		if !ok || len(expressions) == 0 {
			return false
		}
		for _, expr := range expressions {
			if expr.Evaluate(tags) {
				return true
			}
		}
		return false

	case ExpressionXor:
		expressions, ok := be.Value.([]*BooleanExpression)
		if !ok || len(expressions) != 2 {
			return false
		}
		left := expressions[0].Evaluate(tags)
		right := expressions[1].Evaluate(tags)
		return (left && !right) || (!left && right)

	case ExpressionNot:
		expressions, ok := be.Value.([]*BooleanExpression)
		if !ok || len(expressions) != 1 {
			return false
		}
		return !expressions[0].Evaluate(tags)

	default:
		return false
	}
}

// QueryString returns the expression as an editable query string (without brackets for tags)
func (be *BooleanExpression) QueryString() string {
	if be == nil {
		return ""
	}

	switch be.Type {
	case ExpressionTag:
		if tagName, ok := be.Value.(string); ok {
			return tagName // No brackets for query format
		}
		return "unknown"

	case ExpressionAnd:
		if expressions, ok := be.Value.([]*BooleanExpression); ok {
			var parts []string
			for _, expr := range expressions {
				parts = append(parts, expr.QueryString())
			}
			return strings.Join(parts, " AND ")
		}
		return "AND ?"

	case ExpressionOr:
		if expressions, ok := be.Value.([]*BooleanExpression); ok {
			var parts []string
			for _, expr := range expressions {
				parts = append(parts, expr.QueryString())
			}
			return strings.Join(parts, " OR ")
		}
		return "OR ?"

	case ExpressionXor:
		if expressions, ok := be.Value.([]*BooleanExpression); ok && len(expressions) == 2 {
			return fmt.Sprintf("%s XOR %s", expressions[0].QueryString(), expressions[1].QueryString())
		}
		return "XOR ?"

	case ExpressionNot:
		if expressions, ok := be.Value.([]*BooleanExpression); ok && len(expressions) == 1 {
			return fmt.Sprintf("NOT %s", expressions[0].QueryString())
		}
		return "NOT ?"

	default:
		return "?"
	}
}

// String returns a human-readable string representation of the expression
func (be *BooleanExpression) String() string {
	if be == nil {
		return ""
	}

	switch be.Type {
	case ExpressionTag:
		if tagName, ok := be.Value.(string); ok {
			return fmt.Sprintf("[%s]", tagName)
		}
		return "[unknown]"

	case ExpressionAnd:
		if expressions, ok := be.Value.([]*BooleanExpression); ok {
			var parts []string
			for _, expr := range expressions {
				parts = append(parts, expr.String())
			}
			return fmt.Sprintf("(%s)", strings.Join(parts, " AND "))
		}
		return "(AND ?)"

	case ExpressionOr:
		if expressions, ok := be.Value.([]*BooleanExpression); ok {
			var parts []string
			for _, expr := range expressions {
				parts = append(parts, expr.String())
			}
			return fmt.Sprintf("(%s)", strings.Join(parts, " OR "))
		}
		return "(OR ?)"

	case ExpressionXor:
		if expressions, ok := be.Value.([]*BooleanExpression); ok && len(expressions) == 2 {
			return fmt.Sprintf("(%s XOR %s)", expressions[0].String(), expressions[1].String())
		}
		return "(XOR ?)"

	case ExpressionNot:
		if expressions, ok := be.Value.([]*BooleanExpression); ok && len(expressions) == 1 {
			return fmt.Sprintf("NOT %s", expressions[0].String())
		}
		return "NOT ?"

	default:
		return "(?)"
	}
}

// containsTag checks if a tag is present in the tags slice (case-insensitive)
func containsTag(tags []string, target string) bool {
	targetLower := strings.ToLower(target)
	for _, tag := range tags {
		if strings.ToLower(tag) == targetLower {
			return true
		}
	}
	return false
}

// NewTagExpression creates a new tag expression
func NewTagExpression(tag string) *BooleanExpression {
	return &BooleanExpression{
		Type:  ExpressionTag,
		Value: tag,
	}
}

// NewAndExpression creates a new AND expression
func NewAndExpression(expressions ...*BooleanExpression) *BooleanExpression {
	return &BooleanExpression{
		Type:  ExpressionAnd,
		Value: expressions,
	}
}

// NewOrExpression creates a new OR expression
func NewOrExpression(expressions ...*BooleanExpression) *BooleanExpression {
	return &BooleanExpression{
		Type:  ExpressionOr,
		Value: expressions,
	}
}

// NewXorExpression creates a new XOR expression
func NewXorExpression(left, right *BooleanExpression) *BooleanExpression {
	return &BooleanExpression{
		Type:  ExpressionXor,
		Value: []*BooleanExpression{left, right},
	}
}

// NewNotExpression creates a new NOT expression
func NewNotExpression(expr *BooleanExpression) *BooleanExpression {
	return &BooleanExpression{
		Type:  ExpressionNot,
		Value: []*BooleanExpression{expr},
	}
}

// MarshalJSON implements custom JSON marshaling for BooleanExpression
func (be *BooleanExpression) MarshalJSON() ([]byte, error) {
	switch be.Type {
	case ExpressionTag:
		return json.Marshal(struct {
			Type  ExpressionType `json:"type"`
			Value string         `json:"value"`
		}{
			Type:  be.Type,
			Value: be.Value.(string),
		})
	default:
		return json.Marshal(struct {
			Type  ExpressionType        `json:"type"`
			Value []*BooleanExpression  `json:"value"`
		}{
			Type:  be.Type,
			Value: be.Value.([]*BooleanExpression),
		})
	}
}

// UnmarshalJSON implements custom JSON unmarshaling for BooleanExpression
func (be *BooleanExpression) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to check the type
	var temp struct {
		Type  ExpressionType  `json:"type"`
		Value json.RawMessage `json:"value"`
	}
	
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	
	be.Type = temp.Type
	
	switch temp.Type {
	case ExpressionTag:
		var tagValue string
		if err := json.Unmarshal(temp.Value, &tagValue); err != nil {
			return err
		}
		be.Value = tagValue
	default:
		var exprValues []*BooleanExpression
		if err := json.Unmarshal(temp.Value, &exprValues); err != nil {
			return err
		}
		be.Value = exprValues
	}
	
	return nil
}
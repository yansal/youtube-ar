package log

import (
	"fmt"
	"strconv"
)

// A Field is a log field.
type Field struct {
	key, value string
}

// Int returns a field where value is an int.
func Int(key string, value int) Field {
	return Field{key: key, value: strconv.Itoa(value)}
}

// String returns a field where value is a string.
func String(key string, value string) Field {
	return Field{key: key, value: value}
}

// Stringer returns a field where value implements fmt.Stringer.
func Stringer(key string, value fmt.Stringer) Field {
	return Field{key: key, value: value.String()}
}

// Error returns a field where value is an error.
func Error(key string, value error) Field {
	return Field{key: key, value: value.Error()}
}

// Raw returns a field where value is an interface{}.
func Raw(key string, value interface{}) Field {
	return Field{key: key, value: fmt.Sprintf("%+v", value)}
}

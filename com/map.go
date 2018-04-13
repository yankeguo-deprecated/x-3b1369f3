package com

// Map alias to map[string]interface{}
type Map map[string]interface{}

// NewMap create a map[string]interface{} easily
// `com.NewMap("hello", 1, "world", 2)` will create `map[string]interface{} { "hello": 1, "world": 2 }`
// will panic if keys are not string
func NewMap(args ...interface{}) Map {
	if len(args)%2 != 0 {
		return Map{}
	}
	m := make(Map, len(args)/2)
	for i := 0; i < len(args); i = i + 2 {
		m[args[i].(string)] = args[i+1]
	}
	return m
}

// Set set a value and returns self, suitable for chaining
func (m Map) Set(key string, value interface{}) Map {
	m[key] = value
	return m
}

package dispel

import "testing"

func TestTemplateCompiles(t *testing.T) {
	_, err := NewTemplate()
	ok(t, err)
}

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTemplateFuncs(t *testing.T) {
	unit := newTemplateFuncs()
	assertAbsent := []string{"env", "expandenv", "getHostByName"}
	assertPresent := []string{"date", "uuidv4", "hello", "ago", "abbrev", "atoi", "split", "until", "join"}

	for _, needle := range assertAbsent {
		_, ok := unit[needle]
		assert.Falsef(t, ok, "Template functions should not contain %q", needle)
	}

	for _, needle := range assertPresent {
		_, ok := unit[needle]
		assert.Truef(t, ok, "Template functions should contain %q", needle)
	}
}

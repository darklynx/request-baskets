package main

import (
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToThemeCss_Default(t *testing.T) {
	assert.Equal(t, template.HTML(themeDefaultCSS), toThemeCss(ThemeDefault))
}

func TestToThemeCss_Adaptive(t *testing.T) {
	assert.Equal(t, template.HTML(themeAdaptiveCSS), toThemeCss(ThemeAdaptive))
}

func TestToThemeCss_Flatly(t *testing.T) {
	assert.Equal(t, template.HTML(themeFlatlyCSS), toThemeCss(ThemeFlatly))
}

func TestToThemeCss_Unknown(t *testing.T) {
	assert.Equal(t, template.HTML(themeDefaultCSS), toThemeCss("xyz"))
}

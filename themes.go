package main

import "html/template"

const (
	ThemeStandard    = "standard"
	themeStandardCSS = `
  <link rel="stylesheet" href="/static/css/bootstrap-3.3.7.min.css">
  <link rel="stylesheet" href="/static/css/bootstrap-theme-3.3.7.min.css">`
	ThemeAdaptive    = "adaptive"
	themeAdaptiveCSS = `
  <link rel="stylesheet" href="/static/css/bootstrap-dark-1.0.0.min.css">`
	ThemeFlatly    = "flatly"
	themeFlatlyCSS = `
  <link rel="stylesheet" href="/static/css/bootswatch-flatly-3.3.7.min.css">`
)

func toThemeCSS(theme string) template.HTML {
	switch theme {
	case ThemeAdaptive:
		return themeAdaptiveCSS
	case ThemeFlatly:
		return themeFlatlyCSS
	case ThemeStandard:
		fallthrough
	default:
		return themeStandardCSS
	}
}

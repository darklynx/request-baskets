package main

import "html/template"

const (
	ThemeStandard    = "standard"
	themeStandardCSS = `
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css" integrity="sha384-rHyoN1iRsVXV4nD0JutlnGaslCJuC7uwjduW9SVrLvRYooPp2bWYgmgJQIXwl/Sp" crossorigin="anonymous">`
	ThemeAdaptive    = "adaptive"
	themeAdaptiveCSS = `
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/darklynx/requestbaskets-dark-theme@1.0.0/bootstrap.min.css" integrity="sha384-QiFF09wqK5z3/usps1yc+Om75gf8byvdtluQfS0enYGx1nmji2dEbtgRDw1Tw60j" crossorigin="anonymous">`
	ThemeFlatly    = "flatly"
	themeFlatlyCSS = `
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css" integrity="sha384-rHyoN1iRsVXV4nD0JutlnGaslCJuC7uwjduW9SVrLvRYooPp2bWYgmgJQIXwl/Sp" crossorigin="anonymous">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootswatch/3.3.7/flatly/bootstrap.min.css">`
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

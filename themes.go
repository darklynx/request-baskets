package main

import "html/template"

const (
	ThemeDefault    = "default"
	themeDefaultCSS = `
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css" integrity="sha384-rHyoN1iRsVXV4nD0JutlnGaslCJuC7uwjduW9SVrLvRYooPp2bWYgmgJQIXwl/Sp" crossorigin="anonymous">`
	ThemeAdaptive    = "adaptive"
	themeAdaptiveCSS = `
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/socuul/requestbaskets-dark-theme@09176b7/bootstrap.min.css" integrity="sha384-37zMSuO/NCFq9o7XMDpmgGqoLfcr/RZ8YAvI9m2OX+AfMbwaV9HWto9wg9SIwHtc" crossorigin="anonymous">`
	ThemeFlatly    = "flatly"
	themeFlatlyCSS = `
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css" integrity="sha384-rHyoN1iRsVXV4nD0JutlnGaslCJuC7uwjduW9SVrLvRYooPp2bWYgmgJQIXwl/Sp" crossorigin="anonymous">
  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootswatch/3.3.7/flatly/bootstrap.min.css">`
)

func toThemeCss(theme string) template.HTML {
	switch theme {
	case ThemeAdaptive:
		return themeAdaptiveCSS
	case ThemeFlatly:
		return themeFlatlyCSS
	case ThemeDefault:
		fallthrough
	default:
		return themeDefaultCSS
	}
}

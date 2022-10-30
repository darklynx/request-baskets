package main

import (
	"html/template"

	"github.com/Masterminds/sprig/v3"
)

func newTemplateFuncs() template.FuncMap {
	funcs := sprig.FuncMap()

	delete(funcs, "env")
	delete(funcs, "expandenv")
	delete(funcs, "getHostByName")

	return funcs
}

func newTemplate(name string) *template.Template {
	return template.New(name).Funcs(newTemplateFuncs())
}

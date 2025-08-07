package main

import (
	"html/template"
	"path/filepath"
)

// LoadTemplates parses HTML templates for integrations
func LoadTemplates() *template.Template {
	tmpl := template.New("")
	files, err := filepath.Glob("integrations/templates/*.html")
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		tmpl = template.Must(tmpl.ParseFiles(f))
	}
	return tmpl
}


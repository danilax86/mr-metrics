// SPDX-FileCopyrightText: 2025 Danila Gorelko <hello@danilax86.space>
//
// SPDX-License-Identifier: MIT

package web

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed templates/*.gohtml
var fs embed.FS

func templateFrom(funcMap template.FuncMap, filenames ...string) *template.Template {
	// NOTE(danilax86): head.gohtml is the default template that will be used in every ever made template.
	filenames = append(filenames, "head")
	for i, filename := range filenames {
		filenames[i] = "templates/" + filename + ".gohtml"
	}
	return template.Must(template.New("head.gohtml").Funcs(funcMap).ParseFS(fs, filenames...))
}

func TemplateExec(w http.ResponseWriter, t *template.Template, data any) error {
	return t.ExecuteTemplate(w, "head.gohtml", data)
}

func TemplateStats() *template.Template {
	return templateFrom(template.FuncMap{"sum": mapSumFunc}, "stats")
}

func mapSumFunc(m map[string]int) int {
	var sum int
	for _, v := range m {
		sum += v
	}
	return sum
}

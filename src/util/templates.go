package util

import (
	"bytes"
	"text/template"
)

// Helper function to parse, execute and convert "text/template" to string. Panics on error.
func TemplateToString(format string, data interface{}) string {
	bb := &bytes.Buffer{}

	err := template.Must(template.New("").Parse(format)).Execute(bb, data)
	if err != nil {
		panic(err)
	}

	return bb.String()
}

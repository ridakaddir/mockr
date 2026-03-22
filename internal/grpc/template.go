package grpc

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"github.com/google/uuid"
)

// renderGRPCTemplate processes {{uuid}}, {{now}}, {{timestamp}} tokens in a JSON string.
// Mirrors proxy.renderTemplate without the HTTP dependency.
func renderGRPCTemplate(s string) (string, error) {
	funcMap := template.FuncMap{
		"uuid": func() string {
			return uuid.New().String()
		},
		"now": func() string {
			return time.Now().UTC().Format(time.RFC3339)
		},
		"timestamp": func() string {
			return fmt.Sprintf("%d", time.Now().UnixMilli())
		},
	}

	tmpl, err := template.New("grpc-mock").Funcs(funcMap).Parse(s)
	if err != nil {
		return s, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return s, err
	}

	return buf.String(), nil
}

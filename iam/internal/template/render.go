package template

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

func Render(content string, values any) ([]string, error) {
	var buffer bytes.Buffer
	template, err := template.New("test").Parse(content)
	if err != nil {
		return nil, fmt.Errorf("can't create template: %w", err)
	}

	err = template.Execute(&buffer, values)
	if err != nil {
		return nil, fmt.Errorf("can't render template: %w", err)
	}

	return strings.Split(buffer.String(), "\n"), err
}

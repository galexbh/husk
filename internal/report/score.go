package report

import (
	"io"

	"github.com/galexbh/husk/internal/model"
)

// RenderScoreTable escribe el score de resiliencia como tablas ASCII.
func RenderScoreTable(w io.Writer, s *model.Score) error {
	for _, t := range scoreTables(s) {
		if _, err := io.WriteString(w, t.Render()+"\n\n"); err != nil {
			return err
		}
	}
	return nil
}

// RenderScoreMarkdown escribe el score de resiliencia como tablas
// Markdown.
func RenderScoreMarkdown(w io.Writer, s *model.Score) error {
	for _, t := range scoreTables(s) {
		if _, err := io.WriteString(w, t.RenderMarkdown()+"\n\n"); err != nil {
			return err
		}
	}
	return nil
}

// RenderScoreJSON escribe el Score completo como JSON indentado.
func RenderScoreJSON(w io.Writer, s *model.Score) error {
	return writeJSON(w, s)
}

// RenderScore escribe el score de resiliencia en el formato de texto
// pedido.
func RenderScore(w io.Writer, s *model.Score, format string) error {
	switch format {
	case "", "table":
		return RenderScoreTable(w, s)
	case "markdown":
		return RenderScoreMarkdown(w, s)
	case "json":
		return RenderScoreJSON(w, s)
	default:
		return unsupportedFormat(format)
	}
}

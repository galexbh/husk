package excel

import (
	"github.com/xuri/excelize/v2"

	"github.com/galexbh/husk/internal/model"
)

// riskFillColors mapea cada nivel de riesgo al color de fondo usado para
// resaltar la fila en el workbook (rojo: sin límites/una réplica; amarillo:
// sobreaprovisionado —reservado para cuando el sizing de la Fase 2 alimente
// el modelo—; verde: saludable).
var riskFillColors = map[model.RiskLevel]string{
	model.RiskRed:    "FFC7CE",
	model.RiskYellow: "FFEB9C",
	model.RiskGreen:  "C6EFCE",
}

// styles agrupa los IDs de estilo de un *excelize.File, creados una sola
// vez y reutilizados en todas las hojas.
type styles struct {
	header int
	risk   map[model.RiskLevel]int
}

func newStyles(f *excelize.File) (*styles, error) {
	header, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"305496"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
	})
	if err != nil {
		return nil, err
	}

	riskStyles := make(map[model.RiskLevel]int, len(riskFillColors))
	for level, color := range riskFillColors {
		id, err := f.NewStyle(&excelize.Style{
			Fill: excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1},
		})
		if err != nil {
			return nil, err
		}
		riskStyles[level] = id
	}

	return &styles{header: header, risk: riskStyles}, nil
}

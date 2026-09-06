// Package excel construye workbooks .xlsx formateados (encabezados en
// negrita, autofiltro, columnas congeladas, colores condicionales de
// riesgo e hipervínculos entre hojas), compartido por internal/inventory y,
// en fases posteriores, por internal/report para report generate.
package excel

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/galexbh/husk/internal/model"
)

const summarySheetName = "Summary"

// Sheet describe una hoja del workbook: encabezados, filas (ya formateadas
// como texto) y, opcionalmente, el nivel de riesgo de cada fila para el
// resaltado condicional.
type Sheet struct {
	Name     string
	Headers  []string
	Rows     [][]string
	RowRisks []model.RiskLevel // paralelo a Rows; nil si la hoja no clasifica riesgo
}

// SummaryEntry es una fila de la hoja "Summary": una etiqueta, un conteo y
// un hipervínculo a la hoja de detalle correspondiente.
type SummaryEntry struct {
	Label     string
	Count     int
	SheetLink string // nombre de hoja al que enlaza; vacío si no aplica
}

// Workbook construye un .xlsx incrementalmente: una hoja "Summary" (creada
// vacía desde el inicio, para que quede primera) más una hoja por tipo de
// recurso añadida vía AddSheet, y finalmente poblada con FinalizeSummary.
type Workbook struct {
	f              *excelize.File
	styles         *styles
	freezeColumns  int
	highlightRisks bool
}

// New crea un workbook vacío. freezeColumns es cuántas columnas quedan
// congeladas junto con la fila de encabezados; highlightRisks habilita el
// coloreado condicional por riesgo.
func New(freezeColumns int, highlightRisks bool) (*Workbook, error) {
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", summarySheetName); err != nil {
		return nil, err
	}

	st, err := newStyles(f)
	if err != nil {
		return nil, err
	}

	return &Workbook{f: f, styles: st, freezeColumns: freezeColumns, highlightRisks: highlightRisks}, nil
}

// AddSheet agrega una hoja de detalle con encabezados en negrita, autofiltro,
// columnas congeladas, anchos ajustados al contenido y, si highlightRisks
// está habilitado, el color de fondo correspondiente al riesgo de cada fila.
func (w *Workbook) AddSheet(sheet Sheet) error {
	name := sheet.Name
	if _, err := w.f.NewSheet(name); err != nil {
		return fmt.Errorf("creando hoja %s: %w", name, err)
	}

	if len(sheet.Headers) == 0 {
		return nil
	}

	for i, h := range sheet.Headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return err
		}
		if err := w.f.SetCellValue(name, cell, h); err != nil {
			return err
		}
	}
	lastHeaderCell, err := excelize.CoordinatesToCellName(len(sheet.Headers), 1)
	if err != nil {
		return err
	}
	if err := w.f.SetCellStyle(name, "A1", lastHeaderCell, w.styles.header); err != nil {
		return err
	}

	for r, row := range sheet.Rows {
		rowIdx := r + 2
		for c, val := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, rowIdx)
			if err != nil {
				return err
			}
			if err := w.f.SetCellValue(name, cell, val); err != nil {
				return err
			}
		}

		if w.highlightRisks && r < len(sheet.RowRisks) {
			if styleID, ok := w.styles.risk[sheet.RowRisks[r]]; ok {
				first, err := excelize.CoordinatesToCellName(1, rowIdx)
				if err != nil {
					return err
				}
				last, err := excelize.CoordinatesToCellName(len(sheet.Headers), rowIdx)
				if err != nil {
					return err
				}
				if err := w.f.SetCellStyle(name, first, last, styleID); err != nil {
					return err
				}
			}
		}
	}

	lastRow := len(sheet.Rows) + 1
	lastDataCell, err := excelize.CoordinatesToCellName(len(sheet.Headers), lastRow)
	if err != nil {
		return err
	}
	if err := w.f.AutoFilter(name, fmt.Sprintf("A1:%s", lastDataCell), nil); err != nil {
		return err
	}

	freezeCols := w.freezeColumns
	if freezeCols > len(sheet.Headers) {
		freezeCols = len(sheet.Headers)
	}
	if freezeCols < 0 {
		freezeCols = 0
	}
	if err := w.f.SetPanes(name, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      freezeCols,
		YSplit:      1,
		TopLeftCell: mustCellName(freezeCols+1, 2),
		ActivePane:  "bottomRight",
	}); err != nil {
		return err
	}

	return w.autoSizeColumns(name, sheet.Headers, sheet.Rows)
}

func (w *Workbook) autoSizeColumns(sheet string, headers []string, rows [][]string) error {
	for i, h := range headers {
		maxLen := len(h)
		for _, row := range rows {
			if i < len(row) && len(row[i]) > maxLen {
				maxLen = len(row[i])
			}
		}
		width := float64(maxLen) + 2
		if width < 10 {
			width = 10
		}
		if width > 60 {
			width = 60
		}
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		if err := w.f.SetColWidth(sheet, col, col, width); err != nil {
			return err
		}
	}
	return nil
}

// FinalizeSummary rellena la hoja "Summary" (creada vacía por New) con las
// entradas dadas, cada una con un hipervínculo a su hoja de detalle, y deja
// esa hoja como la activa al abrir el archivo.
func (w *Workbook) FinalizeSummary(title string, entries []SummaryEntry) error {
	if err := w.f.SetCellValue(summarySheetName, "A1", title); err != nil {
		return err
	}
	if err := w.f.SetCellStyle(summarySheetName, "A1", "A1", w.styles.header); err != nil {
		return err
	}

	if err := w.f.SetCellValue(summarySheetName, "A3", "Recurso"); err != nil {
		return err
	}
	if err := w.f.SetCellValue(summarySheetName, "B3", "Cantidad"); err != nil {
		return err
	}
	if err := w.f.SetCellStyle(summarySheetName, "A3", "B3", w.styles.header); err != nil {
		return err
	}

	for i, e := range entries {
		row := i + 4
		labelCell := mustCellName(1, row)
		countCell := mustCellName(2, row)
		if err := w.f.SetCellValue(summarySheetName, labelCell, e.Label); err != nil {
			return err
		}
		if err := w.f.SetCellValue(summarySheetName, countCell, e.Count); err != nil {
			return err
		}
		if e.SheetLink != "" {
			if err := w.f.SetCellHyperLink(summarySheetName, labelCell, fmt.Sprintf("%s!A1", e.SheetLink), "Location"); err != nil {
				return err
			}
		}
	}

	if err := w.f.SetColWidth(summarySheetName, "A", "A", 40); err != nil {
		return err
	}
	if err := w.f.SetColWidth(summarySheetName, "B", "B", 15); err != nil {
		return err
	}

	idx, err := w.f.GetSheetIndex(summarySheetName)
	if err != nil {
		return err
	}
	w.f.SetActiveSheet(idx)
	return nil
}

// SaveAs escribe el workbook en path.
func (w *Workbook) SaveAs(path string) error {
	return w.f.SaveAs(path)
}

func mustCellName(col, row int) string {
	name, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		// Solo puede fallar con coordenadas fuera de rango, que nunca
		// generamos aquí (col/row siempre >= 1 y acotados por datos reales).
		panic(err)
	}
	return name
}

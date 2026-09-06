package excel

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/galexbh/husk/internal/model"
)

func TestWorkbook_AddSheetAndSummary_ProducesValidXLSX(t *testing.T) {
	wb, err := New(2, true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	sheet := Sheet{
		Name:     "Deployments",
		Headers:  []string{"Namespace", "Nombre", "Riesgo"},
		Rows:     [][]string{{"shop", "api", "red"}, {"shop", "worker", "green"}},
		RowRisks: []model.RiskLevel{model.RiskRed, model.RiskGreen},
	}
	if err := wb.AddSheet(sheet); err != nil {
		t.Fatalf("AddSheet: %v", err)
	}

	if err := wb.FinalizeSummary("Inventario de prueba", []SummaryEntry{
		{Label: "Deployments", Count: 2, SheetLink: "Deployments"},
		{Label: "Services", Count: 0},
	}); err != nil {
		t.Fatalf("FinalizeSummary: %v", err)
	}

	path := filepath.Join(t.TempDir(), "test.xlsx")
	if err := wb.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("no se pudo reabrir el .xlsx generado: %v", err)
	}
	defer f.Close()

	sheetNames := f.GetSheetList()
	if len(sheetNames) != 2 {
		t.Fatalf("hojas = %v, want 2 (Summary, Deployments)", sheetNames)
	}
	if sheetNames[0] != "Summary" {
		t.Errorf("primera hoja = %q, want Summary (debe quedar primera y activa)", sheetNames[0])
	}

	// Encabezados de la hoja de detalle.
	for col, want := range map[string]string{"A1": "Namespace", "B1": "Nombre", "C1": "Riesgo"} {
		got, err := f.GetCellValue("Deployments", col)
		if err != nil {
			t.Fatalf("GetCellValue(%s): %v", col, err)
		}
		if got != want {
			t.Errorf("Deployments!%s = %q, want %q", col, got, want)
		}
	}

	// Filas de datos.
	got, _ := f.GetCellValue("Deployments", "B2")
	if got != "api" {
		t.Errorf("Deployments!B2 = %q, want api", got)
	}

	// Panes congelados: 2 columnas + fila de encabezados.
	panes, err := f.GetPanes("Deployments")
	if err != nil {
		t.Fatalf("GetPanes: %v", err)
	}
	if !panes.Freeze || panes.XSplit != 2 || panes.YSplit != 1 {
		t.Errorf("se esperaban panes congelados (2 columnas + 1 fila) en Deployments, got %+v", panes)
	}

	// Hipervínculo de Summary hacia Deployments.
	link, target, err := f.GetCellHyperLink("Summary", "A4")
	if err != nil {
		t.Fatalf("GetCellHyperLink: %v", err)
	}
	if !link || target != "Deployments!A1" {
		t.Errorf("Summary!A4 hyperlink = (%v, %q), want (true, \"Deployments!A1\")", link, target)
	}
}

func TestWorkbook_SkipsEmptySheets(t *testing.T) {
	wb, err := New(1, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := wb.FinalizeSummary("Vacío", []SummaryEntry{{Label: "Services", Count: 0}}); err != nil {
		t.Fatalf("FinalizeSummary: %v", err)
	}

	path := filepath.Join(t.TempDir(), "empty.xlsx")
	if err := wb.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	defer f.Close()

	if got := f.GetSheetList(); len(got) != 1 {
		t.Errorf("hojas = %v, want solo Summary (ninguna hoja de detalle creada para un conteo de 0)", got)
	}
}

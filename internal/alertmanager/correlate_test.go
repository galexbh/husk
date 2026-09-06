package alertmanager

import (
	"testing"

	"github.com/galexbh/husk/internal/model"
)

func TestCorrelate_ActiveAlertMarksIncident(t *testing.T) {
	findings := []model.Finding{
		model.NewFinding(model.RiskRed, "sizing-under-provisioned", "shop", "shop/api/api", "subaprovisionado"),
		model.NewFinding(model.RiskYellow, "missing-pdb", "shop", "Deployment/api", "sin pdb"), // categoría no correlacionable
	}
	alerts := []model.Alert{
		{Name: "KubeContainerOOMKilled", Namespace: "shop", State: "active"},
	}

	correlated := Correlate(findings, alerts)
	if len(correlated) != 1 || correlated[0] != findings[0].ID {
		t.Errorf("Correlate = %v, want solo %s", correlated, findings[0].ID)
	}
}

func TestCorrelate_NoActiveAlerts(t *testing.T) {
	findings := []model.Finding{
		model.NewFinding(model.RiskRed, "sizing-under-provisioned", "shop", "shop/api/api", "subaprovisionado"),
	}
	if got := Correlate(findings, nil); len(got) != 0 {
		t.Errorf("Correlate sin alertas = %v, want vacío", got)
	}
}

func TestCorrelate_IgnoresSuppressedAndUnrelatedAlerts(t *testing.T) {
	findings := []model.Finding{
		model.NewFinding(model.RiskRed, "sizing-under-provisioned", "shop", "shop/api/api", "subaprovisionado"),
	}
	alerts := []model.Alert{
		{Name: "KubeContainerOOMKilled", Namespace: "shop", State: "suppressed"},
		{Name: "CertificateExpiringSoon", Namespace: "shop", State: "active"},
	}
	if got := Correlate(findings, alerts); len(got) != 0 {
		t.Errorf("Correlate = %v, want vacío (alerta suprimida o no relacionada)", got)
	}
}

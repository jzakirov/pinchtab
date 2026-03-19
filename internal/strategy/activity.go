package strategy

import (
	"net/http"

	"github.com/pinchtab/pinchtab/internal/activity"
	"github.com/pinchtab/pinchtab/internal/orchestrator"
	"github.com/pinchtab/pinchtab/internal/tenant"
)

func EnrichForTarget(r *http.Request, orch *orchestrator.Orchestrator, target string) {
	if r == nil || orch == nil || target == "" {
		return
	}

	tenantID := tenant.TenantFromContext(r.Context())
	for _, inst := range orch.List() {
		if inst.Status != "running" || inst.URL != target {
			continue
		}
		if tenantID != "" && !tenant.HasTenantPrefix(inst.ProfileName, tenantID) {
			continue
		}
		activity.EnrichRequest(r, activity.Update{
			InstanceID:  inst.ID,
			ProfileID:   inst.ProfileID,
			ProfileName: inst.ProfileName,
		})
		return
	}
}

func TargetForRequest(r *http.Request, orch *orchestrator.Orchestrator) string {
	if orch == nil {
		return ""
	}
	return orch.FirstRunningURLForTenant(tenant.TenantFromContext(r.Context()))
}

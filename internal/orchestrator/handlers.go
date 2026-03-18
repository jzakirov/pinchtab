package orchestrator

import (
	"fmt"
	"net/http"

	"github.com/pinchtab/pinchtab/internal/bridge"
	"github.com/pinchtab/pinchtab/internal/tenant"
	"github.com/pinchtab/pinchtab/internal/web"
)

func registerCapabilityRoute(mux *http.ServeMux, route string, enabled bool, feature, setting, code string, next http.HandlerFunc) {
	if enabled {
		mux.HandleFunc(route, next)
		return
	}
	mux.HandleFunc(route, web.DisabledEndpointHandler(feature, setting, code))
}

func (o *Orchestrator) RegisterHandlers(mux *http.ServeMux) {
	// Profile management
	mux.HandleFunc("POST /profiles/{id}/start", o.handleStartByID)
	mux.HandleFunc("POST /profiles/{id}/stop", o.handleStopByID)
	mux.HandleFunc("GET /profiles/{id}/instance", o.handleProfileInstance)

	// Instance management
	mux.HandleFunc("GET /instances", o.handleList)
	mux.HandleFunc("GET /instances/{id}", o.handleGetInstance)
	mux.HandleFunc("GET /instances/tabs", o.handleAllTabs)
	mux.HandleFunc("GET /instances/metrics", o.handleAllMetrics)
	mux.HandleFunc("POST /instances/start", o.handleStartInstance)
	mux.HandleFunc("POST /instances/launch", o.handleLaunchByName)
	mux.HandleFunc("POST /instances/attach", o.handleAttachInstance)
	mux.HandleFunc("POST /instances/attach-bridge", o.handleAttachBridge)
	mux.HandleFunc("POST /instances/{id}/start", o.handleStartByInstanceID)
	mux.HandleFunc("POST /instances/{id}/stop", o.handleStopByInstanceID)
	mux.HandleFunc("GET /instances/{id}/logs", o.handleLogsByID)
	mux.HandleFunc("GET /instances/{id}/logs/stream", o.handleLogsStreamByID)
	mux.HandleFunc("GET /instances/{id}/tabs", o.handleInstanceTabs)
	mux.HandleFunc("POST /instances/{id}/tabs/open", o.handleInstanceTabOpen)
	mux.HandleFunc("POST /instances/{id}/tab", o.proxyToInstance)
	registerCapabilityRoute(mux, "GET /instances/{id}/proxy/screencast", o.AllowsScreencast(), "screencast", "security.allowScreencast", "screencast_disabled", o.handleProxyScreencast)
	registerCapabilityRoute(mux, "GET /instances/{id}/screencast", o.AllowsScreencast(), "screencast", "security.allowScreencast", "screencast_disabled", o.proxyToInstance)

	// Tab operations - custom handlers
	mux.HandleFunc("POST /tabs/{id}/close", o.handleTabClose)

	// Tab operations - generic proxy (all route to the appropriate instance)
	for _, route := range []string{
		"POST /tabs/{id}/navigate",
		"GET /tabs/{id}/snapshot",
		"GET /tabs/{id}/screenshot",
		"POST /tabs/{id}/action",
		"POST /tabs/{id}/actions",
		"GET /tabs/{id}/text",
		"GET /tabs/{id}/pdf",
		"POST /tabs/{id}/pdf",
		"POST /tabs/{id}/lock",
		"POST /tabs/{id}/unlock",
		"GET /tabs/{id}/cookies",
		"POST /tabs/{id}/cookies",
		"GET /tabs/{id}/metrics",
		"POST /tabs/{id}/find",
		"POST /tabs/{id}/back",
		"POST /tabs/{id}/forward",
		"POST /tabs/{id}/reload",
	} {
		mux.HandleFunc(route, o.proxyTabRequest)
	}
	registerCapabilityRoute(mux, "POST /tabs/{id}/evaluate", o.AllowsEvaluate(), "evaluate", "security.allowEvaluate", "evaluate_disabled", o.proxyTabRequest)
	registerCapabilityRoute(mux, "GET /tabs/{id}/download", o.AllowsDownload(), "download", "security.allowDownload", "download_disabled", o.proxyTabRequest)
	registerCapabilityRoute(mux, "POST /tabs/{id}/upload", o.AllowsUpload(), "upload", "security.allowUpload", "upload_disabled", o.proxyTabRequest)
}

func (o *Orchestrator) handleList(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.TenantFromContext(r.Context())
	instances := o.List()
	if tenantID != "" {
		filtered := make([]bridge.Instance, 0, len(instances))
		for _, inst := range instances {
			if tenant.HasTenantPrefix(inst.ProfileName, tenantID) {
				inst.ProfileName = tenant.StripTenantPrefix(inst.ProfileName, tenantID)
				filtered = append(filtered, inst)
			}
		}
		instances = filtered
	}
	web.JSON(w, 200, instances)
}

func (o *Orchestrator) handleAllTabs(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.TenantFromContext(r.Context())
	tabs := o.AllTabs()
	if tenantID != "" {
		owned := o.tenantInstanceIDs(tenantID)
		filtered := make([]bridge.InstanceTab, 0, len(tabs))
		for _, tab := range tabs {
			if owned[tab.InstanceID] {
				filtered = append(filtered, tab)
			}
		}
		tabs = filtered
	}
	web.JSON(w, 200, tabs)
}

func (o *Orchestrator) handleAllMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.TenantFromContext(r.Context())
	metrics := o.AllMetrics()
	if tenantID != "" {
		owned := o.tenantInstanceIDs(tenantID)
		filtered := metrics[:0]
		for _, m := range metrics {
			if owned[m.InstanceID] {
				m.ProfileName = tenant.StripTenantPrefix(m.ProfileName, tenantID)
				filtered = append(filtered, m)
			}
		}
		metrics = filtered
	}
	web.JSON(w, 200, metrics)
}

// tenantInstanceIDs returns the set of instance IDs owned by the given tenant.
func (o *Orchestrator) tenantInstanceIDs(tenantID string) map[string]bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	ids := make(map[string]bool)
	for id, inst := range o.instances {
		if tenant.HasTenantPrefix(inst.ProfileName, tenantID) {
			ids[id] = true
		}
	}
	return ids
}

// checkInstanceTenant verifies the caller owns the instance. Returns the
// instance or writes a 404 and returns nil.
func (o *Orchestrator) checkInstanceTenant(w http.ResponseWriter, id string, tenantID string) *InstanceInternal {
	o.mu.RLock()
	inst, ok := o.instances[id]
	o.mu.RUnlock()
	if !ok || !tenant.HasTenantPrefix(inst.ProfileName, tenantID) {
		web.Error(w, 404, fmt.Errorf("instance %q not found", id))
		return nil
	}
	return inst
}

func stripInstanceTenant(inst bridge.Instance, tenantID string) bridge.Instance {
	if tenantID != "" {
		inst.ProfileName = tenant.StripTenantPrefix(inst.ProfileName, tenantID)
	}
	return inst
}


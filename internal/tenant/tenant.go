package tenant

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type tenantContextKey struct{}

// TenantFromContext returns the tenant ID from the request context.
// Returns "" for admin/legacy requests (no tenant scoping).
func TenantFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(tenantContextKey{}).(string); ok {
		return v
	}
	return ""
}

// WithTenant returns a context with the tenant ID set.
func WithTenant(ctx context.Context, tenant string) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, tenant)
}

// ValidateHMACToken checks if the token is a valid HMAC tenant token
// (format: tenantID:hmac_signature). Returns (tenantID, true) if valid.
func ValidateHMACToken(token, secret string) (string, bool) {
	idx := strings.IndexByte(token, ':')
	if idx <= 0 {
		return "", false
	}
	tenantID := token[:idx]
	sig := token[idx+1:]
	expected := computeHMAC(tenantID, secret)
	if len(sig) != len(expected) {
		return "", false
	}
	if hmac.Equal([]byte(sig), []byte(expected)) {
		return tenantID, true
	}
	return "", false
}

func computeHMAC(message, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// Tenant profile name helpers.

const tenantSep = "--"

// AddTenantPrefix prepends the tenant prefix to a profile name.
func AddTenantPrefix(name, tenant string) string {
	if tenant == "" {
		return name
	}
	return tenant + tenantSep + name
}

// StripTenantPrefix removes the tenant prefix from a profile name.
func StripTenantPrefix(name, tenant string) string {
	if tenant == "" {
		return name
	}
	return strings.TrimPrefix(name, tenant+tenantSep)
}

// HasTenantPrefix checks if a profile name belongs to the given tenant.
// Returns true if tenant is "" (admin/legacy mode).
func HasTenantPrefix(name, tenant string) bool {
	if tenant == "" {
		return true
	}
	return strings.HasPrefix(name, tenant+tenantSep)
}

// IsTempProfile checks if a profile name (possibly tenant-prefixed) is a
// temporary auto-generated instance profile.
func IsTempProfile(name string) bool {
	if strings.HasPrefix(name, "instance-") {
		return true
	}
	if idx := strings.Index(name, tenantSep); idx >= 0 {
		return strings.HasPrefix(name[idx+len(tenantSep):], "instance-")
	}
	return false
}

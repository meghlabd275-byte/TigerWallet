package services

// tenant_service_test.go — exercises the tenant lifecycle against a real
// PostgreSQL schema. It runs only when TEST_DATABASE_URL (or DATABASE_URL) is
// set, and skips otherwise, matching the convention used by the other Go
// modules in this repository.

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tigerwallet/multi-tenancy/internal/database"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL/DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("database unavailable: %v", err)
	}
	if err := database.Migrate(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("migrate: %v", err)
	}
	Pool = pool
	t.Cleanup(pool.Close)
	return pool
}

func uniqueSlug() string {
	return "t-" + uuid.New().String()[:8]
}

// A tenant must be created together with its default quotas and config. Before
// the lifecycle was wrapped in a transaction, a failure after the tenant INSERT
// left a committed tenant with no quotas, and every later quota call then
// returned "quota not found" forever.
func TestCreateTenantProvisionsQuotasAndConfigAtomically(t *testing.T) {
	pool := testPool(t)
	svc := &TenantService{}
	ctx := context.Background()

	tenant, err := svc.CreateTenant(ctx, "Atomic Co", "atomic@example.com", uniqueSlug())
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenant.ID)
	})

	var quotaCount, configCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM quotas WHERE tenant_id = $1`, tenant.ID).Scan(&quotaCount); err != nil {
		t.Fatalf("count quotas: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tenant_configs WHERE tenant_id = $1`, tenant.ID).Scan(&configCount); err != nil {
		t.Fatalf("count configs: %v", err)
	}
	if quotaCount == 0 {
		t.Error("tenant created with no default quotas")
	}
	if configCount == 0 {
		t.Error("tenant created with no default config")
	}
}

// Lookups must bind the identifier they are given. Both lookups previously
// issued SQL with a missing parameter, so every call returned an error.
func TestLookupBindsIdentifier(t *testing.T) {
	testPool(t)
	svc := &TenantService{}
	ctx := context.Background()

	slug := uniqueSlug()
	tenant, err := svc.CreateTenant(ctx, "Lookup Co", "lookup@example.com", slug)
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = Pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenant.ID)
	})

	byID, err := svc.GetTenantByID(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("GetTenantByID: %v", err)
	}
	if byID.ID != tenant.ID {
		t.Errorf("GetTenantByID returned %s, want %s", byID.ID, tenant.ID)
	}

	bySlug, err := svc.GetTenantBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("GetTenantBySlug: %v", err)
	}
	if bySlug.ID != tenant.ID {
		t.Errorf("GetTenantBySlug returned %s, want %s", bySlug.ID, tenant.ID)
	}

	// A different slug must not resolve to this tenant: that is what a missing
	// or mis-bound parameter produced before the fix.
	if _, err := svc.GetTenantBySlug(ctx, uniqueSlug()); err == nil {
		t.Error("GetTenantBySlug resolved a slug that does not exist")
	}
}

// Quota accounting must read and write the same resource row.
func TestQuotaIncrementAndCheck(t *testing.T) {
	testPool(t)
	svc := &TenantService{}
	ctx := context.Background()

	tenant, err := svc.CreateTenant(ctx, "Quota Co", "quota@example.com", uniqueSlug())
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = Pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenant.ID)
	})

	before, err := svc.GetQuota(ctx, tenant.ID.String(), "api_calls")
	if err != nil {
		t.Fatalf("GetQuota: %v", err)
	}
	if err := svc.IncrementQuota(ctx, tenant.ID.String(), "api_calls", 7); err != nil {
		t.Fatalf("IncrementQuota: %v", err)
	}
	after, err := svc.GetQuota(ctx, tenant.ID.String(), "api_calls")
	if err != nil {
		t.Fatalf("GetQuota after increment: %v", err)
	}
	if after.Used != before.Used+7 {
		t.Errorf("used = %d, want %d", after.Used, before.Used+7)
	}
	if allowed, err := svc.CheckQuota(ctx, tenant.ID.String(), "api_calls"); err != nil || !allowed {
		t.Errorf("CheckQuota = %v, %v; want true, nil", allowed, err)
	}
}

// Lifecycle mutations must leave an audit trail. The audit table existed but
// nothing ever wrote to it, so the trail was permanently empty.
func TestLifecycleWritesAuditTrail(t *testing.T) {
	testPool(t)
	svc := &TenantService{}
	ctx := context.Background()

	tenant, err := svc.CreateTenant(ctx, "Audit Co", "audit@example.com", uniqueSlug())
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = Pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenant.ID)
	})

	userID := uuid.New()
	if _, err := svc.AddUserToTenant(ctx, tenant.ID, userID, "admin"); err != nil {
		t.Fatalf("AddUserToTenant: %v", err)
	}
	if err := svc.UpdateUserRole(ctx, tenant.ID, userID, "owner"); err != nil {
		t.Fatalf("UpdateUserRole: %v", err)
	}
	if err := svc.UpdateTenantStatus(ctx, tenant.ID, "suspended"); err != nil {
		t.Fatalf("UpdateTenantStatus: %v", err)
	}

	logs, err := svc.GetAuditLogs(ctx, tenant.ID, 50)
	if err != nil {
		t.Fatalf("GetAuditLogs: %v", err)
	}
	seen := map[string]bool{}
	for _, l := range logs {
		seen[l.Action] = true
	}
	for _, want := range []string{"tenant.created", "tenant.user_role_changed", "tenant.status_changed"} {
		if !seen[want] {
			t.Errorf("audit trail is missing %q (got %v)", want, seen)
		}
	}
}

// Deleting a tenant is a soft delete and must be observable as a status change.
func TestDeleteTenantIsSoft(t *testing.T) {
	testPool(t)
	svc := &TenantService{}
	ctx := context.Background()

	tenant, err := svc.CreateTenant(ctx, "Delete Co", "delete@example.com", uniqueSlug())
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = Pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, tenant.ID)
	})

	if err := svc.DeleteTenant(ctx, tenant.ID); err != nil {
		t.Fatalf("DeleteTenant: %v", err)
	}
	got, err := svc.GetTenantByID(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("GetTenantByID after delete: %v", err)
	}
	if got.Status != "terminated" {
		t.Errorf("status = %q, want terminated", got.Status)
	}
}

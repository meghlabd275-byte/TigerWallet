package services

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tigerwallet/super-admin/internal/database"
)

// The SLA report aggregates with FILTER, make_interval and NOW(), which cannot
// be checked without a server. Point TEST_DATABASE_URL at a throwaway
// PostgreSQL database to run it; without that the test skips. The fixture
// recreates the tickets and sla_policies tables, so the database must not hold
// data that matters.
func TestGenerateReportCountsRealBreaches(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping SLA integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	previous := database.Pool
	database.Pool = pool
	defer func() { database.Pool = previous }()

	for _, stmt := range []string{
		`DROP TABLE IF EXISTS ticket_messages`,
		`DROP TABLE IF EXISTS tickets`,
		`DROP TABLE IF EXISTS sla_reports`,
		`DROP TABLE IF EXISTS sla_policies`,
		`DROP TABLE IF EXISTS service_health_checks`,
		`CREATE TABLE sla_policies (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			description TEXT,
			priority VARCHAR(20) NOT NULL,
			response_time_sla INTEGER NOT NULL,
			resolution_time_sla INTEGER NOT NULL,
			uptime_sla DECIMAL(5,2),
			is_active BOOLEAN DEFAULT TRUE,
			created_by UUID,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE tickets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			created_at TIMESTAMPTZ NOT NULL,
			first_response_at TIMESTAMPTZ,
			resolved_at TIMESTAMPTZ,
			sla_policy_id UUID REFERENCES sla_policies(id)
		)`,
		`CREATE TABLE sla_reports (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			policy_id UUID REFERENCES sla_policies(id),
			period_start TIMESTAMP WITH TIME ZONE NOT NULL,
			period_end TIMESTAMP WITH TIME ZONE NOT NULL,
			total_tickets INTEGER DEFAULT 0,
			met_sla INTEGER DEFAULT 0,
			breached_sla INTEGER DEFAULT 0,
			avg_response_time DECIMAL(10,2),
			avg_resolution_time DECIMAL(10,2),
			uptime DECIMAL(5,2),
			uptime_measured BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE service_health_checks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service TEXT NOT NULL,
			healthy BOOLEAN NOT NULL,
			latency_ms INTEGER,
			detail TEXT,
			checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("schema (%.40s): %v", stmt, err)
		}
	}

	policyID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	otherPolicyID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	for _, id := range []uuid.UUID{policyID, otherPolicyID} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO sla_policies (id, name, priority, response_time_sla, resolution_time_sla, uptime_sla)
			 VALUES ($1, 'it', 'high', 300, 600, 99.9)`, id); err != nil {
			t.Fatalf("policy: %v", err)
		}
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	insert := `INSERT INTO tickets (created_at, first_response_at, resolved_at, sla_policy_id) VALUES ($1,$2,$3,$4)`
	rows := []struct {
		created, firstResponse, resolved any
	}{
		{base, base.Add(60 * time.Second), base.Add(120 * time.Second)},   // met
		{base, base.Add(900 * time.Second), base.Add(1000 * time.Second)}, // response breach
		{base.Add(10 * time.Minute), nil, nil},                            // unanswered, breach
		{base, base.Add(10 * time.Second), base.Add(9000 * time.Second)},  // resolution breach
	}
	for _, r := range rows {
		if _, err := pool.Exec(ctx, insert, r.created, r.firstResponse, r.resolved, policyID); err != nil {
			t.Fatalf("ticket: %v", err)
		}
	}
	// A ticket under a different policy must not leak into the report.
	if _, err := pool.Exec(ctx, insert, base, base, base, otherPolicyID); err != nil {
		t.Fatalf("other-policy ticket: %v", err)
	}

	// Health checks: 3 of 4 healthy, so uptime is 75%.
	for i, healthy := range []bool{true, true, false, true} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO service_health_checks (service, healthy, checked_at) VALUES ($1,$2,$3)`,
			"api", healthy, base.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("health check: %v", err)
		}
	}

	svc := &SLAService{}
	report, err := svc.GenerateReport(ctx, policyID, base.Add(-time.Hour), base.Add(20*time.Minute))
	if err != nil {
		t.Fatalf("GenerateReport: %v", err)
	}

	if report.TotalTickets != 4 {
		t.Errorf("TotalTickets = %d, want 4", report.TotalTickets)
	}
	if report.MetSLA != 1 {
		t.Errorf("MetSLA = %d, want 1", report.MetSLA)
	}
	if report.BreachedSLA != 3 {
		t.Errorf("BreachedSLA = %d, want 3", report.BreachedSLA)
	}
	if report.MetSLA+report.BreachedSLA != report.TotalTickets {
		t.Errorf("met (%d) + breached (%d) must equal total (%d)",
			report.MetSLA, report.BreachedSLA, report.TotalTickets)
	}
	if !report.UptimeMeasured {
		t.Error("UptimeMeasured should be true when health checks exist")
	}
	if report.Uptime < 74.9 || report.Uptime > 75.1 {
		t.Errorf("Uptime = %.2f, want 75.00", report.Uptime)
	}

	// The report must be persisted, not just returned.
	var stored int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM sla_reports WHERE id = $1`, report.ID).Scan(&stored); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if stored != 1 {
		t.Errorf("persisted rows = %d, want 1", stored)
	}

	// A period with no health checks must report "not measured" rather than 0%.
	empty, err := svc.GenerateReport(ctx, policyID, base.Add(24*time.Hour), base.Add(48*time.Hour))
	if err != nil {
		t.Fatalf("empty-window report: %v", err)
	}
	if empty.UptimeMeasured {
		t.Error("UptimeMeasured should be false when no health checks were recorded")
	}
	if empty.TotalTickets != 0 {
		t.Errorf("empty window TotalTickets = %d, want 0", empty.TotalTickets)
	}

	// An unknown policy must fail closed instead of writing a bogus report.
	if _, err := svc.GenerateReport(ctx, uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		base, base.Add(time.Hour)); err == nil {
		t.Error("GenerateReport should fail for an unknown policy")
	}
}

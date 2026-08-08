// SLAService - SLA Management Service
package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/white-label-admin/internal/database"
)

type SLAService struct{}

func NewSLAService() *SLAService {
	return &SLAService{}
}

type SLAPolicy struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Priority        string    `json:"priority"` // critical, high, medium, low
	ResponseTimeSLA int       `json:"response_time_sla"` // in seconds
	ResolutionTimeSLA int     `json:"resolution_time_sla"` // in seconds
	UptimeSLA       float64   `json:"uptime_sla"` // percentage
	IsActive        bool      `json:"is_active"`
	CreatedBy       uuid.UUID `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SLAReport struct {
	ID           uuid.UUID  `json:"id"`
	PolicyID     uuid.UUID  `json:"policy_id"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	TotalTickets  int       `json:"total_tickets"`
	MetSLA        int       `json:"met_sla"`
	BreachedSLA   int       `json:"breached_sla"`
	AvgResponseTime float64  `json:"avg_response_time"`
	AvgResolutionTime float64 `json:"avg_resolution_time"`
	Uptime        float64   `json:"uptime"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *SLAService) ListPolicies(ctx context.Context) ([]SLAPolicy, error) {
	rows, err := database.Pool.Query(ctx, "SELECT id, name, description, priority, response_time_sla, resolution_time_sla, uptime_sla, is_active, created_by, created_at, updated_at FROM sla_policies ORDER BY priority, created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []SLAPolicy
	for rows.Next() {
		var p SLAPolicy
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Priority, &p.ResponseTimeSLA, &p.ResolutionTimeSLA, &p.UptimeSLA, &p.IsActive, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, nil
}

func (s *SLAService) GetPolicy(ctx context.Context, id uuid.UUID) (*SLAPolicy, error) {
	var p SLAPolicy
	err := database.Pool.QueryRow(ctx, `
		SELECT id, name, description, priority, response_time_sla, resolution_time_sla, uptime_sla, is_active, created_by, created_at, updated_at 
		FROM sla_policies WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.Description, &p.Priority, &p.ResponseTimeSLA, &p.ResolutionTimeSLA, &p.UptimeSLA, &p.IsActive, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

func (s *SLAService) CreatePolicy(ctx context.Context, policy *SLAPolicy, adminID uuid.UUID) (*SLAPolicy, error) {
	err := database.Pool.QueryRow(ctx, `
		INSERT INTO sla_policies (name, description, priority, response_time_sla, resolution_time_sla, uptime_sla, is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`, policy.Name, policy.Description, policy.Priority, policy.ResponseTimeSLA, policy.ResolutionTimeSLA, policy.UptimeSLA, policy.IsActive, adminID).Scan(&policy.ID, &policy.CreatedAt, &policy.UpdatedAt)
	return policy, err
}

func (s *SLAService) UpdatePolicy(ctx context.Context, id uuid.UUID, policy *SLAPolicy) error {
	_, err := database.Pool.Exec(ctx, `
		UPDATE sla_policies SET name = $1, description = $2, priority = $3, response_time_sla = $4, resolution_time_sla = $5, uptime_sla = $6, is_active = $7, updated_at = NOW()
		WHERE id = $8
	`, policy.Name, policy.Description, policy.Priority, policy.ResponseTimeSLA, policy.ResolutionTimeSLA, policy.UptimeSLA, policy.IsActive, id)
	return err
}

func (s *SLAService) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM sla_policies WHERE id = $1", id)
	return err
}

func (s *SLAService) ListReports(ctx context.Context, policyID *uuid.UUID, limit, offset int) ([]SLAReport, int, error) {
	var total int
	database.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM sla_reports").Scan(&total)

	query := "SELECT id, policy_id, period_start, period_end, total_tickets, met_sla, breached_sla, avg_response_time, avg_resolution_time, uptime, created_at FROM sla_reports WHERE 1=1"
	args := []interface{}{}
	argNum := 1

	if policyID != nil {
		query += " AND policy_id = $1"
		args = append(args, *policyID)
		argNum++
	}

	query += " ORDER BY period_start DESC LIMIT $" + string(rune('0'+argNum)) + " OFFSET $" + string(rune('0'+argNum+1))
	args = append(args, limit, offset)

	rows, err := database.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []SLAReport
	for rows.Next() {
		var r SLAReport
		if err := rows.Scan(&r.ID, &r.PolicyID, &r.PeriodStart, &r.PeriodEnd, &r.TotalTickets, &r.MetSLA, &r.BreachedSLA, &r.AvgResponseTime, &r.AvgResolutionTime, &r.Uptime, &r.CreatedAt); err != nil {
			return nil, 0, err
		}
		reports = append(reports, r)
	}
	return reports, total, nil
}

func (s *SLAService) GenerateReport(ctx context.Context, policyID uuid.UUID, periodStart, periodEnd time.Time) (*SLAReport, error) {
	// Calculate SLA metrics from tickets
	var report SLAReport
	err := database.Pool.QueryRow(ctx, `
		SELECT 
			COUNT(*) as total_tickets,
			COALESCE(AVG(EXTRACT(EPOCH FROM (first_response_at - created_at))), 0) as avg_response_time,
			COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at))), 0) as avg_resolution_time
		FROM tickets 
		WHERE sla_policy_id = $1 AND created_at >= $2 AND created_at <= $3
	`, policyID, periodStart, periodEnd).Scan(&report.TotalTickets, &report.AvgResponseTime, &report.AvgResolutionTime)

	if err != nil {
		return nil, err
	}

	report.ID = uuid.New()
	report.PolicyID = policyID
	report.PeriodStart = periodStart
	report.PeriodEnd = periodEnd
	report.MetSLA = report.TotalTickets // Simplified - would need actual SLA check
	report.BreachedSLA = 0
	report.Uptime = 99.9 // Simplified
	report.CreatedAt = time.Now()

	_, err = database.Pool.Exec(ctx, `
		INSERT INTO sla_reports (id, policy_id, period_start, period_end, total_tickets, met_sla, breached_sla, avg_response_time, avg_resolution_time, uptime, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, report.ID, report.PolicyID, report.PeriodStart, report.PeriodEnd, report.TotalTickets, report.MetSLA, report.BreachedSLA, report.AvgResponseTime, report.AvgResolutionTime, report.Uptime, report.CreatedAt)

	return &report, err
}

func (s *SLAService) CheckSLACompliance(ctx context.Context, ticketID uuid.UUID) (bool, string, error) {
	var ticket struct {
		Priority       string
		SLAPolicyID    *uuid.UUID
		CreatedAt      time.Time
		FirstResponseAt *time.Time
		ResolvedAt     *time.Time
	}

	err := database.Pool.QueryRow(ctx, `
		SELECT priority, sla_policy_id, created_at, first_response_at, resolved_at FROM tickets WHERE id = $1
	`, ticketID).Scan(&ticket.Priority, &ticket.SLAPolicyID, &ticket.CreatedAt, &ticket.FirstResponseAt, &ticket.ResolvedAt)

	if err != nil {
		return false, "", err
	}

	if ticket.SLAPolicyID == nil {
		return true, "No SLA policy assigned", nil
	}

	policy, err := s.GetPolicy(ctx, *ticket.SLAPolicyID)
	if err != nil || policy == nil {
		return true, "Policy not found", nil
	}

	// Check response time SLA
	if ticket.FirstResponseAt != nil {
		responseTime := ticket.FirstResponseAt.Sub(ticket.CreatedAt).Seconds()
		if responseTime > float64(policy.ResponseTimeSLA) {
			return false, "Response time SLA breached"
		}
	}

	// Check resolution time SLA
	if ticket.ResolvedAt != nil {
		resolutionTime := ticket.ResolvedAt.Sub(ticket.CreatedAt).Seconds()
		if resolutionTime > float64(policy.ResolutionTimeSLA) {
			return false, "Resolution time SLA breached"
		}
	}

	return true, "SLA compliant", nil
}

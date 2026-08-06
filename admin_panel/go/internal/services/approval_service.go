// ApprovalService - Approval workflow service for high-value transactions
package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type ApprovalService struct{}

func NewApprovalService() *ApprovalService {
	return &ApprovalService{}
}

// Workflow management

func (s *ApprovalService) ListWorkflows(ctx context.Context) ([]models.ApprovalWorkflow, error) {
	rows, err := database.Query(ctx, `
		SELECT id, name, description, workflow_type, threshold_amount, required_approvals, approvers, is_active, created_at, created_by
		FROM approval_workflows WHERE is_active = true ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []models.ApprovalWorkflow
	for rows.Next() {
		var wf models.ApprovalWorkflow
		err := rows.Scan(&wf.ID, &wf.Name, &wf.Description, &wf.WorkflowType, &wf.ThresholdAmount, &wf.RequiredApprovals, &wf.Approvers, &wf.IsActive, &wf.CreatedAt, &wf.CreatedBy)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, wf)
	}
	return workflows, nil
}

func (s *ApprovalService) CreateWorkflow(ctx context.Context, wf *models.ApprovalWorkflow, adminID uuid.UUID) (*models.ApprovalWorkflow, error) {
	err := database.QueryRow(ctx, `
		INSERT INTO approval_workflows (name, description, workflow_type, threshold_amount, required_approvals, approvers, is_active, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8)
		RETURNING id, created_at
	`, wf.Name, wf.Description, wf.WorkflowType, wf.ThresholdAmount, wf.RequiredApprovals, wf.Approvers, wf.IsActive, adminID).Scan(&wf.ID, &wf.CreatedAt)
	return wf, err
}

func (s *ApprovalService) UpdateWorkflow(ctx context.Context, id uuid.UUID, wf *models.ApprovalWorkflow) error {
	_, err := database.Exec(ctx, `
		UPDATE approval_workflows SET name = $1, description = $2, workflow_type = $3, threshold_amount = $4, 
		required_approvals = $5, approvers = $6, is_active = $7 WHERE id = $8
	`, wf.Name, wf.Description, wf.WorkflowType, wf.ThresholdAmount, wf.RequiredApprovals, wf.Approvers, wf.IsActive, id)
	return err
}

func (s *ApprovalService) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	_, err := database.Exec(ctx, "DELETE FROM approval_workflows WHERE id = $1", id)
	return err
}

// Approval request management

func (s *ApprovalService) CreateApprovalRequest(ctx context.Context, req *models.ApprovalRequest, adminID uuid.UUID) (*models.ApprovalRequest, error) {
	err := database.QueryRow(ctx, `
		INSERT INTO approval_requests (workflow_id, request_type, resource_id, requester_id, status, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, created_at
	`, req.WorkflowID, req.RequestType, req.ResourceID, adminID, "pending", req.Details).Scan(&req.ID, &req.CreatedAt)
	return req, err
}

func (s *ApprovalService) GetApprovalRequest(ctx context.Context, id uuid.UUID) (*models.ApprovalRequest, error) {
	var req models.ApprovalRequest
	err := database.QueryRow(ctx, `
		SELECT id, workflow_id, request_type, resource_id, requester_id, status, details, approved_by, approved_at, reject_reason, created_at
		FROM approval_requests WHERE id = $1
	`, id).Scan(&req.ID, &req.WorkflowID, &req.RequestType, &req.ResourceID, &req.RequesterID, &req.Status, &req.Details, &req.ApprovedBy, &req.ApprovedAt, &req.RejectReason, &req.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &req, err
}

func (s *ApprovalService) ListApprovalRequests(ctx context.Context, status string, limit, offset int) ([]models.ApprovalRequest, int, error) {
	var total int
	database.QueryRow(ctx, "SELECT COUNT(*) FROM approval_requests").Scan(&total)

	rows, err := database.Query(ctx, `
		SELECT id, workflow_id, request_type, resource_id, requester_id, status, details, approved_by, approved_at, reject_reason, created_at
		FROM approval_requests ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var requests []models.ApprovalRequest
	for rows.Next() {
		var req models.ApprovalRequest
		err := rows.Scan(&req.ID, &req.WorkflowID, &req.RequestType, &req.ResourceID, &req.RequesterID, &req.Status, &req.Details, &req.ApprovedBy, &req.ApprovedAt, &req.RejectReason, &req.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		requests = append(requests, req)
	}
	return requests, total, nil
}

func (s *ApprovalService) ApproveRequest(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	_, err := database.Exec(ctx, `
		UPDATE approval_requests SET status = 'approved', approved_by = $1, approved_at = $2 WHERE id = $3
	`, adminID, time.Now(), id)
	return err
}

func (s *ApprovalService) RejectRequest(ctx context.Context, id uuid.UUID, adminID uuid.UUID, reason string) error {
	_, err := database.Exec(ctx, `
		UPDATE approval_requests SET status = 'rejected', approved_by = $1, reject_reason = $2 WHERE id = $3
	`, adminID, reason, id)
	return err
}

// Check if a transaction requires approval
func (s *ApprovalService) RequiresApproval(ctx context.Context, amount float64, workflowType string) (bool, uuid.UUID, error) {
	var wf models.ApprovalWorkflow
	err := database.QueryRow(ctx, `
		SELECT id, threshold_amount FROM approval_workflows 
		WHERE workflow_type = $1 AND is_active = true AND threshold_amount <= $2
		ORDER BY threshold_amount DESC LIMIT 1
	`, workflowType, amount).Scan(&wf.ID, &wf.ThresholdAmount)

	if err == pgx.ErrNoRows {
		return false, uuid.Nil, nil
	}
	if err != nil {
		return false, uuid.Nil, err
	}

	return true, wf.ID, nil
}

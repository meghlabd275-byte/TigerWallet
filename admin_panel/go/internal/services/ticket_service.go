// TicketService - Support ticket management service
package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/tigerwallet/admin_panel/internal/database"
	"github.com/tigerwallet/admin_panel/internal/models"
)

type TicketService struct{}

func NewTicketService() *TicketService {
	return &TicketService{}
}

func (s *TicketService) ListTickets(ctx context.Context, status string, limit, offset int) ([]models.Ticket, int, error) {
	var total int
	if status != "" {
		database.QueryRow(ctx, "SELECT COUNT(*) FROM tickets WHERE status = $1", status).Scan(&total)
	} else {
		database.QueryRow(ctx, "SELECT COUNT(*) FROM tickets").Scan(&total)
	}

	var rows, _ = database.Query(ctx, `
		SELECT id, title, description, ticket_type, priority, status, created_by, assigned_to, created_at, updated_at, resolved_at
		FROM tickets ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if rows == nil {
		return nil, total, nil
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var ticket models.Ticket
		err := rows.Scan(&ticket.ID, &ticket.Title, &ticket.Description, &ticket.TicketType, &ticket.Priority, &ticket.Status, &ticket.CreatedBy, &ticket.AssignedTo, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.ResolvedAt)
		if err != nil {
			return nil, 0, err
		}
		tickets = append(tickets, ticket)
	}
	return tickets, total, nil
}

func (s *TicketService) GetTicket(ctx context.Context, id uuid.UUID) (*models.Ticket, []models.TicketMessage, error) {
	var ticket models.Ticket
	err := database.QueryRow(ctx, `
		SELECT id, title, description, ticket_type, priority, status, created_by, assigned_to, created_at, updated_at, resolved_at
		FROM tickets WHERE id = $1
	`, id).Scan(&ticket.ID, &ticket.Title, &ticket.Description, &ticket.TicketType, &ticket.Priority, &ticket.Status, &ticket.CreatedBy, &ticket.AssignedTo, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.ResolvedAt)
	if err != nil {
		return nil, nil, err
	}

	// Get messages
	rows, _ := database.Query(ctx, `
		SELECT id, ticket_id, message, is_internal, created_by, created_at
		FROM ticket_messages WHERE ticket_id = $1 ORDER BY created_at ASC
	`, id)
	if rows != nil {
		defer rows.Close()
		var messages []models.TicketMessage
		for rows.Next() {
			var msg models.TicketMessage
			rows.Scan(&msg.ID, &msg.TicketID, &msg.Message, &msg.IsInternal, &msg.CreatedBy, &msg.CreatedAt)
			messages = append(messages, msg)
		}
		return &ticket, messages, nil
	}

	return &ticket, nil, nil
}

func (s *TicketService) CreateTicket(ctx context.Context, adminID uuid.UUID, title, description, ticketType, priority string) (*models.Ticket, error) {
	if priority == "" {
		priority = "medium"
	}
	ticket := &models.Ticket{
		Title:       title,
		Description: description,
		TicketType:  ticketType,
		Priority:    priority,
		Status:      "open",
		CreatedBy:   adminID,
	}
	err := database.QueryRow(ctx, `
		INSERT INTO tickets (title, description, ticket_type, priority, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`, ticket.Title, ticket.Description, ticket.TicketType, ticket.Priority, ticket.Status, ticket.CreatedBy).Scan(&ticket.ID, &ticket.CreatedAt, &ticket.UpdatedAt)
	return ticket, err
}

func (s *TicketService) UpdateTicketStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := database.Exec(ctx, "UPDATE tickets SET status = $1, updated_at = NOW() WHERE id = $2", status, id)
	return err
}

func (s *TicketService) AddMessage(ctx context.Context, ticketID uuid.UUID, adminID uuid.UUID, message string, isInternal bool) (*models.TicketMessage, error) {
	msg := &models.TicketMessage{
		TicketID:   ticketID,
		Message:    message,
		IsInternal: isInternal,
		CreatedBy:  adminID,
	}
	err := database.QueryRow(ctx, `
		INSERT INTO ticket_messages (ticket_id, message, is_internal, created_by, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, created_at
	`, msg.TicketID, msg.Message, msg.IsInternal, msg.CreatedBy).Scan(&msg.ID, &msg.CreatedAt)
	return msg, err
}

func (s *TicketService) AssignTicket(ctx context.Context, ticketID uuid.UUID, assignedTo uuid.UUID) error {
	_, err := database.Exec(ctx, "UPDATE tickets SET assigned_to = $1, updated_at = NOW() WHERE id = $2", assignedTo, ticketID)
	return err
}

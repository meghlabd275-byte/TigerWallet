// KnowledgeBaseService - Knowledge base article management
package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tigerwallet/super-admin/internal/database"
)

type KnowledgeBaseService struct{}

func NewKnowledgeBaseService() *KnowledgeBaseService {
	return &KnowledgeBaseService{}
}

type KnowledgeArticle struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Category    string     `json:"category"`
	Tags        []string   `json:"tags"`
	IsPublished bool       `json:"is_published"`
	ViewCount   int        `json:"view_count"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (s *KnowledgeBaseService) ListArticles(ctx context.Context, category string, published *bool, limit, offset int) ([]KnowledgeArticle, int, error) {
	var total int
	err := database.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM knowledge_articles").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := "SELECT id, title, content, category, tags, is_published, view_count, created_by, created_at, updated_at FROM knowledge_articles WHERE 1=1"
	args := []interface{}{}
	argNum := 1

	if category != "" {
		query += " AND category = $" + string(rune('0'+argNum))
		args = append(args, category)
		argNum++
	}
	if published != nil {
		query += " AND is_published = $" + string(rune('0'+argNum))
		args = append(args, *published)
		argNum++
	}

	query += " ORDER BY created_at DESC LIMIT $" + string(rune('0'+argNum)) + " OFFSET $" + string(rune('0'+argNum+1))
	args = append(args, limit, offset)

	rows, err := database.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var articles []KnowledgeArticle
	for rows.Next() {
		var a KnowledgeArticle
		err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.Tags, &a.IsPublished, &a.ViewCount, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		articles = append(articles, a)
	}
	return articles, total, nil
}

func (s *KnowledgeBaseService) GetArticle(ctx context.Context, id uuid.UUID) (*KnowledgeArticle, error) {
	var a KnowledgeArticle
	err := database.Pool.QueryRow(ctx, `
		SELECT id, title, content, category, tags, is_published, view_count, created_by, created_at, updated_at 
		FROM knowledge_articles WHERE id = $1
	`, id).Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.Tags, &a.IsPublished, &a.ViewCount, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Increment view count
	database.Pool.Exec(ctx, "UPDATE knowledge_articles SET view_count = view_count + 1 WHERE id = $1", id)
	a.ViewCount++
	return &a, nil
}

func (s *KnowledgeBaseService) CreateArticle(ctx context.Context, article *KnowledgeArticle, adminID uuid.UUID) (*KnowledgeArticle, error) {
	err := database.Pool.QueryRow(ctx, `
		INSERT INTO knowledge_articles (title, content, category, tags, is_published, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`, article.Title, article.Content, article.Category, article.Tags, article.IsPublished, adminID).Scan(&article.ID, &article.CreatedAt, &article.UpdatedAt)
	return article, err
}

func (s *KnowledgeBaseService) UpdateArticle(ctx context.Context, id uuid.UUID, article *KnowledgeArticle) error {
	_, err := database.Pool.Exec(ctx, `
		UPDATE knowledge_articles SET title = $1, content = $2, category = $3, tags = $4, is_published = $5, updated_at = NOW()
		WHERE id = $6
	`, article.Title, article.Content, article.Category, article.Tags, article.IsPublished, id)
	return err
}

func (s *KnowledgeBaseService) DeleteArticle(ctx context.Context, id uuid.UUID) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM knowledge_articles WHERE id = $1", id)
	return err
}

func (s *KnowledgeBaseService) SearchArticles(ctx context.Context, query string, limit int) ([]KnowledgeArticle, error) {
	rows, err := database.Pool.Query(ctx, `
		SELECT id, title, content, category, tags, is_published, view_count, created_by, created_at, updated_at 
		FROM knowledge_articles 
		WHERE is_published = true AND (title ILIKE $1 OR content ILIKE $1 OR $2 = ANY(tags))
		ORDER BY view_count DESC LIMIT $3
	`, "%"+query+"%", query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []KnowledgeArticle
	for rows.Next() {
		var a KnowledgeArticle
		err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.Category, &a.Tags, &a.IsPublished, &a.ViewCount, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, nil
}

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"

	"github.com/gin-gonic/gin"
)

// KnowledgeBaseHandler handles knowledge base operations
type KnowledgeBaseHandler struct {
	db *database.PostgresDB
}

// NewKnowledgeBaseHandler creates a new knowledge base handler
func NewKnowledgeBaseHandler(db *database.PostgresDB) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{db: db}
}

// CategoryRequest represents category request
type CategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ParentID    uint   `json:"parent_id"`
	Order       int    `json:"order"`
	IsActive    bool   `json:"is_active"`
}

// ArticleRequest represents article request
type ArticleRequest struct {
	CategoryID  uint   `json:"category_id" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Content     string `json:"content" binding:"required"`
	Summary     string `json:"summary"`
	Tags        string `json:"tags"`
	IsPublished bool   `json:"is_published"`
	IsFeatured  bool   `json:"is_featured"`
	Order       int    `json:"order"`
}

// CreateCategory creates a new category
// POST /api/v1/admin/knowledge/categories
func (h *KnowledgeBaseHandler) CreateCategory(c *gin.Context) {
	var req CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Generate slug
	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))

	category := models.KnowledgeBaseCategory{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		ParentID:    &req.ParentID,
		Order:       req.Order,
		IsActive:    req.IsActive,
	}

	if err := h.db.Create(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, category)
}

// UpdateCategory updates a category
// PUT /api/v1/admin/knowledge/categories/:id
func (h *KnowledgeBaseHandler) UpdateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	var req CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var category models.KnowledgeBaseCategory
	if err := h.db.First(&category, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	updates := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"order":       req.Order,
		"is_active":   req.IsActive,
	}

	if req.ParentID > 0 {
		updates["parent_id"] = req.ParentID
	}

	if err := h.db.Model(&category).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category"})
		return
	}

	c.JSON(http.StatusOK, category)
}

// DeleteCategory deletes a category
// DELETE /api/v1/admin/knowledge/categories/:id
func (h *KnowledgeBaseHandler) DeleteCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	if err := h.db.Delete(&models.KnowledgeBaseCategory{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}

// ListCategories lists all categories
// GET /api/v1/admin/knowledge/categories
func (h *KnowledgeBaseHandler) ListCategories(c *gin.Context) {
	var categories []models.KnowledgeBaseCategory

	query := h.db.Model(&models.KnowledgeBaseCategory{})
	if c.Query("active_only") == "true" {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Order("\"order\" ASC, name ASC").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// CreateArticle creates a new article
// POST /api/v1/admin/knowledge/articles
func (h *KnowledgeBaseHandler) CreateArticle(c *gin.Context) {
	var req ArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Generate slug
	slug := strings.ToLower(strings.ReplaceAll(req.Title, " ", "-"))
	slug = strings.ReplaceAll(slug, "'", "")
	slug = strings.ReplaceAll(slug, "\"", "")

	article := models.KnowledgeBaseArticle{
		CategoryID:  req.CategoryID,
		Title:       req.Title,
		Slug:        slug,
		Content:     req.Content,
		Summary:     req.Summary,
		Tags:        json.RawMessage(req.Tags),
		IsPublished: req.IsPublished,
		IsFeatured:  req.IsFeatured,
		Order:       req.Order,
		AuthorID:    c.GetUint("admin_id"),
	}

	if err := h.db.Create(&article).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create article"})
		return
	}

	c.JSON(http.StatusCreated, article)
}

// UpdateArticle updates an article
// PUT /api/v1/admin/knowledge/articles/:id
func (h *KnowledgeBaseHandler) UpdateArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	var req ArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var article models.KnowledgeBaseArticle
	if err := h.db.First(&article, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	updates := map[string]interface{}{
		"category_id":  req.CategoryID,
		"title":        req.Title,
		"content":      req.Content,
		"summary":      req.Summary,
		"tags":         req.Tags,
		"is_published": req.IsPublished,
		"is_featured":  req.IsFeatured,
		"order":        req.Order,
	}

	// Update slug if title changed
	if article.Title != req.Title {
		slug := strings.ToLower(strings.ReplaceAll(req.Title, " ", "-"))
		slug = strings.ReplaceAll(slug, "'", "")
		updates["slug"] = slug
	}

	if err := h.db.Model(&article).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update article"})
		return
	}

	c.JSON(http.StatusOK, article)
}

// DeleteArticle deletes an article
// DELETE /api/v1/admin/knowledge/articles/:id
func (h *KnowledgeBaseHandler) DeleteArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	if err := h.db.Delete(&models.KnowledgeBaseArticle{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete article"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article deleted successfully"})
}

// ListArticles lists all articles
// GET /api/v1/admin/knowledge/articles
func (h *KnowledgeBaseHandler) ListArticles(c *gin.Context) {
	categoryID := c.Query("category_id")
	publishedOnly := c.Query("published_only") == "true"
	search := c.Query("search")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt, _ := strconv.Atoi(page)
	pageSizeInt, _ := strconv.Atoi(pageSize)

	var articles []models.KnowledgeBaseArticle
	var total int64

	query := h.db.Model(&models.KnowledgeBaseArticle{})

	if categoryID != "" {
		cid, _ := strconv.ParseUint(categoryID, 10, 32)
		query = query.Where("category_id = ?", cid)
	}

	if publishedOnly {
		query = query.Where("is_published = ?", true)
	}

	if search != "" {
		query = query.Where("title ILIKE ? OR content ILIKE ?",
			"%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)

	offset := (pageInt - 1) * pageSizeInt
	query = query.Offset(offset).Limit(pageSizeInt).Order("created_at DESC")

	if err := query.Find(&articles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch articles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        articles,
		"total":       total,
		"page":        pageInt,
		"page_size":   pageSizeInt,
		"total_pages": (total + int64(pageSizeInt) - 1) / int64(pageSizeInt),
	})
}

// GetArticle gets an article by ID or slug
// GET /api/v1/admin/knowledge/articles/:id
func (h *KnowledgeBaseHandler) GetArticle(c *gin.Context) {
	idOrSlug := c.Param("id")

	var article models.KnowledgeBaseArticle
	err := h.db.Preload("Category").First(&article, idOrSlug).Error
	if err != nil {
		// Try by slug
		err = h.db.Preload("Category").Where("slug = ?", idOrSlug).First(&article).Error
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
			return
		}
	}

	// Increment view count
	h.db.Model(&article).Update("view_count", article.ViewCount+1)

	c.JSON(http.StatusOK, article)
}

// GetPublicArticles gets published articles for public access
// GET /api/v1/knowledge/articles
func (h *KnowledgeBaseHandler) GetPublicArticles(c *gin.Context) {
	categorySlug := c.Query("category")
	search := c.Query("search")

	var articles []models.KnowledgeBaseArticle

	query := h.db.Where("is_published = ?", true)

	if categorySlug != "" {
		query = query.Joins("JOIN knowledge_base_categories ON knowledge_base_categories.id = knowledge_base_articles.category_id").
			Where("knowledge_base_categories.slug = ?", categorySlug)
	}

	if search != "" {
		query = query.Where("title ILIKE ? OR content ILIKE ?",
			"%"+search+"%", "%"+search+"%")
	}

	if err := query.Order("is_featured DESC, view_count DESC, created_at DESC").Find(&articles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch articles"})
		return
	}

	c.JSON(http.StatusOK, articles)
}

// GetKnowledgeBaseStats gets knowledge base statistics
// GET /api/v1/admin/knowledge/stats
func (h *KnowledgeBaseHandler) GetKnowledgeBaseStats(c *gin.Context) {
	var stats struct {
		TotalCategories   int64 `json:"total_categories"`
		ActiveCategories  int64 `json:"active_categories"`
		TotalArticles     int64 `json:"total_articles"`
		PublishedArticles int64 `json:"published_articles"`
		FeaturedArticles  int64 `json:"featured_articles"`
		TotalViews        int64 `json:"total_views"`
	}

	h.db.Model(&models.KnowledgeBaseCategory{}).Count(&stats.TotalCategories)
	h.db.Model(&models.KnowledgeBaseCategory{}).Where("is_active = ?", true).Count(&stats.ActiveCategories)
	h.db.Model(&models.KnowledgeBaseArticle{}).Count(&stats.TotalArticles)
	h.db.Model(&models.KnowledgeBaseArticle{}).Where("is_published = ?", true).Count(&stats.PublishedArticles)
	h.db.Model(&models.KnowledgeBaseArticle{}).Where("is_featured = ?", true).Count(&stats.FeaturedArticles)

	h.db.Model(&models.KnowledgeBaseArticle{}).Select("COALESCE(SUM(view_count), 0)").Scan(&stats.TotalViews)

	c.JSON(http.StatusOK, stats)
}

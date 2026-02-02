package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/optics/backend/internal/models"
)

type CategoryHandler struct {
	db *pgxpool.Pool
}

func NewCategoryHandler(db *pgxpool.Pool) *CategoryHandler {
	return &CategoryHandler{db: db}
}

func (h *CategoryHandler) List(c *gin.Context) {
	rows, err := h.db.Query(
		context.Background(),
		`SELECT c.id, c.name, c.slug, c.parent_id, c.description, c.icon, c.created_at,
		 (SELECT COUNT(*) FROM models WHERE category_id = c.id) as model_count
		 FROM categories c ORDER BY c.name`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}
	defer rows.Close()

	categories := make(map[uuid.UUID]*models.Category)
	var rootCategories []*models.Category

	for rows.Next() {
		var cat models.Category
		err := rows.Scan(
			&cat.ID, &cat.Name, &cat.Slug, &cat.ParentID,
			&cat.Description, &cat.Icon, &cat.CreatedAt, &cat.ModelCount,
		)
		if err != nil {
			continue
		}
		cat.Children = []models.Category{}
		categories[cat.ID] = &cat
	}

	// Build tree structure
	for _, cat := range categories {
		if cat.ParentID == nil {
			rootCategories = append(rootCategories, cat)
		} else if parent, ok := categories[*cat.ParentID]; ok {
			parent.Children = append(parent.Children, *cat)
		}
	}

	// Convert to slice
	result := make([]models.Category, len(rootCategories))
	for i, cat := range rootCategories {
		result[i] = *cat
	}

	c.JSON(http.StatusOK, result)
}

func (h *CategoryHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	var cat models.Category
	err = h.db.QueryRow(
		context.Background(),
		`SELECT c.id, c.name, c.slug, c.parent_id, c.description, c.icon, c.created_at,
		 (SELECT COUNT(*) FROM models WHERE category_id = c.id) as model_count
		 FROM categories c WHERE c.id = $1`,
		id,
	).Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.ParentID, &cat.Description, &cat.Icon, &cat.CreatedAt, &cat.ModelCount)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Load children
	rows, err := h.db.Query(
		context.Background(),
		`SELECT id, name, slug, parent_id, description, icon, created_at
		 FROM categories WHERE parent_id = $1 ORDER BY name`,
		id,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var child models.Category
			if rows.Scan(&child.ID, &child.Name, &child.Slug, &child.ParentID, &child.Description, &child.Icon, &child.CreatedAt) == nil {
				cat.Children = append(cat.Children, child)
			}
		}
	}

	c.JSON(http.StatusOK, cat)
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var req models.CategoryCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slug := slugify(req.Name)

	var cat models.Category
	err := h.db.QueryRow(
		context.Background(),
		`INSERT INTO categories (name, slug, parent_id, description, icon)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, slug, parent_id, description, icon, created_at`,
		req.Name, slug, req.ParentID, req.Description, req.Icon,
	).Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.ParentID, &cat.Description, &cat.Icon, &cat.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Category with this slug already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, cat)
}

func (h *CategoryHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	var req models.CategoryUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current category
	var current models.Category
	err = h.db.QueryRow(
		context.Background(),
		"SELECT id, name, slug, parent_id, description, icon FROM categories WHERE id = $1",
		id,
	).Scan(&current.ID, &current.Name, &current.Slug, &current.ParentID, &current.Description, &current.Icon)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Apply updates
	if req.Name != nil {
		current.Name = *req.Name
		current.Slug = slugify(*req.Name)
	}
	if req.ParentID != nil {
		current.ParentID = req.ParentID
	}
	if req.Description != nil {
		current.Description = req.Description
	}
	if req.Icon != nil {
		current.Icon = req.Icon
	}

	_, err = h.db.Exec(
		context.Background(),
		`UPDATE categories SET name = $1, slug = $2, parent_id = $3, description = $4, icon = $5
		 WHERE id = $6`,
		current.Name, current.Slug, current.ParentID, current.Description, current.Icon, id,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category"})
		return
	}

	c.JSON(http.StatusOK, current)
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	// Check for children
	var childCount int
	err = h.db.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM categories WHERE parent_id = $1",
		id,
	).Scan(&childCount)

	if err == nil && childCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete category with children"})
		return
	}

	// Set models to uncategorized
	_, err = h.db.Exec(
		context.Background(),
		"UPDATE models SET category_id = NULL WHERE category_id = $1",
		id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to uncategorize models"})
		return
	}

	// Delete category
	result, err := h.db.Exec(context.Background(), "DELETE FROM categories WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}

// Tags handlers
type TagHandler struct {
	db *pgxpool.Pool
}

func NewTagHandler(db *pgxpool.Pool) *TagHandler {
	return &TagHandler{db: db}
}

func (h *TagHandler) List(c *gin.Context) {
	rows, err := h.db.Query(
		context.Background(),
		"SELECT id, name, color FROM tags ORDER BY name",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags"})
		return
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		if rows.Scan(&tag.ID, &tag.Name, &tag.Color) == nil {
			tags = append(tags, tag)
		}
	}

	c.JSON(http.StatusOK, tags)
}

func (h *TagHandler) Create(c *gin.Context) {
	var req models.TagCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	color := "#6366f1"
	if req.Color != nil {
		color = *req.Color
	}

	var tag models.Tag
	err := h.db.QueryRow(
		context.Background(),
		`INSERT INTO tags (name, color) VALUES ($1, $2)
		 RETURNING id, name, color`,
		req.Name, color,
	).Scan(&tag.ID, &tag.Name, &tag.Color)

	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Tag already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag"})
		return
	}

	c.JSON(http.StatusCreated, tag)
}

func (h *TagHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	result, err := h.db.Exec(context.Background(), "DELETE FROM tags WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tag"})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tag deleted successfully"})
}

// Collections handlers
type CollectionHandler struct {
	db *pgxpool.Pool
}

func NewCollectionHandler(db *pgxpool.Pool) *CollectionHandler {
	return &CollectionHandler{db: db}
}

func (h *CollectionHandler) List(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := h.db.Query(
		context.Background(),
		`SELECT c.id, c.user_id, c.name, c.description, c.is_public, c.created_at,
		 (SELECT COUNT(*) FROM collection_models WHERE collection_id = c.id) as model_count
		 FROM collections c WHERE c.user_id = $1 ORDER BY c.created_at DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch collections"})
		return
	}
	defer rows.Close()

	var collections []models.Collection
	for rows.Next() {
		var col models.Collection
		if rows.Scan(&col.ID, &col.UserID, &col.Name, &col.Description, &col.IsPublic, &col.CreatedAt, &col.ModelCount) == nil {
			collections = append(collections, col)
		}
	}

	c.JSON(http.StatusOK, collections)
}

func (h *CollectionHandler) Create(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CollectionCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isPublic := false
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	var col models.Collection
	err := h.db.QueryRow(
		context.Background(),
		`INSERT INTO collections (user_id, name, description, is_public)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, name, description, is_public, created_at`,
		userID, req.Name, req.Description, isPublic,
	).Scan(&col.ID, &col.UserID, &col.Name, &col.Description, &col.IsPublic, &col.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create collection"})
		return
	}

	c.JSON(http.StatusCreated, col)
}

func (h *CollectionHandler) AddModel(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	collectionID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	// Verify ownership
	var ownerID uuid.UUID
	err = h.db.QueryRow(
		context.Background(),
		"SELECT user_id FROM collections WHERE id = $1",
		collectionID,
	).Scan(&ownerID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}

	if ownerID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to modify this collection"})
		return
	}

	var req struct {
		ModelID uuid.UUID `json:"model_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = h.db.Exec(
		context.Background(),
		`INSERT INTO collection_models (collection_id, model_id)
		 VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		collectionID, req.ModelID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add model to collection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model added to collection"})
}

func (h *CollectionHandler) RemoveModel(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	collectionID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	modelIDStr := c.Param("modelId")
	modelID, err := uuid.Parse(modelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	// Verify ownership
	var ownerID uuid.UUID
	err = h.db.QueryRow(
		context.Background(),
		"SELECT user_id FROM collections WHERE id = $1",
		collectionID,
	).Scan(&ownerID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}

	if ownerID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to modify this collection"})
		return
	}

	_, err = h.db.Exec(
		context.Background(),
		"DELETE FROM collection_models WHERE collection_id = $1 AND model_id = $2",
		collectionID, modelID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove model from collection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model removed from collection"})
}

func (h *CollectionHandler) Delete(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	collectionID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collection ID"})
		return
	}

	result, err := h.db.Exec(
		context.Background(),
		"DELETE FROM collections WHERE id = $1 AND user_id = $2",
		collectionID, userID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete collection"})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found or not authorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Collection deleted successfully"})
}

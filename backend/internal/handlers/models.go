package handlers

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/optics/backend/internal/config"
	"github.com/optics/backend/internal/middleware"
	"github.com/optics/backend/internal/models"
)

type ModelHandler struct {
	db      *pgxpool.Pool
	storage *config.StorageConfig
}

func NewModelHandler(db *pgxpool.Pool, storage *config.StorageConfig) *ModelHandler {
	return &ModelHandler{db: db, storage: storage}
}

func (h *ModelHandler) List(c *gin.Context) {
	var filter models.ModelFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}

	offset := (filter.Page - 1) * filter.Limit

	// Build query
	query := `SELECT m.id, m.name, m.slug, m.description, m.file_path, m.file_type,
			  m.file_size, m.file_hash, m.thumbnail_path, m.source, m.source_id,
			  m.license, m.author, m.category_id, m.indexed_at, m.created_at, m.updated_at
			  FROM models m WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM models m WHERE 1=1`

	var args []interface{}
	argNum := 1

	if filter.Category != nil && *filter.Category != "" {
		categoryID, err := uuid.Parse(*filter.Category)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
			return
		}
		query += fmt.Sprintf(" AND m.category_id = $%d", argNum)
		countQuery += fmt.Sprintf(" AND m.category_id = $%d", argNum)
		args = append(args, categoryID)
		argNum++
	}

	if filter.FileType != nil {
		query += fmt.Sprintf(" AND m.file_type = $%d", argNum)
		countQuery += fmt.Sprintf(" AND m.file_type = $%d", argNum)
		args = append(args, *filter.FileType)
		argNum++
	}

	if filter.Source != nil {
		query += fmt.Sprintf(" AND m.source = $%d", argNum)
		countQuery += fmt.Sprintf(" AND m.source = $%d", argNum)
		args = append(args, *filter.Source)
		argNum++
	}

	if filter.Search != nil && *filter.Search != "" {
		query += fmt.Sprintf(" AND to_tsvector('english', m.name) @@ plainto_tsquery('english', $%d)", argNum)
		countQuery += fmt.Sprintf(" AND to_tsvector('english', m.name) @@ plainto_tsquery('english', $%d)", argNum)
		args = append(args, *filter.Search)
		argNum++
	}

	// Get total count
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := h.db.QueryRow(context.Background(), countQuery, countArgs...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count models"})
		return
	}

	// Get models
	query += fmt.Sprintf(" ORDER BY m.created_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, filter.Limit, offset)

	rows, err := h.db.Query(context.Background(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch models"})
		return
	}
	defer rows.Close()

	var modelList []models.Model
	for rows.Next() {
		var m models.Model
		err := rows.Scan(
			&m.ID, &m.Name, &m.Slug, &m.Description, &m.FilePath, &m.FileType,
			&m.FileSize, &m.FileHash, &m.ThumbnailPath, &m.Source, &m.SourceID,
			&m.License, &m.Author, &m.CategoryID, &m.IndexedAt, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			continue
		}
		modelList = append(modelList, m)
	}

	totalPages := (total + filter.Limit - 1) / filter.Limit

	c.JSON(http.StatusOK, models.ModelListResponse{
		Models:     modelList,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	})
}

func (h *ModelHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	var m models.Model
	err = h.db.QueryRow(
		context.Background(),
		`SELECT id, name, slug, description, file_path, file_type, file_size,
		 file_hash, thumbnail_path, source, source_id, license, author,
		 category_id, indexed_at, created_at, updated_at
		 FROM models WHERE id = $1`,
		id,
	).Scan(
		&m.ID, &m.Name, &m.Slug, &m.Description, &m.FilePath, &m.FileType, &m.FileSize,
		&m.FileHash, &m.ThumbnailPath, &m.Source, &m.SourceID, &m.License, &m.Author,
		&m.CategoryID, &m.IndexedAt, &m.CreatedAt, &m.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	// Load tags
	rows, err := h.db.Query(
		context.Background(),
		`SELECT t.id, t.name, t.color FROM tags t
		 JOIN model_tags mt ON mt.tag_id = t.id
		 WHERE mt.model_id = $1`,
		id,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tag models.Tag
			if rows.Scan(&tag.ID, &tag.Name, &tag.Color) == nil {
				m.Tags = append(m.Tags, tag)
			}
		}
	}

	// Load category if exists
	if m.CategoryID != nil {
		var cat models.Category
		err = h.db.QueryRow(
			context.Background(),
			`SELECT id, name, slug, parent_id, description, icon, created_at
			 FROM categories WHERE id = $1`,
			*m.CategoryID,
		).Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.ParentID, &cat.Description, &cat.Icon, &cat.CreatedAt)
		if err == nil {
			m.Category = &cat
		}
	}

	c.JSON(http.StatusOK, m)
}

func (h *ModelHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	var req models.ModelUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *req.Name)
		argNum++

		// Update slug too
		updates = append(updates, fmt.Sprintf("slug = $%d", argNum))
		args = append(args, slugify(*req.Name))
		argNum++
	}

	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argNum))
		args = append(args, *req.Description)
		argNum++
	}

	if req.License != nil {
		updates = append(updates, fmt.Sprintf("license = $%d", argNum))
		args = append(args, *req.License)
		argNum++
	}

	if req.Author != nil {
		updates = append(updates, fmt.Sprintf("author = $%d", argNum))
		args = append(args, *req.Author)
		argNum++
	}

	if req.CategoryID != nil {
		updates = append(updates, fmt.Sprintf("category_id = $%d", argNum))
		args = append(args, *req.CategoryID)
		argNum++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(
		"UPDATE models SET %s WHERE id = $%d RETURNING id",
		strings.Join(updates, ", "),
		argNum,
	)

	var updatedID uuid.UUID
	err = h.db.QueryRow(context.Background(), query, args...).Scan(&updatedID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update model"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": updatedID, "message": "Model updated successfully"})
}

func (h *ModelHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	result, err := h.db.Exec(context.Background(), "DELETE FROM models WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete model"})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model deleted successfully"})
}

func (h *ModelHandler) ServeFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	var filePath, fileType string
	err = h.db.QueryRow(
		context.Background(),
		"SELECT file_path, file_type FROM models WHERE id = $1",
		id,
	).Scan(&filePath, &fileType)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	contentType := getContentType(fileType)

	// Check if file is inside a ZIP (format: archive.zip!path/inside/file.stl)
	if strings.Contains(filePath, "!") {
		parts := strings.SplitN(filePath, "!", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path"})
			return
		}

		zipPath := filepath.Join(h.storage.MountPath, parts[0])
		innerPath := parts[1]

		reader, err := zip.OpenReader(zipPath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "ZIP file not found"})
			return
		}
		defer reader.Close()

		for _, file := range reader.File {
			if filepath.ToSlash(file.Name) == innerPath {
				rc, err := file.Open()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file in ZIP"})
					return
				}
				defer rc.Close()

				c.Header("Content-Type", contentType)
				c.Header("Content-Length", fmt.Sprintf("%d", file.UncompressedSize64))
				c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filepath.Base(innerPath)))

				io.Copy(c.Writer, rc)
				return
			}
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "File not found in ZIP"})
		return
	}

	// Regular file
	fullPath := filepath.Join(h.storage.MountPath, filePath)

	file, err := os.Open(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get file info"})
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size()))
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filepath.Base(filePath)))

	io.Copy(c.Writer, file)
}

func (h *ModelHandler) ServeThumbnail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	var thumbnailPath *string
	err = h.db.QueryRow(
		context.Background(),
		"SELECT thumbnail_path FROM models WHERE id = $1",
		id,
	).Scan(&thumbnailPath)

	if err != nil || thumbnailPath == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thumbnail not found"})
		return
	}

	fullPath := filepath.Join(h.storage.ThumbnailPath, *thumbnailPath)
	c.File(fullPath)
}

func (h *ModelHandler) AddTags(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	var req struct {
		TagIDs []uuid.UUID `json:"tag_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, tagID := range req.TagIDs {
		_, err := h.db.Exec(
			context.Background(),
			`INSERT INTO model_tags (model_id, tag_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`,
			id, tagID,
		)
		if err != nil {
			continue
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tags added successfully"})
}

func (h *ModelHandler) RemoveTag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	tagIDStr := c.Param("tagId")
	tagID, err := uuid.Parse(tagIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	_, err = h.db.Exec(
		context.Background(),
		"DELETE FROM model_tags WHERE model_id = $1 AND tag_id = $2",
		id, tagID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove tag"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tag removed successfully"})
}

func (h *ModelHandler) SetCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	var req struct {
		CategoryID *uuid.UUID `json:"category_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = h.db.Exec(
		context.Background(),
		"UPDATE models SET category_id = $1, updated_at = NOW() WHERE id = $2",
		req.CategoryID, id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category set successfully"})
}

func (h *ModelHandler) ToggleFavorite(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model ID"})
		return
	}

	// Check if already favorited
	var exists bool
	err = h.db.QueryRow(
		context.Background(),
		"SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND model_id = $2)",
		userID, id,
	).Scan(&exists)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check favorite status"})
		return
	}

	if exists {
		// Remove favorite
		_, err = h.db.Exec(
			context.Background(),
			"DELETE FROM favorites WHERE user_id = $1 AND model_id = $2",
			userID, id,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove favorite"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"favorited": false})
	} else {
		// Add favorite
		_, err = h.db.Exec(
			context.Background(),
			"INSERT INTO favorites (user_id, model_id) VALUES ($1, $2)",
			userID, id,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add favorite"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"favorited": true})
	}
}

func (h *ModelHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	rows, err := h.db.Query(
		context.Background(),
		`SELECT id, name, slug, description, file_path, file_type, file_size,
		 thumbnail_path, source, created_at
		 FROM models
		 WHERE to_tsvector('english', name) @@ plainto_tsquery('english', $1)
		 ORDER BY ts_rank(to_tsvector('english', name), plainto_tsquery('english', $1)) DESC
		 LIMIT 50`,
		query,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}
	defer rows.Close()

	var results []models.Model
	for rows.Next() {
		var m models.Model
		err := rows.Scan(
			&m.ID, &m.Name, &m.Slug, &m.Description, &m.FilePath, &m.FileType,
			&m.FileSize, &m.ThumbnailPath, &m.Source, &m.CreatedAt,
		)
		if err == nil {
			results = append(results, m)
		}
	}

	c.JSON(http.StatusOK, gin.H{"results": results, "count": len(results)})
}

func getContentType(fileType string) string {
	switch strings.ToLower(fileType) {
	case "stl":
		return "model/stl"
	case "obj":
		return "model/obj"
	case "scad":
		return "text/plain"
	case "3mf":
		return "model/3mf"
	case "step", "stp":
		return "model/step"
	default:
		return "application/octet-stream"
	}
}

func slugify(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/optics/backend/internal/config"
	"github.com/optics/backend/internal/models"
	"github.com/optics/backend/internal/services"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	db         *pgxpool.Pool
	cfg        *config.Config
	scanner    *services.Scanner
	duplicates *services.DuplicateDetector
}

func NewAdminHandler(db *pgxpool.Pool, cfg *config.Config, scanner *services.Scanner) *AdminHandler {
	return &AdminHandler{db: db, cfg: cfg, scanner: scanner, duplicates: services.NewDuplicateDetector(db)}
}

// Settings Management

func (h *AdminHandler) GetSettings(c *gin.Context) {
	ctx := context.Background()

	rows, err := h.db.Query(ctx, "SELECT key, value, updated_at FROM settings")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
		return
	}
	defer rows.Close()

	settings := make(map[string]interface{})
	for rows.Next() {
		var s models.Settings
		if rows.Scan(&s.Key, &s.Value, &s.UpdatedAt) == nil {
			settings[s.Key] = s.Value
		}
	}

	// Add default model path if not set
	if _, ok := settings["model_path"]; !ok {
		settings["model_path"] = h.cfg.Storage.MountPath
	}

	c.JSON(http.StatusOK, settings)
}

func (h *AdminHandler) UpdateSetting(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate allowed settings
	allowedSettings := map[string]bool{
		"model_path":     true,
		"thumbnail_path": true,
	}
	if !allowedSettings[req.Key] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid setting key"})
		return
	}

	ctx := context.Background()
	_, err := h.db.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()
	`, req.Key, req.Value)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting"})
		return
	}

	// Update scanner config if model path changed
	if req.Key == "model_path" {
		h.cfg.Storage.MountPath = req.Value
	}

	c.JSON(http.StatusOK, gin.H{"message": "Setting updated successfully"})
}

func (h *AdminHandler) GetModelPaths(c *gin.Context) {
	ctx := context.Background()

	var pathsJSON string
	err := h.db.QueryRow(ctx, "SELECT value FROM settings WHERE key = 'model_paths'").Scan(&pathsJSON)
	if err != nil {
		// Return default path
		c.JSON(http.StatusOK, gin.H{"paths": []string{h.cfg.Storage.MountPath}})
		return
	}

	// Parse JSON array
	var paths []string
	if err := json.Unmarshal([]byte(pathsJSON), &paths); err != nil {
		c.JSON(http.StatusOK, gin.H{"paths": []string{h.cfg.Storage.MountPath}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"paths": paths})
}

func (h *AdminHandler) SetModelPaths(c *gin.Context) {
	var req struct {
		Paths []string `json:"paths" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pathsJSON, _ := json.Marshal(req.Paths)

	ctx := context.Background()
	_, err := h.db.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES ('model_paths', $1, NOW())
		ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = NOW()
	`, string(pathsJSON))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update model paths"})
		return
	}

	// Update the config with first path
	if len(req.Paths) > 0 {
		h.cfg.Storage.MountPath = req.Paths[0]
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model paths updated successfully", "paths": req.Paths})
}

func (h *AdminHandler) AddModelPath(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Get existing paths
	var pathsJSON string
	var paths []string
	err := h.db.QueryRow(ctx, "SELECT value FROM settings WHERE key = 'model_paths'").Scan(&pathsJSON)
	if err == nil {
		json.Unmarshal([]byte(pathsJSON), &paths)
	} else {
		paths = []string{h.cfg.Storage.MountPath}
	}

	// Check if path already exists
	for _, p := range paths {
		if p == req.Path {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Path already exists"})
			return
		}
	}

	// Add new path
	paths = append(paths, req.Path)
	pathsJSON2, _ := json.Marshal(paths)

	_, err = h.db.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES ('model_paths', $1, NOW())
		ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = NOW()
	`, string(pathsJSON2))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add path"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Path added", "paths": paths})
}

func (h *AdminHandler) RemoveModelPath(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Get existing paths
	var pathsJSON string
	var paths []string
	err := h.db.QueryRow(ctx, "SELECT value FROM settings WHERE key = 'model_paths'").Scan(&pathsJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No paths configured"})
		return
	}
	json.Unmarshal([]byte(pathsJSON), &paths)

	// Remove path
	var newPaths []string
	for _, p := range paths {
		if p != req.Path {
			newPaths = append(newPaths, p)
		}
	}

	if len(newPaths) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove last path"})
		return
	}

	pathsJSON2, _ := json.Marshal(newPaths)
	h.db.Exec(ctx, `UPDATE settings SET value = $1, updated_at = NOW() WHERE key = 'model_paths'`, string(pathsJSON2))

	c.JSON(http.StatusOK, gin.H{"message": "Path removed", "paths": newPaths})
}

func (h *AdminHandler) BrowseDirectory(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = "/mnt"
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read directory: " + err.Error()})
		return
	}

	var dirs []gin.H
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, gin.H{
				"name": entry.Name(),
				"path": filepath.Join(path, entry.Name()),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"current":     path,
		"parent":      filepath.Dir(path),
		"directories": dirs,
	})
}

// Legacy single path support
func (h *AdminHandler) GetModelPath(c *gin.Context) {
	ctx := context.Background()

	var pathsJSON string
	err := h.db.QueryRow(ctx, "SELECT value FROM settings WHERE key = 'model_paths'").Scan(&pathsJSON)
	if err == nil {
		var paths []string
		if json.Unmarshal([]byte(pathsJSON), &paths) == nil && len(paths) > 0 {
			c.JSON(http.StatusOK, gin.H{"path": paths[0]})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"path": h.cfg.Storage.MountPath})
}

func (h *AdminHandler) SetModelPath(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	paths := []string{req.Path}
	pathsJSON, _ := json.Marshal(paths)

	ctx := context.Background()
	_, err := h.db.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES ('model_paths', $1, NOW())
		ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = NOW()
	`, string(pathsJSON))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update model path"})
		return
	}

	h.cfg.Storage.MountPath = req.Path

	c.JSON(http.StatusOK, gin.H{"message": "Model path updated successfully", "path": req.Path})
}

// User Management

func (h *AdminHandler) ListUsers(c *gin.Context) {
	ctx := context.Background()

	rows, err := h.db.Query(ctx, `
		SELECT id, username, email, role, created_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []models.UserAdmin
	for rows.Next() {
		var u models.UserAdmin
		if rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.CreatedAt) == nil {
			users = append(users, u)
		}
	}

	c.JSON(http.StatusOK, users)
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req models.UserCreateAdmin
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	ctx := context.Background()
	var user models.UserAdmin
	err = h.db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, email, role, created_at
	`, req.Username, req.Email, string(hashedPassword), req.Role).Scan(
		&user.ID, &user.Username, &user.Email, &user.Role, &user.CreatedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req models.UserUpdateAdmin
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Build update query
	if req.Username != nil {
		_, err = h.db.Exec(ctx, "UPDATE users SET username = $1, updated_at = NOW() WHERE id = $2", *req.Username, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update username"})
			return
		}
	}

	if req.Email != nil {
		_, err = h.db.Exec(ctx, "UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2", *req.Email, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update email"})
			return
		}
	}

	if req.Role != nil {
		if *req.Role != "user" && *req.Role != "admin" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
			return
		}
		_, err = h.db.Exec(ctx, "UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2", *req.Role, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
			return
		}
	}

	if req.Password != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		_, err = h.db.Exec(ctx, "UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(hashedPassword), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx := context.Background()

	// Don't allow deleting the last admin
	var adminCount int
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&adminCount)

	var userRole string
	h.db.QueryRow(ctx, "SELECT role FROM users WHERE id = $1", id).Scan(&userRole)

	if userRole == "admin" && adminCount <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete the last admin user"})
		return
	}

	result, err := h.db.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// Database Management

func (h *AdminHandler) ClearDatabase(c *gin.Context) {
	var req struct {
		Confirm string `json:"confirm" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Confirm != "CLEAR_ALL_DATA" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Confirmation required. Send {\"confirm\": \"CLEAR_ALL_DATA\"}"})
		return
	}

	ctx := context.Background()

	// Clear tables in order (respecting foreign keys)
	tables := []string{
		"duplicate_members",
		"duplicate_groups",
		"collection_models",
		"collections",
		"favorites",
		"model_tags",
		"models",
		"tags",
		"categories",
	}

	for _, table := range tables {
		_, err := h.db.Exec(ctx, "DELETE FROM "+table)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear " + table})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Database cleared successfully"})
}

func (h *AdminHandler) ClearModels(c *gin.Context) {
	var req struct {
		Confirm string `json:"confirm" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Confirm != "CLEAR_MODELS" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Confirmation required. Send {\"confirm\": \"CLEAR_MODELS\"}"})
		return
	}

	ctx := context.Background()

	// Clear model-related tables
	tables := []string{
		"duplicate_members",
		"duplicate_groups",
		"collection_models",
		"favorites",
		"model_tags",
		"models",
	}

	for _, table := range tables {
		_, err := h.db.Exec(ctx, "DELETE FROM "+table)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear " + table})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Models cleared successfully"})
}

// Scanner endpoints (moved from files.go)

func (h *AdminHandler) StartScan(c *gin.Context) {
	// Get all configured paths
	ctx := context.Background()
	var pathsJSON string
	var paths []string
	err := h.db.QueryRow(ctx, "SELECT value FROM settings WHERE key = 'model_paths'").Scan(&pathsJSON)
	if err == nil {
		json.Unmarshal([]byte(pathsJSON), &paths)
	}
	if len(paths) == 0 {
		paths = []string{h.cfg.Storage.MountPath}
	}

	// Start scan with all paths
	go h.scanner.StartMultiple(paths)
	c.JSON(http.StatusAccepted, gin.H{"message": "Scan started", "status": "running", "paths": paths})
}

func (h *AdminHandler) GetScanStatus(c *gin.Context) {
	status := h.scanner.GetStatus()
	c.JSON(http.StatusOK, status)
}

func (h *AdminHandler) StopScan(c *gin.Context) {
	h.scanner.Stop()
	c.JSON(http.StatusOK, gin.H{"message": "Scan stop requested"})
}

func (h *AdminHandler) DetectDuplicates(c *gin.Context) {
	go h.duplicates.DetectByHash(context.Background())
	c.JSON(http.StatusAccepted, gin.H{"message": "Duplicate detection started"})
}

func (h *AdminHandler) ListDuplicates(c *gin.Context) {
	groups, _, err := h.duplicates.GetDuplicateGroups(context.Background(), 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch duplicates"})
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (h *AdminHandler) ResolveDuplicate(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	var req struct {
		PrimaryModelID uuid.UUID `json:"primary_model_id" binding:"required"`
		DeleteOthers   bool      `json:"delete_others"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.duplicates.ResolveDuplicateGroup(context.Background(), groupID, req.PrimaryModelID, req.DeleteOthers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve duplicate group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Duplicate group resolved"})
}

func (h *AdminHandler) GetStats(c *gin.Context) {
	ctx := context.Background()

	var stats struct {
		TotalModels      int            `json:"total_models"`
		TotalCategories  int            `json:"total_categories"`
		TotalTags        int            `json:"total_tags"`
		TotalUsers       int            `json:"total_users"`
		DuplicateGroups  int            `json:"duplicate_groups"`
		ModelsByType     map[string]int `json:"models_by_type"`
		ModelsBySource   map[string]int `json:"models_by_source"`
		UncategorizedCnt int            `json:"uncategorized_count"`
	}
	stats.ModelsByType = make(map[string]int)
	stats.ModelsBySource = make(map[string]int)

	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM models").Scan(&stats.TotalModels)
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM categories").Scan(&stats.TotalCategories)
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM tags").Scan(&stats.TotalTags)
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM duplicate_groups").Scan(&stats.DuplicateGroups)
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM models WHERE category_id IS NULL").Scan(&stats.UncategorizedCnt)

	// Models by type
	rows, err := h.db.Query(ctx, "SELECT file_type, COUNT(*) FROM models GROUP BY file_type")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var fileType string
			var count int
			if rows.Scan(&fileType, &count) == nil {
				stats.ModelsByType[fileType] = count
			}
		}
	}

	// Models by source
	rows, err = h.db.Query(ctx, "SELECT COALESCE(source, 'unknown'), COUNT(*) FROM models GROUP BY source")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var source string
			var count int
			if rows.Scan(&source, &count) == nil {
				stats.ModelsBySource[source] = count
			}
		}
	}

	c.JSON(http.StatusOK, stats)
}

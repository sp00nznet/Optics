package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ThumbnailGenerator struct {
	db            *pgxpool.Pool
	thumbnailPath string
	mountPath     string
}

func NewThumbnailGenerator(db *pgxpool.Pool, thumbnailPath, mountPath string) *ThumbnailGenerator {
	// Ensure thumbnail directory exists
	os.MkdirAll(thumbnailPath, 0755)

	return &ThumbnailGenerator{
		db:            db,
		thumbnailPath: thumbnailPath,
		mountPath:     mountPath,
	}
}

// GenerateThumbnail generates a thumbnail for a model
// This is a placeholder - actual implementation would require:
// - For STL: Use a 3D rendering library or headless Three.js
// - For SCAD: Render using OpenSCAD CLI then generate thumbnail
func (t *ThumbnailGenerator) GenerateThumbnail(ctx context.Context, modelID uuid.UUID) (string, error) {
	var filePath, fileType string
	err := t.db.QueryRow(
		ctx,
		"SELECT file_path, file_type FROM models WHERE id = $1",
		modelID,
	).Scan(&filePath, &fileType)
	if err != nil {
		return "", fmt.Errorf("model not found: %w", err)
	}

	fullPath := filepath.Join(t.mountPath, filePath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("model file not found: %s", fullPath)
	}

	// Generate thumbnail filename
	thumbnailName := fmt.Sprintf("%s.png", modelID.String())
	thumbnailPath := filepath.Join(t.thumbnailPath, thumbnailName)

	// Placeholder: In a real implementation, you would:
	// 1. Parse the STL/OBJ/3MF file
	// 2. Render it to an image using a 3D library
	// 3. Save the image to thumbnailPath

	// For now, we'll create a placeholder file
	// In production, replace this with actual rendering logic
	if err := t.createPlaceholder(thumbnailPath); err != nil {
		return "", err
	}

	// Update database with thumbnail path
	_, err = t.db.Exec(
		ctx,
		"UPDATE models SET thumbnail_path = $1 WHERE id = $2",
		thumbnailName, modelID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to update thumbnail path: %w", err)
	}

	return thumbnailName, nil
}

func (t *ThumbnailGenerator) createPlaceholder(path string) error {
	// Create a minimal 1x1 PNG (placeholder)
	// In production, replace with actual thumbnail generation
	placeholder := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe,
		0xd4, 0xef, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, // IEND chunk
		0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	return os.WriteFile(path, placeholder, 0644)
}

// GenerateBatch generates thumbnails for multiple models
func (t *ThumbnailGenerator) GenerateBatch(ctx context.Context, limit int) (int, error) {
	rows, err := t.db.Query(
		ctx,
		`SELECT id FROM models WHERE thumbnail_path IS NULL LIMIT $1`,
		limit,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	generated := 0
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			continue
		}

		if _, err := t.GenerateThumbnail(ctx, id); err == nil {
			generated++
		}
	}

	return generated, nil
}

// CleanOrphaned removes thumbnails that don't have corresponding models
func (t *ThumbnailGenerator) CleanOrphaned(ctx context.Context) (int, error) {
	entries, err := os.ReadDir(t.thumbnailPath)
	if err != nil {
		return 0, err
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Check if thumbnail has a corresponding model
		var exists bool
		err := t.db.QueryRow(
			ctx,
			"SELECT EXISTS(SELECT 1 FROM models WHERE thumbnail_path = $1)",
			name,
		).Scan(&exists)

		if err != nil || !exists {
			if err := os.Remove(filepath.Join(t.thumbnailPath, name)); err == nil {
				removed++
			}
		}
	}

	return removed, nil
}

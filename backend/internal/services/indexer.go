package services

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/optics/backend/internal/models"
)

type Indexer struct {
	db *pgxpool.Pool
}

func NewIndexer(db *pgxpool.Pool) *Indexer {
	return &Indexer{db: db}
}

func (i *Indexer) IndexModel(ctx context.Context, create *models.ModelCreate) (*models.Model, error) {
	name := create.Name
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(create.FilePath), filepath.Ext(create.FilePath))
	}

	slug := slugify(name + "-" + uuid.New().String()[:8])

	var model models.Model
	err := i.db.QueryRow(
		ctx,
		`INSERT INTO models (name, slug, description, file_path, file_type, file_size, file_hash, source, source_id, license, author, category_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id, name, slug, description, file_path, file_type, file_size, file_hash, source, source_id, license, author, category_id, indexed_at, created_at, updated_at`,
		name, slug, create.Description, create.FilePath, create.FileType,
		create.FileSize, create.FileHash, create.Source, create.SourceID,
		create.License, create.Author, create.CategoryID,
	).Scan(
		&model.ID, &model.Name, &model.Slug, &model.Description, &model.FilePath,
		&model.FileType, &model.FileSize, &model.FileHash, &model.Source, &model.SourceID,
		&model.License, &model.Author, &model.CategoryID, &model.IndexedAt, &model.CreatedAt, &model.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &model, nil
}

func (i *Indexer) BulkIndex(ctx context.Context, models []*models.ModelCreate) (int, error) {
	indexed := 0
	for _, m := range models {
		_, err := i.IndexModel(ctx, m)
		if err == nil {
			indexed++
		}
	}
	return indexed, nil
}

func (i *Indexer) UpdateModelMetadata(ctx context.Context, id uuid.UUID, update *models.ModelUpdate) error {
	query := `UPDATE models SET updated_at = NOW()`
	args := []interface{}{}
	argNum := 1

	if update.Name != nil {
		query += `, name = $` + string(rune('0'+argNum))
		args = append(args, *update.Name)
		argNum++

		query += `, slug = $` + string(rune('0'+argNum))
		args = append(args, slugify(*update.Name))
		argNum++
	}

	if update.Description != nil {
		query += `, description = $` + string(rune('0'+argNum))
		args = append(args, *update.Description)
		argNum++
	}

	if update.License != nil {
		query += `, license = $` + string(rune('0'+argNum))
		args = append(args, *update.License)
		argNum++
	}

	if update.Author != nil {
		query += `, author = $` + string(rune('0'+argNum))
		args = append(args, *update.Author)
		argNum++
	}

	if update.CategoryID != nil {
		query += `, category_id = $` + string(rune('0'+argNum))
		args = append(args, *update.CategoryID)
		argNum++
	}

	query += ` WHERE id = $` + string(rune('0'+argNum))
	args = append(args, id)

	_, err := i.db.Exec(ctx, query, args...)
	return err
}

func (i *Indexer) DeleteModel(ctx context.Context, id uuid.UUID) error {
	_, err := i.db.Exec(ctx, "DELETE FROM models WHERE id = $1", id)
	return err
}

func (i *Indexer) GetModelByPath(ctx context.Context, path string) (*models.Model, error) {
	var model models.Model
	err := i.db.QueryRow(
		ctx,
		`SELECT id, name, slug, description, file_path, file_type, file_size, file_hash,
		 thumbnail_path, source, source_id, license, author, category_id, indexed_at, created_at, updated_at
		 FROM models WHERE file_path = $1`,
		path,
	).Scan(
		&model.ID, &model.Name, &model.Slug, &model.Description, &model.FilePath,
		&model.FileType, &model.FileSize, &model.FileHash, &model.ThumbnailPath,
		&model.Source, &model.SourceID, &model.License, &model.Author, &model.CategoryID,
		&model.IndexedAt, &model.CreatedAt, &model.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &model, nil
}

func (i *Indexer) CheckExists(ctx context.Context, path string) (bool, error) {
	var exists bool
	err := i.db.QueryRow(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM models WHERE file_path = $1)",
		path,
	).Scan(&exists)
	return exists, err
}

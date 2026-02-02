package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/optics/backend/internal/models"
)

type DuplicateDetector struct {
	db *pgxpool.Pool
}

func NewDuplicateDetector(db *pgxpool.Pool) *DuplicateDetector {
	return &DuplicateDetector{db: db}
}

// DetectByHash finds duplicates based on file hash
func (d *DuplicateDetector) DetectByHash(ctx context.Context) (int, error) {
	// Find all hashes with multiple models
	rows, err := d.db.Query(ctx, `
		SELECT file_hash, array_agg(id ORDER BY created_at) as model_ids
		FROM models
		WHERE file_hash IS NOT NULL AND file_hash != ''
		GROUP BY file_hash
		HAVING COUNT(*) > 1
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	groupsCreated := 0
	for rows.Next() {
		var hash string
		var modelIDs []uuid.UUID
		if err := rows.Scan(&hash, &modelIDs); err != nil {
			continue
		}

		if len(modelIDs) < 2 {
			continue
		}

		// Check if a group already exists for these models
		var existingGroup uuid.UUID
		err := d.db.QueryRow(ctx, `
			SELECT dg.id FROM duplicate_groups dg
			JOIN duplicate_members dm ON dm.group_id = dg.id
			WHERE dm.model_id = $1
			LIMIT 1
		`, modelIDs[0]).Scan(&existingGroup)

		if err == nil {
			// Group already exists, skip
			continue
		}

		// Create new duplicate group
		var groupID uuid.UUID
		err = d.db.QueryRow(ctx,
			"INSERT INTO duplicate_groups (primary_model_id) VALUES ($1) RETURNING id",
			modelIDs[0],
		).Scan(&groupID)
		if err != nil {
			continue
		}

		// Add all models to the group
		for _, modelID := range modelIDs {
			d.db.Exec(ctx,
				"INSERT INTO duplicate_members (group_id, model_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
				groupID, modelID,
			)
		}

		groupsCreated++
	}

	return groupsCreated, nil
}

// GetDuplicateGroups returns all duplicate groups with their models
func (d *DuplicateDetector) GetDuplicateGroups(ctx context.Context, limit, offset int) ([]models.DuplicateGroup, int, error) {
	// Get total count
	var total int
	err := d.db.QueryRow(ctx, "SELECT COUNT(*) FROM duplicate_groups").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get groups
	rows, err := d.db.Query(ctx, `
		SELECT id, primary_model_id, created_at
		FROM duplicate_groups
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var groups []models.DuplicateGroup
	for rows.Next() {
		var group models.DuplicateGroup
		if err := rows.Scan(&group.ID, &group.PrimaryModelID, &group.CreatedAt); err != nil {
			continue
		}

		// Get members
		memberRows, err := d.db.Query(ctx, `
			SELECT m.id, m.name, m.slug, m.file_path, m.file_type, m.file_size, m.thumbnail_path, m.created_at
			FROM models m
			JOIN duplicate_members dm ON dm.model_id = m.id
			WHERE dm.group_id = $1
			ORDER BY m.created_at
		`, group.ID)
		if err == nil {
			for memberRows.Next() {
				var m models.Model
				if memberRows.Scan(&m.ID, &m.Name, &m.Slug, &m.FilePath, &m.FileType, &m.FileSize, &m.ThumbnailPath, &m.CreatedAt) == nil {
					group.Models = append(group.Models, m)
				}
			}
			memberRows.Close()
		}

		groups = append(groups, group)
	}

	return groups, total, nil
}

// ResolveDuplicateGroup resolves a duplicate group by keeping the primary model
func (d *DuplicateDetector) ResolveDuplicateGroup(ctx context.Context, groupID uuid.UUID, keepModelID uuid.UUID, deleteFromDB bool) error {
	if deleteFromDB {
		// Delete all models except the one to keep
		_, err := d.db.Exec(ctx, `
			DELETE FROM models
			WHERE id IN (
				SELECT model_id FROM duplicate_members WHERE group_id = $1
			) AND id != $2
		`, groupID, keepModelID)
		if err != nil {
			return err
		}
	}

	// Delete the duplicate group (members will be cascade deleted)
	_, err := d.db.Exec(ctx, "DELETE FROM duplicate_groups WHERE id = $1", groupID)
	return err
}

// MergeModels merges metadata from duplicate models into one
func (d *DuplicateDetector) MergeModels(ctx context.Context, targetID uuid.UUID, sourceIDs []uuid.UUID) error {
	for _, sourceID := range sourceIDs {
		if sourceID == targetID {
			continue
		}

		// Get source model
		var source models.Model
		err := d.db.QueryRow(ctx, `
			SELECT description, license, author, category_id
			FROM models WHERE id = $1
		`, sourceID).Scan(&source.Description, &source.License, &source.Author, &source.CategoryID)
		if err != nil {
			continue
		}

		// Transfer tags
		d.db.Exec(ctx, `
			INSERT INTO model_tags (model_id, tag_id)
			SELECT $1, tag_id FROM model_tags WHERE model_id = $2
			ON CONFLICT DO NOTHING
		`, targetID, sourceID)

		// Update target with non-null values from source
		if source.Description != nil {
			d.db.Exec(ctx, `
				UPDATE models SET description = COALESCE(description, $1) WHERE id = $2
			`, *source.Description, targetID)
		}
		if source.License != nil {
			d.db.Exec(ctx, `
				UPDATE models SET license = COALESCE(license, $1) WHERE id = $2
			`, *source.License, targetID)
		}
		if source.Author != nil {
			d.db.Exec(ctx, `
				UPDATE models SET author = COALESCE(author, $1) WHERE id = $2
			`, *source.Author, targetID)
		}
		if source.CategoryID != nil {
			d.db.Exec(ctx, `
				UPDATE models SET category_id = COALESCE(category_id, $1) WHERE id = $2
			`, *source.CategoryID, targetID)
		}
	}

	return nil
}

// CleanEmptyGroups removes duplicate groups with no members
func (d *DuplicateDetector) CleanEmptyGroups(ctx context.Context) (int, error) {
	result, err := d.db.Exec(ctx, `
		DELETE FROM duplicate_groups
		WHERE id NOT IN (SELECT DISTINCT group_id FROM duplicate_members)
	`)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}

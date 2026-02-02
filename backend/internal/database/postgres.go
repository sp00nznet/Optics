package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/optics/backend/internal/config"
)

type DB struct {
	Pool *pgxpool.Pool
}

func New(cfg *config.DatabaseConfig) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}

func (db *DB) RunMigrations(ctx context.Context) error {
	migrations := []string{
		migrationUsers,
		migrationCategories,
		migrationTags,
		migrationModels,
		migrationModelTags,
		migrationFavorites,
		migrationCollections,
		migrationCollectionModels,
		migrationDuplicateGroups,
		migrationDuplicateMembers,
		migrationIndexes,
		migrationSettings,
	}

	for i, migration := range migrations {
		if _, err := db.Pool.Exec(ctx, migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	return nil
}

const migrationUsers = `
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
`

const migrationCategories = `
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    parent_id UUID REFERENCES categories(id),
    description TEXT,
    icon VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);
`

const migrationTags = `
CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    color VARCHAR(7) DEFAULT '#6366f1'
);
`

const migrationModels = `
CREATE TABLE IF NOT EXISTS models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    file_path VARCHAR(1000) NOT NULL,
    file_type VARCHAR(10) NOT NULL,
    file_size BIGINT,
    file_hash VARCHAR(64),
    thumbnail_path VARCHAR(500),
    source VARCHAR(50),
    source_id VARCHAR(100),
    license VARCHAR(100),
    author VARCHAR(255),
    category_id UUID REFERENCES categories(id),
    indexed_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
`

const migrationModelTags = `
CREATE TABLE IF NOT EXISTS model_tags (
    model_id UUID REFERENCES models(id) ON DELETE CASCADE,
    tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (model_id, tag_id)
);
`

const migrationFavorites = `
CREATE TABLE IF NOT EXISTS favorites (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    model_id UUID REFERENCES models(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (user_id, model_id)
);
`

const migrationCollections = `
CREATE TABLE IF NOT EXISTS collections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_public BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW()
);
`

const migrationCollectionModels = `
CREATE TABLE IF NOT EXISTS collection_models (
    collection_id UUID REFERENCES collections(id) ON DELETE CASCADE,
    model_id UUID REFERENCES models(id) ON DELETE CASCADE,
    added_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (collection_id, model_id)
);
`

const migrationDuplicateGroups = `
CREATE TABLE IF NOT EXISTS duplicate_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    primary_model_id UUID REFERENCES models(id),
    created_at TIMESTAMP DEFAULT NOW()
);
`

const migrationDuplicateMembers = `
CREATE TABLE IF NOT EXISTS duplicate_members (
    group_id UUID REFERENCES duplicate_groups(id) ON DELETE CASCADE,
    model_id UUID REFERENCES models(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, model_id)
);
`

const migrationIndexes = `
CREATE INDEX IF NOT EXISTS idx_models_category ON models(category_id);
CREATE INDEX IF NOT EXISTS idx_models_file_hash ON models(file_hash);
CREATE INDEX IF NOT EXISTS idx_models_file_type ON models(file_type);
CREATE INDEX IF NOT EXISTS idx_models_name ON models USING gin(to_tsvector('english', name));
CREATE INDEX IF NOT EXISTS idx_models_slug ON models(slug);
CREATE INDEX IF NOT EXISTS idx_categories_parent ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);
CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_collections_user ON collections(user_id);
`

const migrationSettings = `
CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);
`

package services

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/optics/backend/internal/config"
)

type ScanStatus struct {
	Running       bool      `json:"running"`
	StartedAt     time.Time `json:"started_at,omitempty"`
	FilesScanned  int       `json:"files_scanned"`
	FilesIndexed  int       `json:"files_indexed"`
	FilesSkipped  int       `json:"files_skipped"`
	Errors        int       `json:"errors"`
	CurrentPath   string    `json:"current_path,omitempty"`
	LastError     string    `json:"last_error,omitempty"`
	CompletedAt   time.Time `json:"completed_at,omitempty"`
}

type Scanner struct {
	db            *pgxpool.Pool
	cfg           *config.ScanningConfig
	storageCfg    *config.StorageConfig
	status        ScanStatus
	mu            sync.RWMutex
	stopChan      chan struct{}
	categoryCache map[string]uuid.UUID // folder path -> category ID
}

func NewScanner(db *pgxpool.Pool, scanCfg *config.ScanningConfig, storageCfg *config.StorageConfig) *Scanner {
	return &Scanner{
		db:            db,
		cfg:           scanCfg,
		storageCfg:    storageCfg,
		stopChan:      make(chan struct{}),
		categoryCache: make(map[string]uuid.UUID),
	}
}

func (s *Scanner) Start() {
	s.StartMultiple([]string{s.storageCfg.MountPath})
}

func (s *Scanner) StartMultiple(paths []string) {
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return
	}
	s.status = ScanStatus{
		Running:   true,
		StartedAt: time.Now(),
	}
	s.stopChan = make(chan struct{})
	s.categoryCache = make(map[string]uuid.UUID)
	s.mu.Unlock()

	s.scanMultiple(paths)
}

func (s *Scanner) scanMultiple(paths []string) {
	defer func() {
		s.mu.Lock()
		s.status.Running = false
		s.status.CompletedAt = time.Now()
		s.mu.Unlock()
	}()

	extensions := make(map[string]bool)
	for _, ext := range s.cfg.Extensions {
		extensions["."+strings.ToLower(ext)] = true
	}

	for _, rootPath := range paths {
		select {
		case <-s.stopChan:
			return
		default:
		}
		s.scanPath(rootPath, extensions)
	}
}

func (s *Scanner) scanPath(rootPath string, extensions map[string]bool) {
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		select {
		case <-s.stopChan:
			return filepath.SkipAll
		default:
		}

		if err != nil {
			s.incrementError(err.Error())
			return nil
		}

		// Skip directories that match ignore patterns
		if info.IsDir() {
			for _, pattern := range s.cfg.IgnorePatterns {
				if matched, _ := filepath.Match(pattern, info.Name()); matched {
					return filepath.SkipDir
				}
			}
			return nil
		}

		s.mu.Lock()
		s.status.FilesScanned++
		s.status.CurrentPath = path
		s.mu.Unlock()

		ext := strings.ToLower(filepath.Ext(info.Name()))

		// Handle ZIP files
		if ext == ".zip" {
			s.scanZipFile(path, rootPath, extensions)
			return nil
		}

		// Check extension for regular files
		if !extensions[ext] {
			s.mu.Lock()
			s.status.FilesSkipped++
			s.mu.Unlock()
			return nil
		}

		// Check if already indexed
		relPath, _ := filepath.Rel(rootPath, path)
		relPath = filepath.ToSlash(relPath)

		var exists bool
		err = s.db.QueryRow(
			context.Background(),
			"SELECT EXISTS(SELECT 1 FROM models WHERE file_path = $1)",
			relPath,
		).Scan(&exists)

		if err != nil || exists {
			s.mu.Lock()
			s.status.FilesSkipped++
			s.mu.Unlock()
			return nil
		}

		// Get or create category from folder structure
		categoryID := s.getOrCreateCategory(relPath)

		// Index the file
		if err := s.indexFile(path, relPath, info.Name(), info.Size(), categoryID, nil); err != nil {
			s.incrementError(err.Error())
			return nil
		}

		s.mu.Lock()
		s.status.FilesIndexed++
		s.mu.Unlock()

		return nil
	})

	if err != nil {
		s.incrementError(err.Error())
	}
}

func (s *Scanner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.Running {
		close(s.stopChan)
		s.status.Running = false
	}
}

func (s *Scanner) GetStatus() ScanStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}


func (s *Scanner) scanZipFile(zipPath, rootPath string, extensions map[string]bool) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		s.incrementError("Failed to open ZIP: " + err.Error())
		return
	}
	defer reader.Close()

	relZipPath, _ := filepath.Rel(rootPath, zipPath)
	relZipPath = filepath.ToSlash(relZipPath)

	// Try to read README.txt for Thingiverse info
	var thingiverseURL string
	for _, file := range reader.File {
		if strings.ToLower(filepath.Base(file.Name)) == "readme.txt" {
			if rc, err := file.Open(); err == nil {
				scanner := bufio.NewScanner(rc)
				for scanner.Scan() {
					line := scanner.Text()
					if strings.Contains(line, "thingiverse.com/thing:") {
						thingiverseURL = line
						break
					}
				}
				rc.Close()
			}
			break
		}
	}

	// Get category from ZIP's folder structure
	categoryID := s.getOrCreateCategory(relZipPath)

	for _, file := range reader.File {
		select {
		case <-s.stopChan:
			return
		default:
		}

		if file.FileInfo().IsDir() {
			continue
		}

		s.mu.Lock()
		s.status.FilesScanned++
		s.status.CurrentPath = zipPath + "!" + file.Name
		s.mu.Unlock()

		ext := strings.ToLower(filepath.Ext(file.Name))
		if !extensions[ext] {
			s.mu.Lock()
			s.status.FilesSkipped++
			s.mu.Unlock()
			continue
		}

		// Create a virtual path: zipfile.zip!path/inside/file.stl
		virtualPath := relZipPath + "!" + filepath.ToSlash(file.Name)

		// Check if already indexed
		var exists bool
		err := s.db.QueryRow(
			context.Background(),
			"SELECT EXISTS(SELECT 1 FROM models WHERE file_path = $1)",
			virtualPath,
		).Scan(&exists)

		if err != nil || exists {
			s.mu.Lock()
			s.status.FilesSkipped++
			s.mu.Unlock()
			continue
		}

		// Calculate hash from ZIP content
		fileHash := ""
		rc, err := file.Open()
		if err == nil {
			hash := sha256.New()
			io.Copy(hash, rc)
			fileHash = hex.EncodeToString(hash.Sum(nil))
			rc.Close()
		}

		// Index the file
		fileName := filepath.Base(file.Name)
		if err := s.indexZipFile(virtualPath, fileName, int64(file.UncompressedSize64), fileHash, categoryID, thingiverseURL); err != nil {
			s.incrementError(err.Error())
			continue
		}

		s.mu.Lock()
		s.status.FilesIndexed++
		s.mu.Unlock()
	}
}

func (s *Scanner) getOrCreateCategory(relPath string) *uuid.UUID {
	// Get the parent folder as category
	dir := filepath.Dir(relPath)
	if dir == "." || dir == "" {
		return nil
	}

	// Use the first-level folder as main category
	parts := strings.Split(filepath.ToSlash(dir), "/")
	if len(parts) == 0 {
		return nil
	}

	categoryName := parts[0]
	// Clean up the category name (replace + with space, etc.)
	categoryName = strings.ReplaceAll(categoryName, "+", " ")
	categoryName = strings.ReplaceAll(categoryName, "_", " ")
	categoryName = strings.TrimSpace(categoryName)

	if categoryName == "" {
		return nil
	}

	// Check cache first
	if id, ok := s.categoryCache[categoryName]; ok {
		return &id
	}

	// Check if category exists in database
	var categoryID uuid.UUID
	err := s.db.QueryRow(
		context.Background(),
		"SELECT id FROM categories WHERE LOWER(name) = LOWER($1)",
		categoryName,
	).Scan(&categoryID)

	if err == nil {
		s.categoryCache[categoryName] = categoryID
		return &categoryID
	}

	// Create new category
	slug := slugify(categoryName)
	err = s.db.QueryRow(
		context.Background(),
		`INSERT INTO categories (name, slug) VALUES ($1, $2)
		 ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id`,
		categoryName, slug,
	).Scan(&categoryID)

	if err != nil {
		return nil
	}

	s.categoryCache[categoryName] = categoryID
	return &categoryID
}

func (s *Scanner) indexFile(fullPath, relPath, fileName string, fileSize int64, categoryID *uuid.UUID, thingiverseURL *string) error {
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	// Clean up name
	name = strings.ReplaceAll(name, "+", " ")
	name = strings.ReplaceAll(name, "_", " ")
	slug := slugify(name + "-" + uuid.New().String()[:8])
	fileType := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")

	// Calculate file hash
	fileHash, err := s.calculateHash(fullPath)
	if err != nil {
		fileHash = ""
	}

	// Try to extract source info from path
	source, sourceID := s.extractSourceInfo(relPath, thingiverseURL)

	_, err = s.db.Exec(
		context.Background(),
		`INSERT INTO models (name, slug, file_path, file_type, file_size, file_hash, source, source_id, category_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		name, slug, relPath, fileType, fileSize, nullIfEmpty(fileHash), source, sourceID, categoryID,
	)

	return err
}

func (s *Scanner) indexZipFile(virtualPath, fileName string, fileSize int64, fileHash string, categoryID *uuid.UUID, thingiverseURL string) error {
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	// Clean up name
	name = strings.ReplaceAll(name, "+", " ")
	name = strings.ReplaceAll(name, "_", " ")
	slug := slugify(name + "-" + uuid.New().String()[:8])
	fileType := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")

	// Try to extract source info from path
	source, sourceID := s.extractSourceInfo(virtualPath, &thingiverseURL)

	_, err := s.db.Exec(
		context.Background(),
		`INSERT INTO models (name, slug, file_path, file_type, file_size, file_hash, source, source_id, category_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		name, slug, virtualPath, fileType, fileSize, nullIfEmpty(fileHash), source, sourceID, categoryID,
	)

	return err
}

func (s *Scanner) calculateHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *Scanner) extractSourceInfo(relPath string, thingiverseURL *string) (*string, *string) {
	// Try to extract from README URL first
	if thingiverseURL != nil && *thingiverseURL != "" {
		thingRe := regexp.MustCompile(`thing[:\s]+(\d+)`)
		if matches := thingRe.FindStringSubmatch(*thingiverseURL); len(matches) > 1 {
			source := "thingiverse"
			sourceID := matches[1]
			return &source, &sourceID
		}
	}

	// Check for Thing ID pattern in filename: 1528_Tardis.zip or thing_12345
	thingIDRe := regexp.MustCompile(`^(\d{4,})[-_]`)
	parts := strings.Split(relPath, "/")
	for _, part := range parts {
		if matches := thingIDRe.FindStringSubmatch(part); len(matches) > 1 {
			source := "thingiverse"
			sourceID := matches[1]
			return &source, &sourceID
		}
	}

	// Check for Thingiverse pattern in path: e.g., "thingiverse/12345"
	thingiverseRe := regexp.MustCompile(`(?i)thingiverse.*?(\d{4,})`)
	if matches := thingiverseRe.FindStringSubmatch(relPath); len(matches) > 1 {
		source := "thingiverse"
		sourceID := matches[1]
		return &source, &sourceID
	}

	return nil, nil
}

func (s *Scanner) incrementError(msg string) {
	s.mu.Lock()
	s.status.Errors++
	s.status.LastError = msg
	s.mu.Unlock()
}

func slugify(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

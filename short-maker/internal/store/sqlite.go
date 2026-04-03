// internal/store/sqlite.go
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/west-garden/short-maker/internal/domain"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		style TEXT NOT NULL,
		episode_count INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'created',
		created_at DATETIME NOT NULL
	);
	CREATE TABLE IF NOT EXISTS assets (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		scope TEXT NOT NULL,
		project_id TEXT DEFAULT '',
		file_path TEXT DEFAULT '',
		tags TEXT DEFAULT '[]',
		metadata TEXT DEFAULT '{}',
		created_at DATETIME NOT NULL
	);
	CREATE TABLE IF NOT EXISTS blueprints (
		project_id TEXT PRIMARY KEY,
		data TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStore) SaveProject(ctx context.Context, p *domain.Project) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO projects (id, name, style, episode_count, status, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		p.ID, p.Name, p.Style, p.EpisodeCount, p.Status, p.CreatedAt)
	return err
}

func (s *SQLiteStore) GetProject(ctx context.Context, id string) (*domain.Project, error) {
	p := &domain.Project{}
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, style, episode_count, status, created_at FROM projects WHERE id = ?", id).
		Scan(&p.ID, &p.Name, &p.Style, &p.EpisodeCount, &p.Status, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get project %s: %w", id, err)
	}
	return p, nil
}

func (s *SQLiteStore) UpdateProjectStatus(ctx context.Context, id string, status domain.Status) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE projects SET status = ? WHERE id = ?", status, id)
	return err
}

func (s *SQLiteStore) SaveAsset(ctx context.Context, a *domain.Asset) error {
	tagsJSON, _ := json.Marshal(a.Tags)
	metaJSON, _ := json.Marshal(a.Metadata)
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO assets (id, name, type, scope, project_id, file_path, tags, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		a.ID, a.Name, a.Type, a.Scope, a.ProjectID, a.FilePath, string(tagsJSON), string(metaJSON), a.CreatedAt)
	return err
}

func (s *SQLiteStore) GetAsset(ctx context.Context, id string) (*domain.Asset, error) {
	a := &domain.Asset{}
	var tagsJSON, metaJSON string
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, type, scope, project_id, file_path, tags, metadata, created_at FROM assets WHERE id = ?", id).
		Scan(&a.ID, &a.Name, &a.Type, &a.Scope, &a.ProjectID, &a.FilePath, &tagsJSON, &metaJSON, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get asset %s: %w", id, err)
	}
	json.Unmarshal([]byte(tagsJSON), &a.Tags)
	json.Unmarshal([]byte(metaJSON), &a.Metadata)
	return a, nil
}

func (s *SQLiteStore) ListAssets(ctx context.Context, scope domain.AssetScope, projectID string, assetType domain.AssetType) ([]*domain.Asset, error) {
	query := "SELECT id, name, type, scope, project_id, file_path, tags, metadata, created_at FROM assets WHERE scope = ? AND type = ?"
	args := []any{scope, assetType}
	if projectID != "" {
		query += " AND project_id = ?"
		args = append(args, projectID)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*domain.Asset
	for rows.Next() {
		a := &domain.Asset{}
		var tagsJSON, metaJSON string
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Scope, &a.ProjectID, &a.FilePath, &tagsJSON, &metaJSON, &a.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &a.Tags)
		json.Unmarshal([]byte(metaJSON), &a.Metadata)
		assets = append(assets, a)
	}
	return assets, nil
}

func (s *SQLiteStore) SearchAssets(ctx context.Context, scope domain.AssetScope, tags []string) ([]*domain.Asset, error) {
	query := "SELECT id, name, type, scope, project_id, file_path, tags, metadata, created_at FROM assets WHERE scope = ?"
	args := []any{scope}
	for _, tag := range tags {
		query += " AND tags LIKE ?"
		args = append(args, "%"+tag+"%")
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*domain.Asset
	for rows.Next() {
		a := &domain.Asset{}
		var tagsJSON, metaJSON string
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Scope, &a.ProjectID, &a.FilePath, &tagsJSON, &metaJSON, &a.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &a.Tags)
		json.Unmarshal([]byte(metaJSON), &a.Metadata)
		assets = append(assets, a)
	}
	return assets, nil
}

func (s *SQLiteStore) SaveBlueprint(ctx context.Context, bp *domain.StoryBlueprint) error {
	data, err := json.Marshal(bp)
	if err != nil {
		return fmt.Errorf("marshal blueprint: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO blueprints (project_id, data) VALUES (?, ?)",
		bp.ProjectID, string(data))
	return err
}

func (s *SQLiteStore) GetBlueprint(ctx context.Context, projectID string) (*domain.StoryBlueprint, error) {
	var data string
	err := s.db.QueryRowContext(ctx,
		"SELECT data FROM blueprints WHERE project_id = ?", projectID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("get blueprint for %s: %w", projectID, err)
	}
	bp := &domain.StoryBlueprint{}
	if err := json.Unmarshal([]byte(data), bp); err != nil {
		return nil, fmt.Errorf("unmarshal blueprint: %w", err)
	}
	return bp, nil
}

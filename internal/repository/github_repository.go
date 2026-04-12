package repository

import (
	"context"
	"database/sql"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/model"
)

type GitHubRepository interface {
	Create(ctx context.Context, repo *model.GitHubRepository) error
	FindByFullName(ctx context.Context, fullName string) (*model.GitHubRepository, error)
	GetByID(ctx context.Context, id int64) (*model.GitHubRepository, error)
	UpdateLastSeenTag(ctx context.Context, repoID int64, tag string, releaseURL string) error
}

type githubRepository struct {
	db *sql.DB
}

func NewGitHubRepository(db *sql.DB) GitHubRepository {
	return &githubRepository{db: db}
}

func (r *githubRepository) Create(ctx context.Context, repo *model.GitHubRepository) error {
	query := `
		INSERT INTO repositories (full_name, owner, name)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx,
		query,
		repo.FullName,
		repo.Owner,
		repo.Name,
	).Scan(
		&repo.ID,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
}

func (r *githubRepository) FindByFullName(ctx context.Context, fullName string) (*model.GitHubRepository, error) {
	query := `
		SELECT id, full_name, owner, name, last_seen_tag, last_release_url, created_at, updated_at
		FROM repositories
		WHERE full_name = $1
	`

	var repo model.GitHubRepository

	err := r.db.QueryRowContext(ctx, query, fullName).Scan(
		&repo.ID,
		&repo.FullName,
		&repo.Owner,
		&repo.Name,
		&repo.LastSeenTag,
		&repo.LastReleaseURL,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &repo, nil
}

func (r *githubRepository) GetByID(ctx context.Context, id int64) (*model.GitHubRepository, error) {
	query := `
		SELECT id, full_name, owner, name, last_seen_tag, last_release_url, created_at, updated_at
		FROM repositories
		WHERE id = $1
	`

	var repo model.GitHubRepository

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&repo.ID,
		&repo.FullName,
		&repo.Owner,
		&repo.Name,
		&repo.LastSeenTag,
		&repo.LastReleaseURL,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &repo, nil
}

func (r *githubRepository) UpdateLastSeenTag(ctx context.Context, repoID int64, tag string, releaseURL string) error {
	query := `
		UPDATE repositories
		SET last_seen_tag = $1,
		    last_release_url = $2,
		    updated_at = NOW()
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, tag, releaseURL, repoID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

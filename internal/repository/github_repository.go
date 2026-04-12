package repository

import (
	"context"
	"database/sql"
	"errors"

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
	return nil, errors.New("not implemented")
}

func (r *githubRepository) GetByID(ctx context.Context, id int64) (*model.GitHubRepository, error) {
	return nil, errors.New("not implemented")
}

func (r *githubRepository) UpdateLastSeenTag(ctx context.Context, repoID int64, tag string, releaseURL string) error {
	return errors.New("not implemented")
}

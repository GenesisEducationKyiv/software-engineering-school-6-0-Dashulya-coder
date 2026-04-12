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
	return errors.New("not implemented")
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

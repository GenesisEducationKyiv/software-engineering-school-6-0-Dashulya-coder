package repository

import (
	"context"
	"database/sql"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repo"
)

type GitHubRepository interface {
	FindOrCreate(ctx context.Context, owner, name, fullName string) (*repo.Repository, error)
	FindByFullName(ctx context.Context, fullName string) (*repo.Repository, error)
	GetByID(ctx context.Context, id int64) (*repo.Repository, error)
	UpdateLastSeenTag(ctx context.Context, repoID int64, tag string, releaseURL string) error
}

type GitHubRepositoryImpl struct {
	db *sql.DB
}

func NewGitHubRepository(db *sql.DB) *GitHubRepositoryImpl {
	return &GitHubRepositoryImpl{db: db}
}

func (r *GitHubRepositoryImpl) FindOrCreate(
	ctx context.Context,
	owner, name, fullName string,
) (*repo.Repository, error) {
	existing, err := r.FindByFullName(ctx, fullName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	newRepo := &repo.Repository{
		FullName: fullName,
		Owner:    owner,
		Name:     name,
	}
	if err := r.Create(ctx, newRepo); err != nil {
		return nil, err
	}

	return newRepo, nil
}

func (r *GitHubRepositoryImpl) Create(ctx context.Context, ghRepo *repo.Repository) error {
	query := `
		INSERT INTO repositories (full_name, owner, name)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx,
		query,
		ghRepo.FullName,
		ghRepo.Owner,
		ghRepo.Name,
	).Scan(
		&ghRepo.ID,
		&ghRepo.CreatedAt,
		&ghRepo.UpdatedAt,
	)
}

func (r *GitHubRepositoryImpl) FindByFullName(
	ctx context.Context,
	fullName string,
) (*repo.Repository, error) {
	query := `
		SELECT id, full_name, owner, name, last_seen_tag, last_release_url, created_at, updated_at
		FROM repositories
		WHERE full_name = $1
	`
	return scanRepo(r.db.QueryRowContext(ctx, query, fullName))
}

func (r *GitHubRepositoryImpl) GetByID(ctx context.Context, id int64) (*repo.Repository, error) {
	query := `
		SELECT id, full_name, owner, name, last_seen_tag, last_release_url, created_at, updated_at
		FROM repositories
		WHERE id = $1
	`
	return scanRepo(r.db.QueryRowContext(ctx, query, id))
}

func scanRepo(row *sql.Row) (*repo.Repository, error) {
	var result repo.Repository
	err := row.Scan(
		&result.ID,
		&result.FullName,
		&result.Owner,
		&result.Name,
		&result.LastSeenTag,
		&result.LastReleaseURL,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

var _ GitHubRepository = (*GitHubRepositoryImpl)(nil)

func (r *GitHubRepositoryImpl) UpdateLastSeenTag(
	ctx context.Context,
	repoID int64,
	tag string,
	releaseURL string,
) error {
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
		return ErrNotFound
	}

	return nil
}

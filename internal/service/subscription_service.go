package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repository"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/urlbuilder"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/validator"
)

var (
	ErrInvalidEmail      = errors.New("invalid email")
	ErrInvalidRepo       = errors.New("invalid repo format")
	ErrRepoNotFound      = errors.New("repository not found")
	ErrAlreadySubscribed = errors.New("email already subscribed to this repository")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenNotFound     = errors.New("token not found")
)

type SubscriptionService interface {
	Subscribe(ctx context.Context, email, repo string) error
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	GetSubscriptionsByEmail(ctx context.Context, email string) ([]SubscriptionView, error)
}

type SubscriptionView struct {
	Email       string
	Repo        string
	Confirmed   bool
	LastSeenTag *string
}

type SubscriptionServiceImpl struct {
	subRepo  repository.SubscriptionRepository
	repoRepo repository.GitHubRepository
	ghClient github.Client
	mailer   mailer.Mailer
	urls     urlbuilder.URLBuilder
}

func NewSubscriptionService(
	subRepo repository.SubscriptionRepository,
	repoRepo repository.GitHubRepository,
	ghClient github.Client,
	m mailer.Mailer,
	urls urlbuilder.URLBuilder,
) *SubscriptionServiceImpl {
	return &SubscriptionServiceImpl{
		subRepo:  subRepo,
		repoRepo: repoRepo,
		ghClient: ghClient,
		mailer:   m,
		urls:     urls,
	}
}

func (s *SubscriptionServiceImpl) Subscribe(ctx context.Context, email, repo string) error {
	if err := validator.ValidateEmail(email); err != nil {
		return ErrInvalidEmail
	}

	if err := validator.ValidateRepo(repo); err != nil {
		return ErrInvalidRepo
	}

	owner, name := validator.ParseRepo(repo)

	exists, err := s.ghClient.RepositoryExists(ctx, owner, name)
	if err != nil {
		return err
	}
	if !exists {
		return ErrRepoNotFound
	}

	dbRepo, err := s.repoRepo.FindOrCreate(ctx, owner, name, repo)
	if err != nil {
		return err
	}

	existsSub, err := s.subRepo.ExistsByEmailAndRepo(ctx, email, dbRepo.ID)
	if err != nil {
		return err
	}
	if existsSub {
		return ErrAlreadySubscribed
	}

	sub := &subscription.Subscription{
		Email:            email,
		RepositoryID:     dbRepo.ID,
		ConfirmToken:     uuid.NewString(),
		UnsubscribeToken: uuid.NewString(),
	}

	if err := s.subRepo.Create(ctx, sub); err != nil {
		return err
	}

	return s.mailer.SendConfirmation(email, s.urls.ConfirmURL(sub.ConfirmToken))
}

func (s *SubscriptionServiceImpl) Confirm(ctx context.Context, token string) error {
	if token == "" {
		return ErrInvalidToken
	}

	sub, err := s.subRepo.FindByConfirmToken(ctx, token)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrTokenNotFound
	}

	if sub.Confirmed {
		return nil
	}

	return s.subRepo.ConfirmByToken(ctx, token)
}

func (s *SubscriptionServiceImpl) Unsubscribe(ctx context.Context, token string) error {
	if token == "" {
		return ErrInvalidToken
	}

	sub, err := s.subRepo.FindByUnsubscribeToken(ctx, token)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrTokenNotFound
	}

	if !sub.Active {
		return nil
	}

	return s.subRepo.DeactivateByToken(ctx, token)
}

func (s *SubscriptionServiceImpl) GetSubscriptionsByEmail(
	ctx context.Context,
	email string,
) ([]SubscriptionView, error) {
	if err := validator.ValidateEmail(email); err != nil {
		return nil, ErrInvalidEmail
	}

	subs, err := s.subRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	result := make([]SubscriptionView, 0, len(subs))

	for _, sub := range subs {
		repo, err := s.repoRepo.GetByID(ctx, sub.RepositoryID)
		if err != nil {
			return nil, err
		}
		if repo == nil {
			continue
		}

		result = append(result, SubscriptionView{
			Email:       sub.Email,
			Repo:        repo.FullName,
			Confirmed:   sub.Confirmed,
			LastSeenTag: repo.LastSeenTag,
		})
	}

	return result, nil
}

var _ SubscriptionService = (*SubscriptionServiceImpl)(nil)

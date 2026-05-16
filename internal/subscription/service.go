package subscription

import (
	"context"
	"errors"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repo"
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

type SubscriptionStore interface {
	Create(ctx context.Context, sub *Subscription) error
	FindByConfirmToken(ctx context.Context, token string) (*Subscription, error)
	FindByUnsubscribeToken(ctx context.Context, token string) (*Subscription, error)
	GetByEmail(ctx context.Context, email string) ([]Subscription, error)
	ExistsByEmailAndRepo(ctx context.Context, email string, repoID int64) (bool, error)
	ConfirmByToken(ctx context.Context, token string) error
	DeactivateByToken(ctx context.Context, token string) error
}

type RepoStore interface {
	FindOrCreate(ctx context.Context, owner, name, fullName string) (*repo.Repository, error)
	GetByID(ctx context.Context, id int64) (*repo.Repository, error)
}

type SubscriptionView struct {
	Email       string
	Repo        string
	Confirmed   bool
	LastSeenTag *string
}

type TokenGenerator interface {
	Generate() (string, error)
}

type SubscriptionServiceImpl struct {
	subRepo  SubscriptionStore
	repoRepo RepoStore
	ghClient github.Client
	mailer   mailer.Mailer
	urls     urlbuilder.URLBuilder
	tokenGen TokenGenerator
}

func NewSubscriptionService(
	subRepo SubscriptionStore,
	repoRepo RepoStore,
	ghClient github.Client,
	m mailer.Mailer,
	urls urlbuilder.URLBuilder,
	tokenGen TokenGenerator,
) *SubscriptionServiceImpl {
	return &SubscriptionServiceImpl{
		subRepo:  subRepo,
		repoRepo: repoRepo,
		ghClient: ghClient,
		mailer:   m,
		urls:     urls,
		tokenGen: tokenGen,
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

	confirmToken, err := s.tokenGen.Generate()
	if err != nil {
		return err
	}

	unsubscribeToken, err := s.tokenGen.Generate()
	if err != nil {
		return err
	}

	sub := &Subscription{
		Email:            email,
		RepositoryID:     dbRepo.ID,
		ConfirmToken:     confirmToken,
		UnsubscribeToken: unsubscribeToken,
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


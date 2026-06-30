package subscription

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repo"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/saga"
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

var errAlreadyActive = errors.New("subscription already active")

type SubscriptionStore interface {
	CreateForSaga(ctx context.Context, sub *Subscription, sagaID string) (alreadyActive bool, err error)
	CancelBySaga(ctx context.Context, sagaID string) error
	FindByConfirmToken(ctx context.Context, token string) (*Subscription, error)
	FindByUnsubscribeToken(ctx context.Context, token string) (*Subscription, error)
	GetByEmail(ctx context.Context, email string) ([]Subscription, error)
	ConfirmByToken(ctx context.Context, token string) error
	DeactivateByToken(ctx context.Context, token string) error
}

type ConfirmationNotifier interface {
	ReserveConfirmation(ctx context.Context, sagaID, email, confirmURL string) error
	CommitConfirmation(ctx context.Context, sagaID string) error
	CancelConfirmation(ctx context.Context, sagaID string) error
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
	notifier ConfirmationNotifier
	urls     urlbuilder.URLBuilder
	tokenGen TokenGenerator
}

func NewSubscriptionService(
	subRepo SubscriptionStore,
	repoRepo RepoStore,
	ghClient github.Client,
	notifier ConfirmationNotifier,
	urls urlbuilder.URLBuilder,
	tokenGen TokenGenerator,
) *SubscriptionServiceImpl {
	return &SubscriptionServiceImpl{
		subRepo:  subRepo,
		repoRepo: repoRepo,
		ghClient: ghClient,
		notifier: notifier,
		urls:     urls,
		tokenGen: tokenGen,
	}
}

func (s *SubscriptionServiceImpl) Subscribe(ctx context.Context, email, fullName string) error {
	if err := validator.ValidateEmail(email); err != nil {
		return ErrInvalidEmail
	}

	if err := validator.ValidateRepo(fullName); err != nil {
		return ErrInvalidRepo
	}

	owner, name := validator.ParseRepo(fullName)

	exists, err := s.ghClient.RepositoryExists(ctx, owner, name)
	if err != nil {
		return err
	}
	if !exists {
		return ErrRepoNotFound
	}

	dbRepo, err := s.repoRepo.FindOrCreate(ctx, owner, name, fullName)
	if err != nil {
		return err
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

	sagaID := uuid.NewString()
	confirmURL := s.urls.ConfirmURL(sub.ConfirmToken)

	err = saga.Run(ctx,
		saga.Step{
			Name: "create-subscription",
			Action: func(ctx context.Context) error {
				active, err := s.subRepo.CreateForSaga(ctx, sub, sagaID)
				if err != nil {
					return err
				}
				if active {
					return errAlreadyActive
				}
				return nil
			},
			Compensation: func(ctx context.Context) error {
				return s.subRepo.CancelBySaga(ctx, sagaID)
			},
		},
		saga.Step{
			Name: "reserve-confirmation",
			Action: func(ctx context.Context) error {
				return s.notifier.ReserveConfirmation(ctx, sagaID, email, confirmURL)
			},
			Compensation: func(ctx context.Context) error {
				return s.notifier.CancelConfirmation(ctx, sagaID)
			},
		},
		saga.Step{
			Name: "commit-confirmation",
			Action: func(ctx context.Context) error {
				return s.notifier.CommitConfirmation(ctx, sagaID)
			},
		},
	)
	if err != nil {
		if errors.Is(err, errAlreadyActive) {
			return ErrAlreadySubscribed
		}
		return err
	}

	return nil
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
		r, err := s.repoRepo.GetByID(ctx, sub.RepositoryID)
		if err != nil {
			return nil, err
		}
		if r == nil {
			continue
		}

		result = append(result, SubscriptionView{
			Email:       sub.Email,
			Repo:        r.FullName,
			Confirmed:   sub.Confirmed,
			LastSeenTag: r.LastSeenTag,
		})
	}

	return result, nil
}

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/model"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repository"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/validator"
)

var (
	ErrInvalidEmail      = errors.New("invalid email")
	ErrInvalidRepo       = errors.New("invalid repo format")
	ErrRepoNotFound      = errors.New("repository not found")
	ErrAlreadySubscribed = errors.New("email already subscribed to this repository")
)

type SubscriptionService interface {
	Subscribe(ctx context.Context, email, repo string) error
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	GetSubscriptionsByEmail(ctx context.Context, email string) ([]model.Subscription, error)
}

type subscriptionService struct {
	subRepo  repository.SubscriptionRepository
	repoRepo repository.GitHubRepository
	ghClient github.Client
	mailer   mailer.Mailer
	baseURL  string
}

func NewSubscriptionService(
	subRepo repository.SubscriptionRepository,
	repoRepo repository.GitHubRepository,
	ghClient github.Client,
	m mailer.Mailer,
	baseURL string,
) SubscriptionService {
	return &subscriptionService{
		subRepo:  subRepo,
		repoRepo: repoRepo,
		ghClient: ghClient,
		mailer:   m,
		baseURL:  baseURL,
	}
}

func (s *subscriptionService) Subscribe(ctx context.Context, email, repo string) error {
	if err := validator.ValidateEmail(email); err != nil {
		return ErrInvalidEmail
	}

	if err := validator.ValidateRepo(repo); err != nil {
		return ErrInvalidRepo
	}

	parts := strings.Split(repo, "/")
	owner := parts[0]
	name := parts[1]

	exists, err := s.ghClient.RepositoryExists(ctx, owner, name)
	if err != nil {
		return err
	}
	if !exists {
		return ErrRepoNotFound
	}

	dbRepo, err := s.repoRepo.FindByFullName(ctx, repo)
	if err != nil {
		return err
	}

	if dbRepo == nil {
		newRepo := &model.GitHubRepository{
			FullName: repo,
			Owner:    owner,
			Name:     name,
		}

		if err := s.repoRepo.Create(ctx, newRepo); err != nil {
			return err
		}

		dbRepo = newRepo
	}

	existsSub, err := s.subRepo.ExistsByEmailAndRepo(ctx, email, dbRepo.ID)
	if err != nil {
		return err
	}
	if existsSub {
		return ErrAlreadySubscribed
	}

	confirmToken := uuid.NewString()
	unsubscribeToken := uuid.NewString()

	sub := &model.Subscription{
		Email:            email,
		RepositoryID:     dbRepo.ID,
		ConfirmToken:     confirmToken,
		UnsubscribeToken: unsubscribeToken,
	}

	if err := s.subRepo.Create(ctx, sub); err != nil {
		return err
	}

	confirmLink := fmt.Sprintf("%s/api/confirm/%s", s.baseURL, confirmToken)

	if err := s.mailer.SendConfirmation(email, confirmLink); err != nil {
		return err
	}

	return nil
}
func (s *subscriptionService) Confirm(ctx context.Context, token string) error {
	if token == "" {
		return errors.New("invalid token")
	}

	sub, err := s.subRepo.FindByConfirmToken(ctx, token)
	if err != nil {
		return err
	}
	if sub == nil {
		return errors.New("token not found")
	}

	if sub.Confirmed {
		return nil
	}

	if err := s.subRepo.ConfirmByToken(ctx, token); err != nil {
		return err
	}

	return nil
}
func (s *subscriptionService) Unsubscribe(ctx context.Context, token string) error {
	if token == "" {
		return errors.New("invalid token")
	}

	sub, err := s.subRepo.FindByUnsubscribeToken(ctx, token)
	if err != nil {
		return err
	}
	if sub == nil {
		return errors.New("token not found")
	}

	if !sub.Active {
		return nil
	}

	if err := s.subRepo.DeactivateByToken(ctx, token); err != nil {
		return err
	}

	return nil
}
func (s *subscriptionService) GetSubscriptionsByEmail(ctx context.Context, email string) ([]model.Subscription, error) {
	if err := validator.ValidateEmail(email); err != nil {
		return nil, ErrInvalidEmail
	}

	return s.subRepo.GetByEmail(ctx, email)
}

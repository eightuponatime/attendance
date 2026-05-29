package impl

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"attendance/config"
	"attendance/internal/domain"
	"attendance/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrInvalidGoogleProfile  = errors.New("invalid google profile")
	ErrEmailDomainNotAllowed = errors.New("email domain is not allowed")
)

type UsersService struct {
	cfg *config.Config
	rp  repository.UsersRepository
}

func NewUsersService(
	cfg *config.Config,
	rp repository.UsersRepository,
) *UsersService {
	return &UsersService{
		cfg: cfg,
		rp:  rp,
	}
}

func (s *UsersService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Users, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: id is empty", ErrInvalidGoogleProfile)
	}

	return s.rp.GetByID(ctx, id)
}

func (s *UsersService) GetByGoogleSub(ctx context.Context, googleSub string) (*domain.Users, error) {
	googleSub = strings.TrimSpace(googleSub)
	if googleSub == "" {
		return nil, fmt.Errorf("%w: google_sub is empty", ErrInvalidGoogleProfile)
	}

	return s.rp.GetByGoogleSub(ctx, googleSub)
}

func (s *UsersService) FindOrCreateFromGoogle(
	ctx context.Context,
	input domain.GoogleUserInput,
) (*domain.Users, error) {
	normalized, err := normalizeGoogleUserInput(input)
	if err != nil {
		return nil, err
	}

	if err := s.ensureCorporateDomain(normalized.Email); err != nil {
		return nil, err
	}

	return s.rp.CreateOrUpdateFromGoogle(ctx, normalized)
}

func normalizeGoogleUserInput(input domain.GoogleUserInput) (domain.GoogleUserInput, error) {
	normalized := domain.GoogleUserInput{
		GoogleSub: strings.TrimSpace(input.GoogleSub),
		Email:     strings.ToLower(strings.TrimSpace(input.Email)),
		FullName:  strings.TrimSpace(input.FullName),
	}

	if normalized.GoogleSub == "" {
		return domain.GoogleUserInput{}, fmt.Errorf("%w: google_sub is empty", ErrInvalidGoogleProfile)
	}
	if normalized.Email == "" {
		return domain.GoogleUserInput{}, fmt.Errorf("%w: email is empty", ErrInvalidGoogleProfile)
	}
	if normalized.FullName == "" {
		return domain.GoogleUserInput{}, fmt.Errorf("%w: full_name is empty", ErrInvalidGoogleProfile)
	}

	return normalized, nil
}

func (s *UsersService) ensureCorporateDomain(email string) error {
	domain := strings.ToLower(strings.TrimSpace(s.cfg.CorporateDomain))
	if domain == "" {
		return nil
	}

	domain = strings.TrimPrefix(domain, "@")
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.HasSuffix(email, "@"+domain) {
		return ErrEmailDomainNotAllowed
	}

	return nil
}

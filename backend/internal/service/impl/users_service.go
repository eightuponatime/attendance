package impl

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"attendance/config"
	"attendance/internal/domain"
	"attendance/internal/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidGoogleProfile  = errors.New("invalid google profile")
	ErrEmailDomainNotAllowed = errors.New("email domain is not allowed")
	ErrInvalidLocalAuth      = errors.New("invalid local auth")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrEmailAlreadyExists    = errors.New("email already exists")
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

	user, err := s.rp.GetByGoogleSub(ctx, normalized.GoogleSub)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	user, err = s.rp.GetByEmail(ctx, normalized.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("%w: register with email first", ErrInvalidCredentials)
	}

	return s.rp.LinkGoogleSub(ctx, user.Id, normalized.GoogleSub)
}

func (s *UsersService) RegisterLocal(
	ctx context.Context,
	input domain.LocalRegisterInput,
) (*domain.Users, error) {
	normalized, fullName, err := normalizeLocalRegisterInput(input)
	if err != nil {
		return nil, err
	}

	existing, err := s.rp.GetByEmail(ctx, normalized.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(normalized.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.rp.CreateLocal(ctx, normalized, string(hash), fullName)
}

func (s *UsersService) LoginLocal(
	ctx context.Context,
	input domain.LocalLoginInput,
) (*domain.Users, error) {
	normalized := domain.LocalLoginInput{
		Email:    strings.ToLower(strings.TrimSpace(input.Email)),
		Password: input.Password,
	}
	if normalized.Email == "" || normalized.Password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.rp.GetByEmail(ctx, normalized.Email)
	if err != nil {
		return nil, err
	}
	if user == nil || user.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(normalized.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func normalizeGoogleUserInput(input domain.GoogleUserInput) (domain.GoogleUserInput, error) {
	normalized := domain.GoogleUserInput{
		GoogleSub: strings.TrimSpace(input.GoogleSub),
		Email:     strings.ToLower(strings.TrimSpace(input.Email)),
		FullName:  normalizeNamePart(input.FullName),
	}

	if normalized.GoogleSub == "" {
		return domain.GoogleUserInput{}, fmt.Errorf("%w: google_sub is empty", ErrInvalidGoogleProfile)
	}
	if normalized.Email == "" {
		return domain.GoogleUserInput{}, fmt.Errorf("%w: email is empty", ErrInvalidGoogleProfile)
	}

	return normalized, nil
}

func normalizeLocalRegisterInput(input domain.LocalRegisterInput) (domain.LocalRegisterInput, string, error) {
	normalized := domain.LocalRegisterInput{
		Email:      strings.ToLower(strings.TrimSpace(input.Email)),
		Password:   input.Password,
		LastName:   normalizeNamePart(input.LastName),
		FirstName:  normalizeNamePart(input.FirstName),
		MiddleName: normalizeNamePart(input.MiddleName),
	}

	if _, err := mail.ParseAddress(normalized.Email); err != nil {
		return domain.LocalRegisterInput{}, "", fmt.Errorf("%w: email is invalid", ErrInvalidLocalAuth)
	}
	if len([]rune(normalized.Password)) < 6 {
		return domain.LocalRegisterInput{}, "", fmt.Errorf("%w: password is too short", ErrInvalidLocalAuth)
	}
	if normalized.LastName == "" {
		return domain.LocalRegisterInput{}, "", fmt.Errorf("%w: last_name is empty", ErrInvalidLocalAuth)
	}
	if normalized.FirstName == "" {
		return domain.LocalRegisterInput{}, "", fmt.Errorf("%w: first_name is empty", ErrInvalidLocalAuth)
	}

	fullName := normalized.LastName + " " + normalized.FirstName
	if normalized.MiddleName != "" {
		fullName += " " + normalized.MiddleName
	}

	return normalized, fullName, nil
}

func normalizeNamePart(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

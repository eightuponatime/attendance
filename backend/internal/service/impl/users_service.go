package impl

import (
	"context"
	"errors"
	"fmt"
	"log"
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
		s.authDebugf("google login normalize failed: input_email=%q input_sub=%q err=%v", input.Email, input.GoogleSub, err)
		return nil, err
	}
	s.authDebugf(
		"google login start: email=%q google_sub=%q full_name=%q",
		normalized.Email,
		normalized.GoogleSub,
		normalized.FullName,
	)

	user, err := s.rp.GetByGoogleSub(ctx, normalized.GoogleSub)
	if err != nil {
		s.authDebugf("google login get by sub failed: email=%q google_sub=%q err=%v", normalized.Email, normalized.GoogleSub, err)
		return nil, err
	}
	if user != nil {
		s.authDebugf(
			"google login matched by sub: email=%q google_sub=%q user_id=%s user_email=%q password_hash_present=%t",
			normalized.Email,
			normalized.GoogleSub,
			user.Id,
			user.Email,
			user.PasswordHash != nil && *user.PasswordHash != "",
		)
		return user, nil
	}

	user, err = s.rp.GetByEmail(ctx, normalized.Email)
	if err != nil {
		s.authDebugf("google login get by email failed: email=%q google_sub=%q err=%v", normalized.Email, normalized.GoogleSub, err)
		return nil, err
	}
	if user == nil {
		s.authDebugf("google login user not found: email=%q google_sub=%q", normalized.Email, normalized.GoogleSub)
		return nil, fmt.Errorf("%w: register with email first", ErrInvalidCredentials)
	}

	linked, err := s.rp.LinkGoogleSub(ctx, user.Id, normalized.GoogleSub)
	if err != nil {
		s.authDebugf("google login link failed: email=%q google_sub=%q user_id=%s err=%v", normalized.Email, normalized.GoogleSub, user.Id, err)
		return nil, err
	}
	s.authDebugf("google login linked: email=%q google_sub=%q user_id=%s", normalized.Email, normalized.GoogleSub, linked.Id)
	return linked, nil
}

func (s *UsersService) RegisterLocal(
	ctx context.Context,
	input domain.LocalRegisterInput,
) (*domain.Users, error) {
	normalized, fullName, err := normalizeLocalRegisterInput(input)
	if err != nil {
		s.authDebugf(
			"local register normalize failed: input_email=%q input_password=%q input_google_sub=%q input_last_name=%q input_first_name=%q input_middle_name=%q err=%v",
			input.Email,
			input.Password,
			input.GoogleSub,
			input.LastName,
			input.FirstName,
			input.MiddleName,
			err,
		)
		return nil, err
	}
	s.authDebugf(
		"local register start: email=%q password=%q google_sub=%q last_name=%q first_name=%q middle_name=%q full_name=%q",
		normalized.Email,
		normalized.Password,
		normalized.GoogleSub,
		normalized.LastName,
		normalized.FirstName,
		normalized.MiddleName,
		fullName,
	)

	existing, err := s.rp.GetByEmail(ctx, normalized.Email)
	if err != nil {
		s.authDebugf("local register get by email failed: email=%q err=%v", normalized.Email, err)
		return nil, err
	}
	if existing != nil {
		s.authDebugf("local register duplicate email: email=%q existing_user_id=%s", normalized.Email, existing.Id)
		return nil, ErrEmailAlreadyExists
	}

	passwordHash := ""
	if normalized.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(normalized.Password), bcrypt.DefaultCost)
		if err != nil {
			s.authDebugf("local register bcrypt generate failed: email=%q password=%q err=%v", normalized.Email, normalized.Password, err)
			return nil, err
		}
		passwordHash = string(hash)
	}
	s.authDebugf("local register password hash prepared: email=%q password=%q generated_password_hash=%q", normalized.Email, normalized.Password, passwordHash)

	user, err := s.rp.CreateLocal(ctx, normalized, passwordHash, fullName)
	if err != nil {
		s.authDebugf("local register create failed: email=%q google_sub=%q password_hash=%q err=%v", normalized.Email, normalized.GoogleSub, passwordHash, err)
		return nil, err
	}
	s.authDebugf(
		"local register success: email=%q user_id=%s google_sub=%q stored_password_hash=%q",
		user.Email,
		user.Id,
		stringPtrValue(user.GoogleSub),
		stringPtrValue(user.PasswordHash),
	)
	return user, nil
}

func (s *UsersService) LoginLocal(
	ctx context.Context,
	input domain.LocalLoginInput,
) (*domain.Users, error) {
	normalized := domain.LocalLoginInput{
		Email:    strings.ToLower(strings.TrimSpace(input.Email)),
		Password: input.Password,
	}
	s.authDebugf("local login start: input_email=%q normalized_email=%q password=%q", input.Email, normalized.Email, normalized.Password)
	if normalized.Email == "" || normalized.Password == "" {
		s.authDebugf("local login rejected empty field: email_empty=%t password_empty=%t", normalized.Email == "", normalized.Password == "")
		return nil, ErrInvalidCredentials
	}

	user, err := s.rp.GetByEmail(ctx, normalized.Email)
	if err != nil {
		s.authDebugf("local login get by email failed: email=%q err=%v", normalized.Email, err)
		return nil, err
	}
	if user == nil {
		s.authDebugf("local login user not found: email=%q", normalized.Email)
		return nil, ErrInvalidCredentials
	}
	if user.PasswordHash == nil || *user.PasswordHash == "" {
		s.authDebugf(
			"local login password missing on account: email=%q user_id=%s google_sub=%v password=%q",
			normalized.Email,
			user.Id,
			stringPtrValue(user.GoogleSub),
			normalized.Password,
		)
		return nil, ErrInvalidCredentials
	}
	inputHash, hashErr := bcrypt.GenerateFromPassword([]byte(normalized.Password), bcrypt.DefaultCost)
	if hashErr != nil {
		s.authDebugf("local login bcrypt sample hash failed: email=%q password=%q err=%v", normalized.Email, normalized.Password, hashErr)
	} else {
		s.authDebugf(
			"local login password comparison input: email=%q user_id=%s password=%q input_password_bcrypt_sample=%q stored_password_hash=%q",
			normalized.Email,
			user.Id,
			normalized.Password,
			string(inputHash),
			*user.PasswordHash,
		)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(normalized.Password)); err != nil {
		s.authDebugf(
			"local login bcrypt compare failed: email=%q user_id=%s password=%q stored_password_hash=%q err=%v",
			normalized.Email,
			user.Id,
			normalized.Password,
			*user.PasswordHash,
			err,
		)
		return nil, ErrInvalidCredentials
	}

	s.authDebugf("local login success: email=%q user_id=%s", normalized.Email, user.Id)
	return user, nil
}

func (s *UsersService) authDebugf(format string, args ...any) {
	if s == nil || s.cfg == nil || !s.cfg.AuthDebugLogCredentials {
		return
	}

	log.Printf("AUTH DEBUG: "+format, args...)
}

func stringPtrValue(value *string) string {
	if value == nil {
		return "<nil>"
	}

	return *value
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
		GoogleSub:  strings.TrimSpace(input.GoogleSub),
	}

	if _, err := mail.ParseAddress(normalized.Email); err != nil {
		return domain.LocalRegisterInput{}, "", fmt.Errorf("%w: email is invalid", ErrInvalidLocalAuth)
	}
	if normalized.GoogleSub == "" && len([]rune(normalized.Password)) < 6 {
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

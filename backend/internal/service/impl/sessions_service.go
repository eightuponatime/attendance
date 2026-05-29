package impl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"attendance/config"
	"attendance/internal/domain"
	"attendance/internal/repository"

	"github.com/google/uuid"
)

var ErrInvalidSession = errors.New("invalid session")

type SessionsService struct {
	cfg *config.Config
	rp  repository.SessionsRepository
}

func NewSessionsService(
	cfg *config.Config,
	rp repository.SessionsRepository,
) *SessionsService {
	return &SessionsService{
		cfg: cfg,
		rp:  rp,
	}
}

func (s *SessionsService) Create(ctx context.Context, userId uuid.UUID) (*domain.Sessions, error) {
	if userId == uuid.Nil {
		return nil, fmt.Errorf("%w: user_id is empty", ErrInvalidSession)
	}

	expiresAt := time.Now().UTC().Add(s.cfg.SessionTTL)
	return s.rp.Create(ctx, domain.CreateSessionInput{
		UserId:    userId,
		ExpiresAt: expiresAt,
	})
}

func (s *SessionsService) GetValidByID(ctx context.Context, id uuid.UUID) (*domain.Sessions, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: id is empty", ErrInvalidSession)
	}

	return s.rp.GetValidByID(ctx, id)
}

func (s *SessionsService) Revoke(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: id is empty", ErrInvalidSession)
	}

	return s.rp.Revoke(ctx, id)
}

package handler

import (
	"time"

	"attendance/internal/domain"
)

type userResponse struct {
	Id        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
}

func newUserResponse(user *domain.Users) userResponse {
	return userResponse{
		Id:        user.Id.String(),
		Email:     user.Email,
		FullName:  user.FullName,
		CreatedAt: user.CreatedAt,
	}
}

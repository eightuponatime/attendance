package handler

import (
	"time"

	"attendance/internal/domain"
)

type userResponse struct {
	Id         string    `json:"id"`
	Email      string    `json:"email"`
	LastName   string    `json:"last_name"`
	FirstName  string    `json:"first_name"`
	MiddleName *string   `json:"middle_name"`
	FullName   string    `json:"full_name"`
	CreatedAt  time.Time `json:"created_at"`
}

func newUserResponse(user *domain.Users) userResponse {
	return userResponse{
		Id:         user.Id.String(),
		Email:      user.Email,
		LastName:   user.LastName,
		FirstName:  user.FirstName,
		MiddleName: user.MiddleName,
		FullName:   user.FullName,
		CreatedAt:  user.CreatedAt,
	}
}

package dto

import (
	userdto "github.com/sweetlife999/chain-of-trades-avito/internal/user/dto"
	usermodel "github.com/sweetlife999/chain-of-trades-avito/internal/user/model"
)

type LoginRequest struct {
	Nickname string `json:"nickname"`
	Password string `json:"password"`
}

// AuthenticatedUserResponse расширяет безопасный публичный профиль только для ответов
// login/me. Публичный GET /users/{id} не раскрывает, кто является администратором.
type AuthenticatedUserResponse struct {
	userdto.UserResponse
	IsAdmin bool `json:"is_admin"`
}

func UserFromModel(user usermodel.User) AuthenticatedUserResponse {
	return AuthenticatedUserResponse{
		UserResponse: userdto.FromModel(user),
		IsAdmin:      user.IsAdmin,
	}
}

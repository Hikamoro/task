package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"task/internal/auth"
	"task/internal/model"
)

var (
	ErrEmailTaken   = errors.New("email already registered")
	ErrInvalidCreds = errors.New("invalid email or password")
)

func validateEmail(email string) error {
	if len(email) > 255 {
		return model.ErrInvalidInput
	}
	_, err := mail.ParseAddress(email)
	if err != nil {
		return model.ErrInvalidInput
	}
	return nil
}

func (a *App) Register(ctx context.Context, email, password, name string) (model.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)

	if err := validateEmail(email); err != nil {
		return model.User{}, model.ErrInvalidInput
	}
	if len(password) < 6 || len(password) > 72 {
		return model.User{}, model.ErrInvalidInput
	}
	if name == "" || len(name) > 100 {
		return model.User{}, model.ErrInvalidInput
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return model.User{}, err
	}
	user, err := a.repo.CreateUser(ctx, email, hash, name)
	if err != nil {
		if errors.Is(err, model.ErrDuplicate) {
			return model.User{}, ErrEmailTaken
		}
		return model.User{}, err
	}
	return user, nil
}

func (a *App) Login(ctx context.Context, email, password string) (string, model.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := a.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return "", model.User{}, ErrInvalidCreds
		}
		return "", model.User{}, err
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return "", model.User{}, ErrInvalidCreds
	}
	token, err := a.auth.Generate(user.ID)
	if err != nil {
		return "", model.User{}, err
	}
	return token, user, nil
}

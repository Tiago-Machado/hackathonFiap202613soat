package usecase

import (
	"context"
	"errors"
	"strings"

	"video-processor/internal/domain"

	"github.com/google/uuid"
)

const minPasswordLength = 8

var (
	ErrEmailInUse         = errors.New("e-mail já cadastrado")
	ErrInvalidCredentials = errors.New("credenciais inválidas")
	ErrWeakPassword       = errors.New("a senha deve ter ao menos 8 caracteres")
	ErrInvalidEmail       = errors.New("e-mail inválido")
)

type UserAccountRepository interface {
	Create(ctx context.Context, u *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
}

type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) bool
}

type TokenIssuer interface {
	Issue(userID string) (string, error)
}

type Credentials struct {
	Email    string
	Password string
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

type Register struct {
	users  UserAccountRepository
	hasher PasswordHasher
	now    Clock
}

func NewRegister(users UserAccountRepository, hasher PasswordHasher, now Clock) *Register {
	return &Register{users: users, hasher: hasher, now: now}
}

func (uc *Register) Execute(ctx context.Context, in Credentials) (*domain.User, error) {
	email := normalizeEmail(in.Email)
	if !strings.Contains(email, "@") {
		return nil, ErrInvalidEmail
	}
	if len(in.Password) < minPasswordLength {
		return nil, ErrWeakPassword
	}

	existing, err := uc.users.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailInUse
	}

	hash, err := uc.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: hash,
		IsActive:     true,
		CreatedAt:    uc.now(),
	}
	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

type Login struct {
	users  UserAccountRepository
	hasher PasswordHasher
	tokens TokenIssuer
}

func NewLogin(users UserAccountRepository, hasher PasswordHasher, tokens TokenIssuer) *Login {
	return &Login{users: users, hasher: hasher, tokens: tokens}
}

func (uc *Login) Execute(ctx context.Context, in Credentials) (string, error) {
	user, err := uc.users.FindByEmail(ctx, normalizeEmail(in.Email))
	if err != nil || user == nil {
		return "", ErrInvalidCredentials
	}
	if !user.IsActive || !uc.hasher.Compare(user.PasswordHash, in.Password) {
		return "", ErrInvalidCredentials
	}
	return uc.tokens.Issue(user.ID)
}

package usecase

import (
	"context"
	"errors"
	"testing"

	"video-processor/internal/domain"
)

func TestRegisterHappyPathNormalizesEmail(t *testing.T) {
	users := newFakeUserRepo()
	uc := NewRegister(users, fakeHasher{}, fixedClock)

	user, err := uc.Execute(context.Background(), Credentials{Email: "  Neo@Exemplo.COM ", Password: "senhaforte1"})
	if err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if user.Email != "neo@exemplo.com" {
		t.Errorf("email = %q, esperado normalizado", user.Email)
	}
	if user.PasswordHash != "hashed:senhaforte1" {
		t.Errorf("senha não foi hasheada: %q", user.PasswordHash)
	}
	if !user.IsActive {
		t.Error("usuário deveria nascer ativo")
	}
}

func TestRegisterRejectsWeakPassword(t *testing.T) {
	uc := NewRegister(newFakeUserRepo(), fakeHasher{}, fixedClock)
	_, err := uc.Execute(context.Background(), Credentials{Email: "a@b.com", Password: "1234"})
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("esperava ErrWeakPassword, veio: %v", err)
	}
}

func TestRegisterRejectsInvalidEmail(t *testing.T) {
	uc := NewRegister(newFakeUserRepo(), fakeHasher{}, fixedClock)
	_, err := uc.Execute(context.Background(), Credentials{Email: "sem-arroba", Password: "senhaforte1"})
	if !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("esperava ErrInvalidEmail, veio: %v", err)
	}
}

func TestRegisterRejectsDuplicate(t *testing.T) {
	users := newFakeUserRepo()
	users.byEmail["a@b.com"] = &domain.User{ID: "existente", Email: "a@b.com"}
	uc := NewRegister(users, fakeHasher{}, fixedClock)

	_, err := uc.Execute(context.Background(), Credentials{Email: "a@b.com", Password: "senhaforte1"})
	if !errors.Is(err, ErrEmailInUse) {
		t.Fatalf("esperava ErrEmailInUse, veio: %v", err)
	}
}

func TestLoginHappyPath(t *testing.T) {
	users := newFakeUserRepo()
	users.byEmail["a@b.com"] = &domain.User{ID: "u1", Email: "a@b.com", PasswordHash: "hashed:senhaforte1", IsActive: true}
	uc := NewLogin(users, fakeHasher{}, fakeTokens{})

	token, err := uc.Execute(context.Background(), Credentials{Email: "a@b.com", Password: "senhaforte1"})
	if err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if token != "token:u1" {
		t.Errorf("token = %q", token)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	users := newFakeUserRepo()
	users.byEmail["a@b.com"] = &domain.User{ID: "u1", Email: "a@b.com", PasswordHash: "hashed:certa", IsActive: true}
	uc := NewLogin(users, fakeHasher{}, fakeTokens{})

	_, err := uc.Execute(context.Background(), Credentials{Email: "a@b.com", Password: "errada"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("esperava ErrInvalidCredentials, veio: %v", err)
	}
}

func TestLoginUnknownEmail(t *testing.T) {
	uc := NewLogin(newFakeUserRepo(), fakeHasher{}, fakeTokens{})
	_, err := uc.Execute(context.Background(), Credentials{Email: "ninguem@b.com", Password: "senhaforte1"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("esperava ErrInvalidCredentials, veio: %v", err)
	}
}

func TestLoginInactiveUser(t *testing.T) {
	users := newFakeUserRepo()
	users.byEmail["a@b.com"] = &domain.User{ID: "u1", Email: "a@b.com", PasswordHash: "hashed:senhaforte1", IsActive: false}
	uc := NewLogin(users, fakeHasher{}, fakeTokens{})

	_, err := uc.Execute(context.Background(), Credentials{Email: "a@b.com", Password: "senhaforte1"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("esperava ErrInvalidCredentials para usuário inativo, veio: %v", err)
	}
}

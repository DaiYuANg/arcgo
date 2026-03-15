package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/DaiYuANg/archgo/examples/rbac_backend/internal/entity"
	repoauth "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/repository/auth"
	"golang.org/x/crypto/bcrypt"
)

// Service handles authentication business logic, including credential
// verification (bcrypt) and JWT issuance.
type Service struct {
	repo repoauth.Repository
	jwt  *JWTService
}

func NewService(repo repoauth.Repository, jwt *JWTService) *Service {
	return &Service{repo: repo, jwt: jwt}
}

// Login verifies credentials and returns a Principal + signed JWT on success.
// Bcrypt comparison happens here so the repository never performs plaintext
// password checks in SQL.
func (s *Service) Login(ctx context.Context, username string, password string) (entity.Principal, string, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return entity.Principal{}, "", errors.New("invalid username or password")
	}

	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Principal{}, "", errors.New("invalid username or password")
		}
		return entity.Principal{}, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		// Return a generic message to avoid user-enumeration attacks.
		return entity.Principal{}, "", errors.New("invalid username or password")
	}

	roles, err := s.repo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return entity.Principal{}, "", err
	}

	principal := entity.Principal{
		UserID:   user.ID,
		Username: user.Username,
		Roles:    roles,
	}

	token, err := s.jwt.IssueToken(principal)
	if err != nil {
		return entity.Principal{}, "", err
	}

	return principal, token, nil
}

// AuthorizationService checks whether a user is allowed to perform an action
// on a resource using the RBAC permission tables.
type AuthorizationService struct {
	repo repoauth.AuthorizationRepository
}

func NewAuthorizationService(repo repoauth.AuthorizationRepository) *AuthorizationService {
	return &AuthorizationService{repo: repo}
}

func (s *AuthorizationService) Can(ctx context.Context, userID int64, action string, resource string) (bool, error) {
	return s.repo.Can(ctx, userID, action, resource)
}

package auth

import (
	"context"

	"github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/entity"
	repoauth "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/repository/auth"
)

type Service struct {
	repo repoauth.Repository
	jwt  *JWTService
}

func NewService(repo repoauth.Repository, jwt *JWTService) *Service {
	return &Service{repo: repo, jwt: jwt}
}

func (s *Service) Login(ctx context.Context, username string, password string) (entity.Principal, string, error) {
	principal, err := s.repo.Login(ctx, username, password)
	if err != nil {
		return entity.Principal{}, "", err
	}
	token, err := s.jwt.IssueToken(principal)
	if err != nil {
		return entity.Principal{}, "", err
	}
	return principal, token, nil
}

type AuthorizationService struct {
	repo repoauth.AuthorizationRepository
}

func NewAuthorizationService(repo repoauth.AuthorizationRepository) *AuthorizationService {
	return &AuthorizationService{repo: repo}
}

func (s *AuthorizationService) Can(ctx context.Context, userID int64, action string, resource string) (bool, error) {
	return s.repo.Can(ctx, userID, action, resource)
}

package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/DaiYuANg/archgo/examples/rbac_backend/internal/entity"
	modeluser "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/model/user"
	repouser "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/repository/user"
	"golang.org/x/crypto/bcrypt"
)

type CreateCommand struct {
	Username  string
	Password  string
	RoleCodes []string
}

type UpdateCommand struct {
	Username string
	// Password is optional on update. When empty the stored hash is left unchanged.
	Password  string
	RoleCodes []string
}

// Service handles user management business logic.
type Service struct {
	repo repouser.Repository
}

func NewService(repo repouser.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]modeluser.Item, error) {
	rows, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]modeluser.Item, 0, len(rows))
	for _, row := range rows {
		item, buildErr := s.buildUserItem(ctx, row)
		if buildErr != nil {
			return nil, buildErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) Get(ctx context.Context, id int64) (modeluser.Item, error) {
	row, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return modeluser.Item{}, err
	}
	return s.buildUserItem(ctx, row)
}

func (s *Service) Create(ctx context.Context, cmd CreateCommand) (modeluser.Item, error) {
	if strings.TrimSpace(cmd.Username) == "" {
		return modeluser.Item{}, fmt.Errorf("username is required")
	}
	if cmd.Password == "" {
		return modeluser.Item{}, fmt.Errorf("password is required")
	}

	hashed, err := hashPassword(cmd.Password)
	if err != nil {
		return modeluser.Item{}, err
	}

	row, err := s.repo.CreateUser(ctx, strings.TrimSpace(cmd.Username), hashed)
	if err != nil {
		return modeluser.Item{}, err
	}
	if err := s.repo.ReplaceUserRoles(ctx, row.ID, cmd.RoleCodes); err != nil {
		return modeluser.Item{}, err
	}
	return s.Get(ctx, row.ID)
}

func (s *Service) Update(ctx context.Context, id int64, cmd UpdateCommand) (modeluser.Item, error) {
	// Hash the new password only when the caller actually wants to change it.
	// An empty password means "leave the stored hash as-is".
	var hashedPassword string
	if cmd.Password != "" {
		h, err := hashPassword(cmd.Password)
		if err != nil {
			return modeluser.Item{}, err
		}
		hashedPassword = h
	}

	if _, err := s.repo.UpdateUser(ctx, id, strings.TrimSpace(cmd.Username), hashedPassword); err != nil {
		return modeluser.Item{}, err
	}
	if err := s.repo.ReplaceUserRoles(ctx, id, cmd.RoleCodes); err != nil {
		return modeluser.Item{}, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) (bool, error) {
	return s.repo.DeleteUser(ctx, id)
}

// buildUserItem enriches a raw UserModel with its role codes.
func (s *Service) buildUserItem(ctx context.Context, row entity.UserModel) (modeluser.Item, error) {
	roles, err := s.repo.UserRoles(ctx, row.ID)
	if err != nil {
		return modeluser.Item{}, err
	}
	return modeluser.Item{
		ID:        row.ID,
		Username:  row.Username,
		Roles:     roles,
		CreatedAt: row.CreatedAt,
	}, nil
}

// hashPassword generates a bcrypt hash from a plaintext password.
func hashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}

package shared

import (
	"strings"
	"sync"
	"time"

	collectionmapping "github.com/DaiYuANg/archgo/collectionx/mapping"
)

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateUserBody struct {
	Name  string `json:"name" validate:"required,min=2,max=64"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0,lte=130"`
}

type UpdateUserBody struct {
	Name  *string `json:"name,omitempty" validate:"omitempty,min=2,max=64"`
	Email *string `json:"email,omitempty" validate:"omitempty,email"`
	Age   *int    `json:"age,omitempty" validate:"omitempty,gte=0,lte=130"`
}

type UserService interface {
	List(search string, limit, offset int) ([]User, int)
	Get(id int) (User, bool)
	Create(in CreateUserBody) User
	Update(id int, in UpdateUserBody) (User, bool)
	Delete(id int) bool
}

// MockUserService is an in-memory mock service used by examples.
type MockUserService struct {
	mu     sync.RWMutex
	nextID int
	users  *collectionmapping.Map[int, User]
}

func NewMockUserService() *MockUserService {
	now := time.Now().UTC()
	return &MockUserService{
		nextID: 3,
		users: collectionmapping.NewMapFrom(map[int]User{
			1: {ID: 1, Name: "Alice", Email: "alice@example.com", Age: 26, CreatedAt: now, UpdatedAt: now},
			2: {ID: 2, Name: "Bob", Email: "bob@example.com", Age: 30, CreatedAt: now, UpdatedAt: now},
		}),
	}
}

func (s *MockUserService) List(search string, limit, offset int) ([]User, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lowered := strings.ToLower(strings.TrimSpace(search))
	items := make([]User, 0, s.users.Len())
	s.users.Range(func(_ int, user User) bool {
		if lowered != "" && !strings.Contains(strings.ToLower(user.Name+user.Email), lowered) {
			return true
		}
		items = append(items, user)
		return true
	})

	total := len(items)
	if offset >= total {
		return []User{}, total
	}

	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], total
}

func (s *MockUserService) Get(id int) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users.Get(id)
}

func (s *MockUserService) Create(in CreateUserBody) User {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	user := User{
		ID:        s.nextID,
		Name:      in.Name,
		Email:     in.Email,
		Age:       in.Age,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.users.Set(user.ID, user)
	s.nextID++
	return user
}

func (s *MockUserService) Update(id int, in UpdateUserBody) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users.Get(id)
	if !ok {
		return User{}, false
	}

	if in.Name != nil {
		user.Name = *in.Name
	}
	if in.Email != nil {
		user.Email = *in.Email
	}
	if in.Age != nil {
		user.Age = *in.Age
	}
	user.UpdatedAt = time.Now().UTC()

	s.users.Set(id, user)
	return user, true
}

func (s *MockUserService) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.users.Delete(id)
}

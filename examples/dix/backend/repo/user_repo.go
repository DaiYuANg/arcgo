package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/examples/dix/backend/domain"
	backendschema "github.com/DaiYuANg/arcgo/examples/dix/backend/schema"
)

// UserRepository persists backend users.
type UserRepository interface {
	List(ctx context.Context, search string, limit, offset int) ([]domain.User, int, error)
	GetByID(ctx context.Context, id int64) (domain.User, bool, error)
	Create(ctx context.Context, in domain.CreateUserInput) (domain.User, error)
	Update(ctx context.Context, id int64, in domain.UpdateUserInput) (domain.User, bool, error)
	Delete(ctx context.Context, id int64) (bool, error)
}

type userRepo struct {
	db     *dbx.DB
	schema backendschema.UserSchema
}

// NewUserRepository creates a user repository backed by dbx.
func NewUserRepository(db *dbx.DB, userSchema backendschema.UserSchema) UserRepository {
	return &userRepo{db: db, schema: userSchema}
}

func (r *userRepo) List(ctx context.Context, search string, limit, offset int) ([]domain.User, int, error) {
	s := r.schema
	mapper := dbx.MustMapper[backendschema.UserRow](s)

	q := dbx.Select(s.AllColumns()...).From(s)
	if search != "" {
		pattern := "%" + strings.TrimSpace(search) + "%"
		q = q.Where(dbx.Or(
			dbx.Like(s.Name, pattern),
			dbx.Like(s.Email, pattern),
		))
	}
	q = q.OrderBy(s.ID.Asc())

	all, err := dbx.QueryAll[backendschema.UserRow](ctx, r.db, q, mapper)
	if err != nil {
		return nil, 0, err
	}
	total := len(all)

	if offset >= total {
		return []domain.User{}, total, nil
	}
	end := min(offset+limit, total)
	page := all[offset:end]

	users := make([]domain.User, len(page))
	for i, row := range page {
		users[i] = rowToDomain(row)
	}
	return users, total, nil
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (domain.User, bool, error) {
	s := r.schema
	mapper := dbx.MustMapper[backendschema.UserRow](s)

	rows, err := dbx.QueryAll[backendschema.UserRow](ctx, r.db,
		dbx.Select(s.AllColumns()...).From(s).Where(s.ID.Eq(id)),
		mapper,
	)
	if err != nil {
		return domain.User{}, false, err
	}
	if len(rows) == 0 {
		return domain.User{}, false, nil
	}
	return rowToDomain(rows[0]), true, nil
}

func (r *userRepo) Create(ctx context.Context, in domain.CreateUserInput) (domain.User, error) {
	s := r.schema
	mapper := dbx.MustMapper[backendschema.UserRow](s)
	now := time.Now().UTC()

	rows, err := dbx.QueryAll[backendschema.UserRow](ctx, r.db,
		dbx.InsertInto(s).
			Columns(s.Name, s.Email, s.Age, s.CreatedAt, s.UpdatedAt).
			Values(
				s.Name.Set(in.Name),
				s.Email.Set(in.Email),
				s.Age.Set(in.Age),
				s.CreatedAt.Set(now),
				s.UpdatedAt.Set(now),
			).
			Returning(s.AllColumns()...),
		mapper,
	)
	if err != nil {
		return domain.User{}, err
	}
	return rowToDomain(rows[0]), nil
}

func (r *userRepo) Update(ctx context.Context, id int64, in domain.UpdateUserInput) (domain.User, bool, error) {
	s := r.schema
	mapper := dbx.MustMapper[backendschema.UserRow](s)

	assignments := []dbx.Assignment{s.UpdatedAt.Set(time.Now().UTC())}
	if in.Name != nil {
		assignments = append(assignments, s.Name.Set(*in.Name))
	}
	if in.Email != nil {
		assignments = append(assignments, s.Email.Set(*in.Email))
	}
	if in.Age != nil {
		assignments = append(assignments, s.Age.Set(*in.Age))
	}

	rows, err := dbx.QueryAll[backendschema.UserRow](ctx, r.db,
		dbx.Update(s).Set(assignments...).Where(s.ID.Eq(id)).Returning(s.AllColumns()...),
		mapper,
	)
	if err != nil {
		return domain.User{}, false, err
	}
	if len(rows) == 0 {
		return domain.User{}, false, nil
	}
	return rowToDomain(rows[0]), true, nil
}

func (r *userRepo) Delete(ctx context.Context, id int64) (bool, error) {
	s := r.schema
	res, err := dbx.Exec(ctx, r.db, dbx.DeleteFrom(s).Where(s.ID.Eq(id)))
	if err != nil {
		return false, fmt.Errorf("delete user: %w", err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete user rows affected: %w", err)
	}
	return ra > 0, nil
}

func rowToDomain(row backendschema.UserRow) domain.User {
	return domain.User{
		ID:        row.ID,
		Name:      row.Name,
		Email:     row.Email,
		Age:       row.Age,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

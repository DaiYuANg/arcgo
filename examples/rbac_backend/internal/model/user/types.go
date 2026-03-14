package user

import (
	"time"

	modelresult "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/model/resultx"
)

type Item struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
}

type ListData struct {
	Items []Item `json:"items"`
	Total int    `json:"total"`
}

type ListOutput = modelresult.Result[ListData]

type GetInput struct {
	ID int64 `path:"id" validate:"required,min=1"`
}

type GetOutput = modelresult.Result[Item]

type CreateInput struct {
	Body struct {
		Username  string   `json:"username" validate:"required,min=3,max=64"`
		Password  string   `json:"password" validate:"required,min=3,max=128"`
		RoleCodes []string `json:"role_codes"`
	} `json:"body"`
}

type CreateOutput = modelresult.Result[Item]

type UpdateInput struct {
	ID   int64 `path:"id" validate:"required,min=1"`
	Body struct {
		Username  string   `json:"username" validate:"required,min=3,max=64"`
		Password  string   `json:"password" validate:"required,min=3,max=128"`
		RoleCodes []string `json:"role_codes"`
	} `json:"body"`
}

type UpdateOutput = modelresult.Result[Item]

type DeleteInput struct {
	ID int64 `path:"id" validate:"required,min=1"`
}

type DeleteData struct {
	Deleted bool `json:"deleted"`
}

type DeleteOutput = modelresult.Result[DeleteData]

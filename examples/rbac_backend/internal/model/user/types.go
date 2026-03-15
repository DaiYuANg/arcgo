package user

import (
	"time"

	modelresult "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/model/resultx"
)

type UserItem struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
}

type Item = UserItem

type UserListData struct {
	Items []UserItem `json:"items"`
	Total int        `json:"total"`
}

type ListData = UserListData

type ListOutput = modelresult.Result[UserListData]

type UserGetInput struct {
	ID int64 `path:"id" validate:"required,min=1"`
}

type GetInput = UserGetInput

type GetOutput = modelresult.Result[UserItem]

type UserCreateInput struct {
	Body struct {
		Username  string   `json:"username" validate:"required,min=3,max=64"`
		Password  string   `json:"password" validate:"required,min=3,max=128"`
		RoleCodes []string `json:"role_codes"`
	} `json:"body"`
}

type CreateInput = UserCreateInput

type CreateOutput = modelresult.Result[UserItem]

type UserUpdateInput struct {
	ID   int64 `path:"id" validate:"required,min=1"`
	Body struct {
		Username  string   `json:"username" validate:"required,min=3,max=64"`
		Password  string   `json:"password" validate:"omitempty,min=3,max=128"`
		RoleCodes []string `json:"role_codes"`
	} `json:"body"`
}

type UpdateInput = UserUpdateInput

type UpdateOutput = modelresult.Result[UserItem]

type UserDeleteInput struct {
	ID int64 `path:"id" validate:"required,min=1"`
}

type DeleteInput = UserDeleteInput

type UserDeleteData struct {
	Deleted bool `json:"deleted"`
}

type DeleteData = UserDeleteData

type DeleteOutput = modelresult.Result[UserDeleteData]

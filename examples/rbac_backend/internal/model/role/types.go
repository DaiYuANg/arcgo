package role

import modelresult "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/model/resultx"

type RoleItem struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type Item = RoleItem

type RoleListData struct {
	Items []RoleItem `json:"items"`
	Total int        `json:"total"`
}

type ListData = RoleListData

type ListOutput = modelresult.Result[RoleListData]

type RoleGetInput struct {
	ID int64 `path:"id" validate:"required,min=1"`
}

type GetInput = RoleGetInput

type GetOutput = modelresult.Result[RoleItem]

type RoleCreateInput struct {
	Body struct {
		Code string `json:"code" validate:"required,min=2,max=64"`
		Name string `json:"name" validate:"required,min=1,max=120"`
	} `json:"body"`
}

type CreateInput = RoleCreateInput

type CreateOutput = modelresult.Result[RoleItem]

type RoleUpdateInput struct {
	ID   int64 `path:"id" validate:"required,min=1"`
	Body struct {
		Code string `json:"code" validate:"required,min=2,max=64"`
		Name string `json:"name" validate:"required,min=1,max=120"`
	} `json:"body"`
}

type UpdateInput = RoleUpdateInput

type UpdateOutput = modelresult.Result[RoleItem]

type RoleDeleteInput struct {
	ID int64 `path:"id" validate:"required,min=1"`
}

type DeleteInput = RoleDeleteInput

type RoleDeleteData struct {
	Deleted bool `json:"deleted"`
}

type DeleteData = RoleDeleteData

type DeleteOutput = modelresult.Result[RoleDeleteData]

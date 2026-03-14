package role

import modelresult "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/model/resultx"

type Item struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
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
		Code string `json:"code" validate:"required,min=2,max=64"`
		Name string `json:"name" validate:"required,min=1,max=120"`
	} `json:"body"`
}

type CreateOutput = modelresult.Result[Item]

type UpdateInput struct {
	ID   int64 `path:"id" validate:"required,min=1"`
	Body struct {
		Code string `json:"code" validate:"required,min=2,max=64"`
		Name string `json:"name" validate:"required,min=1,max=120"`
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

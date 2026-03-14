package book

import modelresult "github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/model/resultx"

type Item struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	CreatedBy int64  `json:"created_by"`
}

type ListData struct {
	Items []Item `json:"items"`
	Total int    `json:"total"`
}

type ListOutput = modelresult.Result[ListData]

type CreateInput struct {
	Body struct {
		Title  string `json:"title" validate:"required,min=1,max=200"`
		Author string `json:"author" validate:"required,min=1,max=120"`
	} `json:"body"`
}

type CreateOutput = modelresult.Result[Item]

type DeleteInput struct {
	ID int64 `path:"id" validate:"required,min=1"`
}

type DeleteData struct {
	Deleted bool `json:"deleted"`
}

type DeleteOutput = modelresult.Result[DeleteData]

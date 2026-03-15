package book

import modelresult "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/model/resultx"

type BookItem struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	CreatedBy int64  `json:"created_by"`
}

type Item = BookItem

type BookListData struct {
	Items []BookItem `json:"items"`
	Total int        `json:"total"`
}

type ListData = BookListData

type ListOutput = modelresult.Result[BookListData]

type BookCreateInput struct {
	Body struct {
		Title  string `json:"title" validate:"required,min=1,max=200"`
		Author string `json:"author" validate:"required,min=1,max=120"`
	} `json:"body"`
}

type CreateInput = BookCreateInput

type CreateOutput = modelresult.Result[BookItem]

type BookDeleteInput struct {
	ID int64 `path:"id" validate:"required,min=1"`
}

type DeleteInput = BookDeleteInput

type BookDeleteData struct {
	Deleted bool `json:"deleted"`
}

type DeleteData = BookDeleteData

type DeleteOutput = modelresult.Result[BookDeleteData]

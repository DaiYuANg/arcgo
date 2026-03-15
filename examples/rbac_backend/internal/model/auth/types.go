package auth

import modelresult "github.com/DaiYuANg/archgo/examples/rbac_backend/internal/model/resultx"

type LoginInput struct {
	Body struct {
		Username string `json:"username" validate:"required,min=3,max=64"`
		Password string `json:"password" validate:"required,min=3,max=128"`
	} `json:"body"`
}

type LoginData struct {
	Token    string   `json:"token"`
	UserID   int64    `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

type LoginOutput = modelresult.Result[LoginData]

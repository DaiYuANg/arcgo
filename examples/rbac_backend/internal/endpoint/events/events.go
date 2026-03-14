package events

type LoginSucceededEvent struct {
	UserID   int64
	Username string
	Roles    []string
}

func (e LoginSucceededEvent) Name() string {
	return "auth.login.succeeded"
}

type BookCreatedEvent struct {
	BookID  int64
	Title   string
	Author  string
	ActorID int64
	Actor   string
}

func (e BookCreatedEvent) Name() string {
	return "rbac.book.created"
}

type BookDeletedEvent struct {
	BookID  int64
	ActorID int64
	Actor   string
}

func (e BookDeletedEvent) Name() string {
	return "rbac.book.deleted"
}

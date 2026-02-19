package domain

type Command interface {
	Name() string
	Execute(chatID int64) (string, error)
}

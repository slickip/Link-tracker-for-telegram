package repositories

type ChatRepository interface {
	Add(chatID int64) error

	Remove(chatID int64) error

	Exists(chatID int64) bool
}

type InMemoryChatRepository struct {
	chats map[int64]struct{}
}

func NewInMemoryChatRepository() *InMemoryChatRepository {
	return &InMemoryChatRepository{
		chats: make(map[int64]struct{}),
	}
}

func (r *InMemoryChatRepository) Add(chatID int64) error {
	r.chats[chatID] = struct{}{}
	return nil
}

func (r *InMemoryChatRepository) Remove(chatID int64) error {
	delete(r.chats, chatID)
	return nil
}

func (r *InMemoryChatRepository) Exists(chatID int64) bool {
	_, ok := r.chats[chatID]
	return ok
}


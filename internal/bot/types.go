package bot

import (
	"context"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CommandHandler обработчик команды
type CommandHandler func(update tgbotapi.Update) error

// UserState состояние пользователя
type UserState struct {
	CurrentCommand string
	Step           int
	Data           map[string]interface{}
	LastActivity   time.Time
	Context        context.Context
	CancelFunc     context.CancelFunc
	MessageID      int // ID сообщения для редактирования
}

// BotStats статистика бота
type BotStats struct {
	StartTime        time.Time
	MessagesReceived int64
	MessagesSent     int64
	CommandsExecuted int64
	Errors           int64
	ActiveUsers      map[int64]time.Time
	mu               sync.RWMutex
}

// NewBotStats создает новую статистику
func NewBotStats() *BotStats {
	return &BotStats{
		StartTime:   time.Now(),
		ActiveUsers: make(map[int64]time.Time),
	}
}

// UpdateStats обновляет статистику
func (s *BotStats) UpdateStats(updateType string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch updateType {
	case "message_received":
		s.MessagesReceived++
	case "message_sent":
		s.MessagesSent++
	case "command_executed":
		s.CommandsExecuted++
	case "error":
		s.Errors++
	}
}

// AddActiveUser добавляет активного пользователя
func (s *BotStats) AddActiveUser(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActiveUsers[userID] = time.Now()
}

// GetActiveUsersCount возвращает количество активных пользователей
func (s *BotStats) GetActiveUsersCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.ActiveUsers)
}

// IsUserActive проверяет активен ли пользователь
func (s *BotStats) IsUserActive(userID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.ActiveUsers[userID]
	return exists
}

// GetUptime возвращает время работы бота
func (s *BotStats) GetUptime() time.Duration {
	return time.Since(s.StartTime)
}

// CleanupInactiveUsers очищает неактивных пользователей
func (s *BotStats) CleanupInactiveUsers(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for userID, lastActivity := range s.ActiveUsers {
		if now.Sub(lastActivity) > timeout {
			delete(s.ActiveUsers, userID)
		}
	}
}

// CommandInfo информация о команде
type CommandInfo struct {
	Name        string
	Description string
	Handler     CommandHandler
	IsAdmin     bool
}

// MessageQueue очередь сообщений
type MessageQueue struct {
	Messages chan tgbotapi.Chattable
	Done     chan struct{}
}

// NewMessageQueue создает новую очередь сообщений
func NewMessageQueue(bufferSize int) *MessageQueue {
	return &MessageQueue{
		Messages: make(chan tgbotapi.Chattable, bufferSize),
		Done:     make(chan struct{}),
	}
}

// Start запускает обработчик очереди
func (q *MessageQueue) Start(botAPI *tgbotapi.BotAPI) {
	go func() {
		for {
			select {
			case msg := <-q.Messages:
				if _, err := botAPI.Send(msg); err != nil {
					// Логируем ошибку, но не паникуем
					log.Printf("❌ Ошибка отправки сообщения: %v", err)
				}
			case <-q.Done:
				return
			}
		}
	}()
}

// Stop останавливает обработчик очереди
func (q *MessageQueue) Stop() {
	close(q.Done)
}

// BotConfig конфигурация бота (дополнение к основной конфиг)
type BotConfig struct {
	EnableRateLimit      bool
	RateLimitInterval    time.Duration
	MaxMessageQueueSize  int
	EnableUserStates     bool
	StateCleanupInterval time.Duration
}

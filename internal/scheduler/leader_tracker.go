package scheduler

import (
	"brochievements/internal/achievements"
	"brochievements/internal/storage"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

type LeaderTracker struct {
	session   *discordgo.Session
	store     *storage.Storage
	channelID string
	stopChan  chan struct{}
}

func NewLeaderTracker(
	session *discordgo.Session,
	store *storage.Storage,
	channelID string,
) *LeaderTracker {
	return &LeaderTracker{
		session:   session,
		store:     store,
		channelID: channelID,
		stopChan:  make(chan struct{}),
	}
}

// Start запускает отслеживание лидеров
func (lt *LeaderTracker) Start() {
	go lt.run()
	log.Println("[leader_tracker] Leader tracking started")
}

// Stop останавливает отслеживание
func (lt *LeaderTracker) Stop() {
	close(lt.stopChan)
	log.Println("[leader_tracker] Leader tracking stopped")
}

func (lt *LeaderTracker) run() {
	for {
		next := lt.nextCheckTime()
		duration := time.Until(next)

		log.Printf("[leader_tracker] Next leader check: %s (in %s)",
			next.Format("2006-01-02 15:04:05"),
			duration.Round(time.Minute),
		)

		select {
		case <-time.After(duration):
			lt.checkLeaders()
		case <-lt.stopChan:
			return
		}
	}
}

// nextCheckTime вычисляет следующее время проверки (каждый час с четверга по воскресенье с 18:00 до 24:00)
func (lt *LeaderTracker) nextCheckTime() time.Time {
	now := time.Now()
	weekday := now.Weekday()
	currentHour := now.Hour()

	// Проверяем, является ли текущий день четвергом или позже до воскресенья включительно
	isCheckDay := weekday >= time.Thursday || weekday == time.Sunday

	// Если сейчас подходящий день
	if isCheckDay {
		// Если сейчас между 18:00 и 23:59
		if currentHour >= 18 && currentHour < 24 {
			// Следующая проверка - через час (в начале часа)
			nextHour := now.Add(time.Hour).Truncate(time.Hour)

			// Если следующий час всё ещё в рамках 18:00-24:00, используем его
			if nextHour.Hour() < 24 {
				return nextHour
			}
		}

		// Если сейчас после 00:00 и до 18:00, ждём до 18:00 текущего дня
		if currentHour < 18 {
			return time.Date(
				now.Year(), now.Month(), now.Day(),
				18, 0, 0, 0,
				now.Location(),
			)
		}

		// Если сейчас после 24:00 (00:00-18:00 следующего дня) и это воскресенье
		if weekday == time.Sunday && currentHour < 18 {
			return time.Date(
				now.Year(), now.Month(), now.Day(),
				18, 0, 0, 0,
				now.Location(),
			)
		}

		// Переходим к следующему дню
		tomorrow := now.AddDate(0, 0, 1)
		tomorrowWeekday := tomorrow.Weekday()

		// Если завтра тоже подходящий день
		if tomorrowWeekday >= time.Thursday || tomorrowWeekday == time.Sunday {
			return time.Date(
				tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
				18, 0, 0, 0,
				now.Location(),
			)
		}
	}

	// Если сейчас до четверга, ждём до четверга 18:00
	daysUntilThursday := (int(time.Thursday) - int(weekday) + 7) % 7
	if daysUntilThursday == 0 {
		daysUntilThursday = 7
	}

	nextThursday := now.AddDate(0, 0, daysUntilThursday)
	return time.Date(
		nextThursday.Year(), nextThursday.Month(), nextThursday.Day(),
		18, 0, 0, 0,
		now.Location(),
	)
}

func (lt *LeaderTracker) checkLeaders() {
	log.Println("[leader_tracker] Checking leaders...")

	// Получаем начало текущей недели (понедельник 00:00)
	weekStart := lt.getCurrentWeekStart()

	// Очищаем старых лидеров (если началась новая неделя)
	if err := lt.store.ClearOldLeaders(weekStart); err != nil {
		log.Println("[leader_tracker] Error clearing old leaders:", err)
	}

	// Загружаем текущую статистику
	stats, err := achievements.LoadWeeklyStats(lt.store)
	if err != nil {
		log.Println("[leader_tracker] Error loading weekly stats:", err)
		return
	}

	// Получаем все достижения
	achs := achievements.RunWeeklyAll(stats)

	var changedLeaders []LeaderChange

	// Проверяем каждое достижение
	for _, ach := range achs {
		changed := lt.checkAchievementLeader(ach, weekStart)
		if changed != nil {
			changedLeaders = append(changedLeaders, *changed)
		}
	}

	// Отправляем уведомления о сменах лидеров
	if len(changedLeaders) > 0 {
		lt.sendLeaderChangeNotifications(changedLeaders)
	}

	log.Printf("[leader_tracker] Check complete, %d leader changes detected", len(changedLeaders))
}

type LeaderChange struct {
	AchievementCode  string
	AchievementTitle string
	OldLeader        string
	OldValue         string
	NewLeader        string
	NewValue         string
	NumericValue     float64
}

func (lt *LeaderTracker) checkAchievementLeader(
	ach *achievements.WeeklyAchievement,
	weekStart time.Time,
) *LeaderChange {
	// Получаем текущего сохранённого лидера
	currentLeader, err := lt.store.GetCurrentLeader(ach.Code)
	if err != nil {
		log.Printf("[leader_tracker] Error getting current leader for %s: %v", ach.Code, err)
		return nil
	}

	// Извлекаем числовое значение из достижения
	numericValue := extractNumericValue(ach)

	// Если лидера ещё нет, сохраняем текущего
	if currentLeader == nil {
		err := lt.store.UpdateCurrentLeader(
			ach.Code,
			ach.UserID,
			ach.Username,
			ach.Value,
			numericValue,
			weekStart,
		)
		if err != nil {
			log.Printf("[leader_tracker] Error updating leader for %s: %v", ach.Code, err)
		}
		log.Printf("[leader_tracker] Initial leader set for %s: %s", ach.Code, ach.Username)
		return nil
	}

	// Проверяем, сменился ли лидер
	if currentLeader.UserID != ach.UserID {
		// Лидер сменился!
		change := &LeaderChange{
			AchievementCode:  ach.Code,
			AchievementTitle: ach.Title,
			OldLeader:        currentLeader.Username,
			OldValue:         currentLeader.Value,
			NewLeader:        ach.Username,
			NewValue:         ach.Value,
			NumericValue:     numericValue,
		}

		// Обновляем лидера в БД
		err := lt.store.UpdateCurrentLeader(
			ach.Code,
			ach.UserID,
			ach.Username,
			ach.Value,
			numericValue,
			weekStart,
		)
		if err != nil {
			log.Printf("[leader_tracker] Error updating leader for %s: %v", ach.Code, err)
		}

		log.Printf("[leader_tracker] Leader changed for %s: %s -> %s",
			ach.Code, currentLeader.Username, ach.Username)

		return change
	}

	// Лидер тот же, но обновляем значение
	if currentLeader.Value != ach.Value {
		err := lt.store.UpdateCurrentLeader(
			ach.Code,
			ach.UserID,
			ach.Username,
			ach.Value,
			numericValue,
			weekStart,
		)
		if err != nil {
			log.Printf("[leader_tracker] Error updating leader value for %s: %v", ach.Code, err)
		}
	}

	return nil
}

// getCurrentWeekStart возвращает начало текущей недели (понедельник 00:00)
func (lt *LeaderTracker) getCurrentWeekStart() time.Time {
	now := time.Now()

	weekday := now.Weekday()

	// В Go воскресенье = 0, понедельник = 1
	// Нужно вычислить количество дней назад до понедельника
	var daysFromMonday int
	if weekday == time.Sunday {
		daysFromMonday = 6 // воскресенье → 6 дней назад до понедельника
	} else {
		daysFromMonday = int(weekday) - 1 // вторник = 2, значит 1 день назад
	}

	weekStart := now.AddDate(0, 0, -daysFromMonday)
	return time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
}

func (lt *LeaderTracker) sendLeaderChangeNotifications(changes []LeaderChange) {
	for _, change := range changes {
		message := formatLeaderChangeMessage(change)

		_, err := lt.session.ChannelMessageSend(lt.channelID, message)
		if err != nil {
			log.Printf("[leader_tracker] Error sending notification: %v", err)
		}
	}
}

func formatLeaderChangeMessage(change LeaderChange) string {
	return fmt.Sprintf(
		"🔄 **Смена лидера: %s**\n\n"+
			"❌ Старый лидер: **%s** (%s)\n"+
			"✅ Новый лидер: **%s** (%s)\n\n"+
			"_Проверка %s_",
		change.AchievementTitle,
		change.OldLeader,
		change.OldValue,
		change.NewLeader,
		change.NewValue,
		time.Now().Format("15:04 02.01"),
	)
}

// extractNumericValue извлекает числовое значение из строки достижения
// Примеры: "15 смен игр" -> 15, "18.5 ч" -> 18.5
func extractNumericValue(ach *achievements.WeeklyAchievement) float64 {
	var value float64

	// Пытаемся извлечь число из Value
	// Для разных форматов: "15 смен игр", "18.5 ч", "Dota 2 — 18.0 ч"
	_, err := fmt.Sscanf(ach.Value, "%f", &value)
	if err == nil {
		return value
	}

	// Для формата "Game — 18.0 ч"
	var gameName string
	_, err = fmt.Sscanf(ach.Value, "%s — %f", &gameName, &value)
	if err == nil {
		return value
	}

	return 0
}

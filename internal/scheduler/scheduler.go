package scheduler

import (
	"brochievements/internal/achievements"
	"brochievements/internal/ai"
	"brochievements/internal/storage"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

type WeeklyScheduler struct {
	session       *discordgo.Session
	store         *storage.Storage
	channelID     string
	generator     ai.Generator
	stopChan      chan struct{}
	publishHour   int          // Час публикации (0-23)
	publishMinute int          // Минута публикации (0-59)
	publishDay    time.Weekday // День недели (0=Sunday, 6=Saturday)
}

func NewWeeklyScheduler(
	session *discordgo.Session,
	store *storage.Storage,
	channelID string,
	generator ai.Generator,
	publishHour int,
	publishMinute int,
	publishDay time.Weekday,
) *WeeklyScheduler {
	return &WeeklyScheduler{
		session:       session,
		store:         store,
		channelID:     channelID,
		generator:     generator,
		stopChan:      make(chan struct{}),
		publishHour:   publishHour,
		publishMinute: publishMinute,
		publishDay:    publishDay,
	}
}

// Start запускает планировщик в отдельной горутине
func (ws *WeeklyScheduler) Start() {
	go ws.run()
	log.Println("[scheduler] Weekly achievements scheduler started")
}

// Stop останавливает планировщик
func (ws *WeeklyScheduler) Stop() {
	close(ws.stopChan)
	log.Println("[scheduler] Weekly achievements scheduler stopped")
}

func (ws *WeeklyScheduler) run() {
	for {
		next := ws.nextPublishTime()
		duration := time.Until(next)

		log.Printf("[scheduler] Next weekly publish: %s (in %s)",
			next.Format("2006-01-02 15:04:05"),
			duration.Round(time.Minute),
		)

		select {
		case <-time.After(duration):
			ws.publishWeekly()
		case <-ws.stopChan:
			return
		}
	}
}

// nextPublishTime вычисляет следующее время публикации
func (ws *WeeklyScheduler) nextPublishTime() time.Time {
	now := time.Now()
	targetDay := ws.publishDay
	targetHour := ws.publishHour
	targetMinute := ws.publishMinute

	// Вычисляем дату целевого дня на ЭТОЙ неделе
	currentWeekday := int(now.Weekday())
	targetWeekday := int(targetDay)

	daysUntilTarget := (targetWeekday - currentWeekday + 7) % 7

	targetDate := now.AddDate(0, 0, daysUntilTarget)
	publishTime := time.Date(
		targetDate.Year(), targetDate.Month(), targetDate.Day(),
		targetHour, targetMinute, 0, 0,
		now.Location(),
	)

	// Если целевое время уже прошло (сегодня и час прошёл, или день на этой неделе прошёл)
	if now.After(publishTime) || now.Equal(publishTime) {
		// Берём следующую неделю
		publishTime = publishTime.AddDate(0, 0, 7)
	}

	return publishTime
}

func (ws *WeeklyScheduler) publishWeekly() {
	log.Println("[scheduler] Publishing weekly achievements...")

	stats, err := achievements.LoadWeeklyStats(ws.store)
	if err != nil {
		log.Println("[scheduler] Error loading weekly stats:", err)
		return
	}

	achs := achievements.RunWeeklyAll(stats)

	if len(achs) == 0 {
		log.Println("[scheduler] No achievements this week")
		ws.sendNoAchievementsMessage()
		return
	}

	ws.sendWeeklyEmbed(achs)
	log.Printf("[scheduler] Published %d achievements", len(achs))
}

func (ws *WeeklyScheduler) sendWeeklyEmbed(achs []*achievements.WeeklyAchievement) {
	achievements.SendWeeklyEmbed(ws.session, ws.channelID, achs, ws.generator, ws.store)
}

func (ws *WeeklyScheduler) sendNoAchievementsMessage() {
	_, err := ws.session.ChannelMessageSend(
		ws.channelID,
		"📊 Итоги недели\n\nНикто ничего не заслужил 😈",
	)
	if err != nil {
		log.Println("[scheduler] Error sending no achievements message:", err)
	}
}

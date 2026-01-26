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
	publishHour   int
	publishMinute int
	publishDay    time.Weekday
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
		generator:     generator, // ← ЭТА СТРОКА ОТСУТСТВОВАЛА!
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

// nextPublishTime вычисляет следующее время публикации (воскресенье 18:00)
func (ws *WeeklyScheduler) nextPublishTime() time.Time {
	now := time.Now()
	targetDay := ws.publishDay
	targetHour := ws.publishHour
	targetMinute := ws.publishMinute

	// Находим ближайшее воскресенье
	daysUntilSunday := (int(targetDay) - int(now.Weekday())) % 7
	if daysUntilSunday == 0 {
		// Сегодня воскресенье
		publishTime := time.Date(
			now.Year(), now.Month(), now.Day(),
			targetHour, targetMinute, 0, 0,
			now.Location(),
		)

		// Если 18:00 уже прошло — берём следующее воскресенье
		if now.After(publishTime) {
			daysUntilSunday = 7
		} else {
			return publishTime
		}
	}

	// Вычисляем дату следующего воскресенья в 18:00
	nextSunday := now.AddDate(0, 0, daysUntilSunday)
	return time.Date(
		nextSunday.Year(), nextSunday.Month(), nextSunday.Day(),
		targetHour, targetMinute, 0, 0,
		now.Location(),
	)
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

package main

import (
	"brochievements/internal/achievements"
	"brochievements/internal/ai"
	"brochievements/internal/scheduler"
	"brochievements/internal/storage"
	"brochievements/internal/version"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {

	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.Version)
		os.Exit(0)
	}

	store := storage.New("brochievements.db")
	defer store.DB.Close()

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	generator, err := ai.NewOpenAIGenerator()
	if err != nil {
		log.Println("[AI] disabled:", err)
		generator = nil
	} else {
		log.Println("[AI] enabled")
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN not set")
	}
	channelID := os.Getenv("ACHIEVEMENTS_CHANNEL_ID")
	if channelID == "" {
		log.Fatal("ACHIEVEMENTS_CHANNEL_ID not set")
	}

	guildID := os.Getenv("GUILD_ID")
	if guildID == "" {
		log.Fatal("GUILD_ID is not set")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("error creating Discord session:", err)
	}

	if err := store.CloseUnfinishedGameSessions(); err != nil {
		log.Fatal("failed to close unfinished game sessions:", err)
	}

	if err := store.CloseUnfinishedAFKGameSessions(); err != nil {
		log.Println("[startup] close unfinished AFK sessions error:", err)
	}

	if err := store.CloseUnfinishedStreamSessions(); err != nil {
		log.Fatal("failed to close unfinished stream sessions:", err)
	}
	if err := store.CloseUnfinishedStreamViews(); err != nil {
		log.Fatal("failed to close unfinished stream views:", err)
	}

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "stats",
			Description: "Показать мою статистику",
		},
		{
			Name:        "weekly_stats",
			Description: "Показать мою статистику с начала недели",
		},
		{
			Name:        "my_achievements",
			Description: "Мои достижения за всё время",
		},
	}

	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentGuildPresences

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.GuildID == "" {
			return
		}

		// ignore other servers
		if m.GuildID != guildID {
			return
		}

		messageCreate(s, m, store)
	})

	dg.AddHandler(func(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {

		if v.GuildID != guildID {
			return
		}

		handleVoiceState(s, v, store)
	})

	dg.AddHandler(onPresenceUpdate(store, guildID))

	dg.AddHandler(func(
		s *discordgo.Session,
		i *discordgo.InteractionCreate,
	) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		switch i.ApplicationCommandData().Name {
		case "stats":
			handleStats(s, i, store, false)
		case "weekly_stats":
			handleStats(s, i, store, true)
		case "my_achievements":
			handleMyAchievements(s, i, store)
		}
	})

	if err = dg.Open(); err != nil {
		log.Fatal("error opening connection:", err)
	}

	if err := closeUnfinishedGameSessions(store); err != nil {
		log.Fatal(err)
	}

	for _, cmd := range commands {
		_, err := dg.ApplicationCommandCreate(
			dg.State.User.ID,
			"",
			cmd,
		)
		if err != nil {
			log.Fatalf("Cannot create command %q: %v", cmd.Name, err)
		}
	}

	log.Println("Starting Brochievements", version.Version)
	log.Println("Brochievements bot is running. Press CTRL-C to exit.")

	notifyIfUpdated(dg)

	weeklyScheduler := scheduler.NewWeeklyScheduler(dg, store, channelID, generator)
	weeklyScheduler.Start()
	defer weeklyScheduler.Stop()

	// ------ удалить
	// ===== WEEKLY DEBUG RUN (ONE-TIME) =====

	go func() {

		achievements.PublishWeeklyDebug(store, generator)

		// achievements.PublishWeeklyDebug(
		// 	dg,    // *discordgo.Session
		// 	store, // *storage.Storage
		// 	channelID,
		// 	generator,
		// )
	}()
	// ------ удалить

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	dg.Close()
}

func extractGame(p *discordgo.Presence) string {
	for _, a := range p.Activities {
		log.Printf(
			"[presence] activity: type=%v name=%q applicationID=%q",
			a.Type,
			a.Name,
			a.ApplicationID,
		)

		if a.Name != "" {
			return a.Name
		}
	}
	return ""
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate, store *storage.Storage) {
	if m.Author.Bot {
		return
	}

	err := store.SaveMessage(
		m.Author.ID,
		m.Author.Username,
		m.ChannelID,
		m.Content,
	)

	if err != nil {
		log.Println("DB error:", err)
	}

	log.Printf(
		"[Brochievements] %s: %s",
		m.Author.Username,
		m.Content,
	)
}

func handleVoiceState(
	s *discordgo.Session,
	v *discordgo.VoiceStateUpdate,
	store *storage.Storage,
) {
	userID := v.UserID
	if v.Member == nil || v.Member.User == nil {
		log.Println("[voice][user] Member or User is nil, skipping UpsertUser")
		return
	}

	var username string

	// 1️⃣ Ник на сервере
	if v.Member.Nick != "" {
		username = v.Member.Nick
	} else if v.Member.User.GlobalName != "" {
		// 2️⃣ Глобальное отображаемое имя
		username = v.Member.User.GlobalName
	} else {
		// 3️⃣ Фолбэк — старый username
		username = v.Member.User.Username
	}

	joinedAt := v.Member.JoinedAt

	log.Printf(
		"[voice][user] UpsertUser: userID=%s username=%q joinedAt=%v",
		userID,
		username,
		joinedAt,
	)

	if err := store.UpsertUser(
		userID,
		username,
		&joinedAt,
	); err != nil {
		log.Println("[voice][user] UpsertUser ERROR:", err)
	}

	if v.ChannelID != "" {
		ch, err := s.Channel(v.ChannelID)
		if err == nil {
			store.UpsertVoiceChannel(ch.ID, ch.Name)
		}
	}

	// ---- STREAM tracking (Go Live) ---- start
	wasStreaming := false
	wasChannelID := ""
	if v.BeforeUpdate != nil {
		wasStreaming = v.BeforeUpdate.SelfStream
		wasChannelID = v.BeforeUpdate.ChannelID
	}
	isStreaming := v.SelfStream

	// 1) старт/стоп стрима
	if !wasStreaming && isStreaming && v.ChannelID != "" {
		if err := store.StartStreamSession(userID, username, v.ChannelID); err != nil {
			log.Println("[stream] StartStreamSession error:", err)
		}
		log.Println("[stream] START:", username, userID, "channel", v.ChannelID)
	}

	if wasStreaming && !isStreaming {
		if err := store.EndStreamSession(userID); err != nil {
			log.Println("[stream] EndStreamSession error:", err)
		}
		// когда стример выключил стрим — всем зрителям закрываем просмотр
		if err := store.EndAllViewsForStreamer(userID); err != nil {
			log.Println("[stream] EndAllViewsForStreamer error:", err)
		}
		log.Println("[stream] END:", username, userID)
	}

	// 2) если пользователь сменил voice-канал — закрываем просмотры в прошлом канале
	if wasChannelID != "" && v.ChannelID != "" && wasChannelID != v.ChannelID {
		if err := store.EndAllViewsForViewerInChannel(userID, wasChannelID); err != nil {
			log.Println("[stream] EndAllViewsForViewerInChannel(move) error:", err)
		}
	}

	// 3) если вышел из voice — закрываем все просмотры
	if v.ChannelID == "" {
		if err := store.EndAllViewsForViewer(userID); err != nil {
			log.Println("[stream] EndAllViewsForViewer(leave) error:", err)
		}
	} else {
		// иначе синхронизируем просмотры в текущем канале
		if err := store.SyncStreamViews(userID, v.ChannelID, v.SelfStream, v.SelfDeaf); err != nil {
			log.Println("[stream] SyncStreamViews error:", err)
		}
	}

	// ---- STREAM tracking (Go Live) ---- end

	// 🔊 JOIN: не было канала → появился
	if v.BeforeUpdate == nil && v.ChannelID != "" {
		log.Println("Voice join:", username, userID, "channel", v.ChannelID)

		store.StartVoiceSession(userID, username, v.ChannelID)

		if err := store.EnsureVoiceChannelSession(userID, v.ChannelID); err != nil {
			log.Println("EnsureVoiceChannelSession error:", err)
		}
		return
	}

	// 🔄 MOVE: был канал → другой канал
	if v.BeforeUpdate != nil &&
		v.BeforeUpdate.ChannelID != "" &&
		v.ChannelID != "" &&
		v.BeforeUpdate.ChannelID != v.ChannelID {

		log.Println("Voice move:", username, userID,
			v.BeforeUpdate.ChannelID, "→", v.ChannelID)

		if err := store.EnsureVoiceChannelSession(userID, v.ChannelID); err != nil {
			log.Println("EnsureVoiceChannelSession error:", err)
		}
		return
	}

	// 🔇 LEAVE: был канал → пусто
	if v.BeforeUpdate != nil && v.ChannelID == "" {
		log.Println("Voice leave:", username, userID)

		// закрываем channel-сессию
		active, err := store.GetActiveVoiceChannelSession(userID)
		if err == nil && active != nil {
			_ = store.EndVoiceChannelSession(active.ID)
		}

		// закрываем общую voice-сессию
		store.EndVoiceSession(userID)
		return
	}
}

func handleStats(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	store *storage.Storage,
	isWeeky bool,
) {
	user := i.Member.User
	userID := user.ID

	var stats *storage.UserStats
	var err error

	if isWeeky {
		stats, err = store.GetUserStatsWeekly(userID)
	} else {
		stats, err = store.GetUserStats(userID)
	}

	// --- Форматирование времени ---
	voiceHours := float64(stats.VoiceSeconds) / 3600
	gameHours := float64(stats.GameSeconds) / 3600
	favChanHours := float64(stats.FavoriteChannelSec) / 3600
	favGameHours := float64(stats.FavoriteGameSec) / 3600

	// --- Безопасные фолбэки ---
	favChannel := "—"
	if stats.FavoriteChannel != "" {
		favChannel = fmt.Sprintf(
			"%s (%.1f ч)",
			stats.FavoriteChannel,
			favChanHours,
		)
	}

	favGame := "—"
	if stats.FavoriteGame != "" {
		favGame = fmt.Sprintf(
			"%s (%.1f ч)",
			stats.FavoriteGame,
			favGameHours,
		)
	}

	joinedText := "—"
	if stats.JoinedAt != "" {
		if t, err := time.Parse(time.RFC3339, stats.JoinedAt); err == nil {
			joinedText = t.Format("02.01.2006")
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📊 Статистика: %s", user.Username),
		Color: 0x5865F2,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: user.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💬 Сообщения",
				Value:  fmt.Sprintf("%d", stats.MessagesCount),
				Inline: true,
			},
			{
				Name:   "🗓 На сервере с",
				Value:  joinedText,
				Inline: true,
			},
			{
				Name: "🎧 Voice",
				Value: fmt.Sprintf(
					"⏱ Всего: %.1f ч\n"+
						"🔊 Любимый канал: %s\n"+
						"🔁 Входов: %d",
					voiceHours,
					favChannel,
					stats.VoiceJoins,
				),
				Inline: false,
			},
			{
				Name: "🎮 Игры",
				Value: fmt.Sprintf(
					"⏱ Всего: %.1f ч\n"+
						"🎯 Любимая игра: %s\n"+
						"🔄 Смен игр: %d",
					gameHours,
					favGame,
					stats.GameSwitches,
				),
				Inline: false,
			},
		},
	}

	err = s.InteractionRespond(
		i.Interaction,
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		},
	)

	if err != nil {
		log.Println("InteractionRespond error:", err)
	}
}

func handleMyAchievements(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	store *storage.Storage,
) {
	user := i.Member.User
	userID := user.ID

	// Получаем достижения из БД
	achievements, err := store.GetUserAchievements(userID)
	if err != nil {
		log.Println("GetUserAchievements error:", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Ошибка при получении достижений",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Если достижений нет
	if len(achievements) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "🏆 У тебя пока нет достижений. Продолжай активничать!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Группируем достижения по коду для подсчёта
	achievementCounts := make(map[string]int)
	achievementTitles := make(map[string]string)
	latestAchievements := make(map[string]storage.AchievementRecord)

	for _, ach := range achievements {
		achievementCounts[ach.AchievementCode]++
		achievementTitles[ach.AchievementCode] = ach.AchievementTitle

		// Сохраняем самое свежее достижение каждого типа
		if _, exists := latestAchievements[ach.AchievementCode]; !exists {
			latestAchievements[ach.AchievementCode] = ach
		}
	}

	// Формируем список достижений
	var achievementList []string
	totalCount := 0

	for code, count := range achievementCounts {
		title := achievementTitles[code]
		latest := latestAchievements[code]

		var line string
		if count > 1 {
			line = fmt.Sprintf(
				"%s **×%d**\n_Последнее: %s (%s)_",
				title,
				count,
				latest.Value,
				latest.AwardedAt.Format("02.01.2006"),
			)
		} else {
			line = fmt.Sprintf(
				"%s\n_Получено: %s (%s)_",
				title,
				latest.Value,
				latest.AwardedAt.Format("02.01.2006"),
			)
		}

		achievementList = append(achievementList, line)
		totalCount += count
	}

	// Формируем embed
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🏆 Достижения: %s", user.Username),
		Color: 0xFFD700, // золотой цвет
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: user.AvatarURL(""),
		},
		Description: fmt.Sprintf(
			"**Всего получено:** %d достижений\n**Уникальных:** %d типов\n\n%s",
			totalCount,
			len(achievementCounts),
			strings.Join(achievementList, "\n\n"),
		),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Продолжай в том же духе! 💪",
		},
	}

	// Если достижений слишком много — ограничиваем
	if len(embed.Description) > 4000 {
		embed.Description = embed.Description[:3900] + "\n\n_... и ещё несколько достижений_"
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		log.Println("InteractionRespond error:", err)
	}
}

func closeUnfinishedGameSessions(store *storage.Storage) error {
	_, err := store.DB.Exec(`
		UPDATE game_sessions
		SET ended_at = CURRENT_TIMESTAMP
		WHERE ended_at IS NULL
	`)
	return err
}

func onPresenceUpdate(store *storage.Storage, guildID string) func(*discordgo.Session, *discordgo.PresenceUpdate) {
	return func(s *discordgo.Session, p *discordgo.PresenceUpdate) {

		if p.GuildID != guildID {
			return
		}

		if p.User == nil {
			return
		}

		userID := p.User.ID
		username := p.User.Username
		newGame := extractGame(&p.Presence)

		status := p.Status
		isActive := status == discordgo.StatusOnline || status == discordgo.StatusDoNotDisturb

		// =========================================================
		// 🥔 AFK LOGIC START
		// =========================================================

		// 🔴 Пользователь стал idle / offline С ЗАПУЩЕННОЙ ИГРОЙ
		if !isActive && newGame != "" {
			// старт AFK-сессии (если ещё нет)
			_ = store.StartAFKGameSession(userID, username, newGame)

			// закрываем обычную игровую сессию
			active, err := store.GetActiveGameSession(userID)
			if err == nil && active != nil {
				_ = store.EndGameSession(active.ID)
				log.Println("[presence] game closed due to idle:", userID)
			}
			return
		}

		// 🟢 Пользователь снова активен ИЛИ игра пропала → закрываем AFK
		afk, _ := store.GetActiveAFKGameSession(userID)
		if afk != nil && (isActive || newGame == "") {
			_ = store.EndAFKGameSession(afk.ID)
			log.Println("[presence] AFK game closed:", userID)
		}

		// =========================================================
		// 🎮 GAME LOGIC (твоя исходная логика)
		// =========================================================

		// ❌ Активен, но НЕ играет → закрываем игру
		if newGame == "" {
			active, err := store.GetActiveGameSession(userID)
			if err == nil && active != nil {
				_ = store.EndGameSession(active.ID)
				log.Println("[presence] game closed (no activity):", userID)
			}
			return
		}

		// ▶️ Активен + играет → старт / продолжение
		if err := store.EnsureGameSession(userID, newGame); err != nil {
			log.Println("[presence] EnsureGameSession error:", err)
		} else {
			log.Println("[presence] EnsureGameSession OK:", userID, newGame)
		}
	}
}

const (
	versionFile        = "/opt/brochievements/.version"
	versionChangedFile = "/opt/brochievements/.version_changed"
)

func notifyIfUpdated(s *discordgo.Session) {
	if _, err := os.Stat(versionChangedFile); err != nil {
		return // версия не менялась
	}

	data, err := os.ReadFile(versionFile)
	if err != nil {
		log.Println("cannot read version:", err)
		return
	}
	deployedVersion := strings.TrimSpace(string(data))

	channelID := os.Getenv("ACHIEVEMENTS_CHANNEL_ID")
	if channelID == "" {
		log.Println("ACHIEVEMENTS_CHANNEL_ID is not set")
		return
	}

	msg := formatDeployMessage(
		deployedVersion,
		version.ChangeNotes,
	)

	_, err = s.ChannelMessageSend(channelID, msg)
	if err != nil {
		log.Println("failed to send deploy message:", err)
		return
	}

	// важно: чтобы не спамить при рестартах
	_ = os.Remove(versionChangedFile)
}

func formatDeployMessage(ver string, notes []string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("🚀 **Бот обновлён до версии `%s`**\n\n", ver))

	if len(notes) > 0 {
		b.WriteString("📝 **Изменения:**\n")
		for _, note := range notes {
			b.WriteString("• " + note + "\n")
		}
	}

	return b.String()
}

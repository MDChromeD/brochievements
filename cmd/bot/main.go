package main

import (
	"brochievements/internal/achievements"
	"brochievements/internal/ai"
	"brochievements/internal/storage"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {

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

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "stats",
			Description: "Показать мою статистику",
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
			handleStats(s, i, store)
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

	log.Println("Brochievements bot is running. Press CTRL-C to exit.")

	// ------ удалить
	// ===== WEEKLY DEBUG RUN (ONE-TIME) =====

	go func() {

		achievements.PublishWeeklyDebug(
			dg,    // *discordgo.Session
			store, // *storage.Storage
			channelID,
			generator,
		)
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
) {
	user := i.Member.User
	userID := user.ID

	stats, err := store.GetUserStats(userID)
	if err != nil {
		log.Println("GetUserStats error:", err)
		return
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

		log.Printf("[presence] event: user=%v guild=%v activities=%d",
			func() string {
				if p.User != nil {
					return p.User.ID
				}
				return "<nil>"
			}(),
			p.GuildID,
			len(p.Activities),
		)

		if p.User == nil {
			return
		}

		userID := p.User.ID
		newGame := extractGame(&p.Presence)

		// ❌ пользователь не играет → закрываем, если было
		if newGame == "" {
			active, err := store.GetActiveGameSession(userID)
			if err != nil || active == nil {
				return
			}
			_ = store.EndGameSession(active.ID)
			return
		}

		// ▶️ пользователь играет → storage сам решит, что делать
		if err := store.EnsureGameSession(userID, newGame); err != nil {
			log.Println("[presence] EnsureGameSession error:", err)
		} else {
			log.Println("[presence] EnsureGameSession OK:", userID, newGame)
		}

	}
}

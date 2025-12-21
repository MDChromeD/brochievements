package main

import (
	"brochievements/internal/achievements"
	"brochievements/internal/storage"
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

	store := storage.New("brochievements.db")
	defer store.DB.Close()

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN not set")
	}
	channelID := os.Getenv("ACHIEVEMENTS_CHANNEL_ID")
	if channelID == "" {
		log.Fatal("ACHIEVEMENTS_CHANNEL_ID not set")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("error creating Discord session:", err)
	}

	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentGuildPresences

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		messageCreate(s, m, store)
	})

	dg.AddHandler(func(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
		handleVoiceState(s, v, store)
	})

	dg.AddHandler(func(
		s *discordgo.Session,
		p *discordgo.PresenceUpdate,
	) {
		handlePresence(s, p, store)
	})

	if err = dg.Open(); err != nil {
		log.Fatal("error opening connection:", err)
	}

	log.Println("Brochievements bot is running. Press CTRL-C to exit.")

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		log.Println("Weekly achievements scheduler started")

		// 🔹 Опционально: первый запуск сразу
		//publishWeeklyAchievements(dg, store, channelID)

		for {
			<-ticker.C
			publishWeeklyAchievements(dg, store, channelID)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	dg.Close()
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
	username := ""

	if v.Member != nil && v.Member.User != nil {
		username = v.Member.User.Username
	}

	// Пользователь зашёл в voice
	if v.BeforeUpdate == nil && v.ChannelID != "" {
		log.Println("Voice join:", username)
		store.StartVoiceSession(userID, username, v.ChannelID)
		return
	}

	// Пользователь вышел из voice
	if v.ChannelID == "" {
		log.Println("Voice leave:", username)
		store.EndVoiceSession(userID)
		return
	}
}

func publishWeeklyAchievements(
	dg *discordgo.Session,
	store *storage.Storage,
	channelID string,
) {
	var achievementsList []achievements.Achievement

	if stat, err := store.TopVoiceUserLastWeek(); err == nil {
		achievementsList = append(
			achievementsList,
			achievements.VoiceMaster(stat),
		)
	}

	if stat, err := store.TopVoiceJoinsLastWeek(); err == nil {
		achievementsList = append(
			achievementsList,
			achievements.FrequentVisitor(stat),
		)
	}

	if stat, err := store.LongestVoiceSessionLastWeek(); err == nil {
		achievementsList = append(
			achievementsList,
			achievements.Marathoner(stat),
		)
	}

	if stat, err := store.TopGameLastWeek(); err == nil {
		achievementsList = append(
			achievementsList,
			achievements.GameFan(stat),
		)
	}

	if len(achievementsList) == 0 {
		log.Println("No achievements to publish")
		return
	}

	var message strings.Builder
	message.WriteString("🏆 **Итоги недели**\n\n")

	for _, ach := range achievementsList {
		message.WriteString(fmt.Sprintf(
			"**%s**\n%s\n\n",
			ach.Title,
			ach.Description,
		))
	}

	_, err := dg.ChannelMessageSend(channelID, message.String())
	if err != nil {
		log.Println("Failed to post achievements:", err)
	} else {
		log.Println("Weekly achievements posted")
	}
}

func handlePresence(
	s *discordgo.Session,
	p *discordgo.PresenceUpdate,
	store *storage.Storage,
) {
	if p.User == nil {
		return
	}

	username := p.User.Username
	userID := p.User.ID

	for _, activity := range p.Activities {
		if activity.Type == discordgo.ActivityTypeGame && activity.Name != "" {
			log.Printf(
				"Game detected: %s is playing %s",
				username,
				activity.Name,
			)

			err := store.SaveGameActivity(
				userID,
				username,
				activity.Name,
			)
			if err != nil {
				log.Println("Game activity save error:", err)
			}
		}
	}
}

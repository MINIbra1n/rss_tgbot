package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"news-feed-bot/internal/bot"
	"news-feed-bot/internal/bot/middleware"
	"news-feed-bot/internal/botkit"
	"news-feed-bot/internal/config"
	"news-feed-bot/internal/fetcher"
	"news-feed-bot/internal/notifier"
	"news-feed-bot/internal/storage"
	"news-feed-bot/internal/summary"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	botAPI, err := tgbotapi.NewBotAPI(config.Get().TelegramBotToken)
	if err != nil {
		log.Printf("failed to create bot :%v", err)
		return
	}

	db, err := sqlx.Connect("postgres", config.Get().DataBaseDSN)
	if err != nil {
		log.Printf("failed to connect db :%v", err)
		return
	}
	defer db.Close()

	var (
		articleStorage = storage.NewArticleStorage(db)
		sourcesStorage = storage.NewSourceStorage(db)
		fetcher        = fetcher.New(
			articleStorage,
			sourcesStorage,
			config.Get().FetchInterval,
			config.Get().FilterKeyWords,
		)
		summarizer = summary.NewOpenAISummarizer(
			config.Get().OpenAIKey,
			config.Get().OpenAIModel,
			config.Get().OpenAIPrompt,
		)
		notifier = notifier.New(
			articleStorage,
			summarizer,
			botAPI,
			config.Get().NotificationInterval,
			2*config.Get().FetchInterval,
			config.Get().TelegramChannelID,
		)
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	newsBot := botkit.New(botAPI)
	newsBot.RegisterCmdView("start", bot.ViewCmdStart())
	newsBot.RegisterCmdView("addsource", middleware.AdminOnly(config.Get().TelegramChannelID, bot.ViewCmdAddSource(sourcesStorage)))
	newsBot.RegisterCmdView("givepost", middleware.AdminOnly(config.Get().TelegramChannelID, func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		notifier.SelectAllArticleAndSendOne(ctx)
		return nil
	}))
	newsBot.RegisterCmdView("list", bot.ViewCmdListSource(sourcesStorage))
	go func(ctx context.Context) {
		if err := fetcher.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("[ERROR] failed to start fetcher %v", err)
				return
			}
		}

	}(ctx)
	// go func(ctx context.Context) {
	// 	if err := notifier.SelectArticleAll(ctx); err != nil {
	// 		return
	// 	}
	// }(ctx)
	go func(ctx context.Context) {
		if err := notifier.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("[ERROR] failed to run notifier: %v", err)
				return
			}

			log.Printf("[INFO] notifier stopped")
		}
	}(ctx)
	// go func(ctx context.Context) {
	// 	if err := http.ListenAndServe("9.0.0.0:8080", mux); err != nil {
	// 		if !errors.Is(err, context.Canceled) {
	// 			log.Printf("[ERROR] failed to run http server: %v", err)
	// 			return
	// 		}

	// 		log.Printf("[INFO] http server stopped")
	// 	}
	// }(ctx)

	if err := newsBot.Run(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Printf("[ERROR] failed to start bot: %v", err)
			return
		}
		log.Printf("bot stoped")
	}
}

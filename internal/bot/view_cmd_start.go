package bot

import (
	"context"
	"news-feed-bot/internal/botkit"
	"news-feed-bot/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ArticleStorage interface {
	AllPosted(ctx context.Context) ([]model.Article, error)
}

type Article struct {
	ArticleStorage
}

func ViewCmdStart() botkit.ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {

		// _, err :=
		// if err != nil {
		// 	return err
		// }

		if _, err := bot.Send(tgbotapi.NewMessage(update.FromChat().ID, "Priv")); err != nil {
			return err
		}
		return nil
	}
}

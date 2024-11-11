package bot

import (
	"context"
	"fmt"
	"news-feed-bot/internal/botkit"
	"news-feed-bot/internal/botkit/markup"
	"news-feed-bot/internal/model"
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/samber/lo"
)

type SourceLister interface {
	Sources(ctx context.Context) ([]model.Source, error)
}

func ViewCmdListSource(lister SourceLister) botkit.ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		sources, err := lister.Sources(ctx)
		if err != nil {
			return err
		}

		sort.SliceStable(sources, func(i, j int) bool {
			return sources[i].Priority > sources[j].Priority
		})

		var (
			sourceInfos = lo.Map(sources, func(source model.Source, _ int) string { return formatSource(source) })
			msgText     = fmt.Sprintf(
				"Список источников \\(всего %d\\):\n\n%s",
				len(sources),
				strings.Join(sourceInfos, "\n\n"),
			)
		)

		reply := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
		reply.ParseMode = "MarkdownV2"

		if _, err := bot.Send(reply); err != nil {
			return err
		}

		return nil
	}
}

func formatSource(source model.Source) string {
	return fmt.Sprintf(
		"*%s*\nID: `%d`\n URL фида: %s",
		markup.EscapeForMarkdown(source.Name),
		source.ID,
		markup.EscapeForMarkdown(source.FeedURL),
	)
}

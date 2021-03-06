package tags

import (
	"github.com/TicketsBot/common/permission"
	"github.com/TicketsBot/common/sentry"
	translations "github.com/TicketsBot/database/translations"
	"github.com/TicketsBot/worker/bot/command"
	"github.com/TicketsBot/worker/bot/dbclient"
	"github.com/TicketsBot/worker/bot/utils"
	"github.com/rxdn/gdl/objects/channel/embed"
	"strings"
)

type ManageTagsAddCommand struct {
}

func (ManageTagsAddCommand) Properties() command.Properties {
	return command.Properties{
		Name:            "add",
		Description:     translations.HelpTagAdd,
		Aliases:         []string{"new", "create"},
		PermissionLevel: permission.Support,
		Category:        command.Tags,
	}
}

func (ManageTagsAddCommand) Execute(ctx command.CommandContext) {
	usageEmbed := embed.EmbedField{
		Name:   "Usage",
		Value:  "`t!managetags add [TagID] [Tag contents]`",
		Inline: false,
	}

	if len(ctx.Args) < 2 {
		ctx.ReactWithCross()
		ctx.SendEmbedWithFields(utils.Red, "Error", translations.MessageTagCreateInvalidArguments, utils.FieldsToSlice(usageEmbed))
		return
	}

	id := ctx.Args[0]
	content := ctx.Args[1:] // content cannot be bigger than the Discord limit, obviously

	// Length check
	if len(id) > 16 {
		ctx.ReactWithCross()
		ctx.SendEmbedWithFields(utils.Red, "Error", translations.MessageTagCreateTooLong, utils.FieldsToSlice(usageEmbed))
		return
	}

	// Verify a tag with the ID doesn't already exist
	var tagExists bool
	{
		tag, err := dbclient.Client.Tag.Get(ctx.GuildId, id)
		if err != nil {
			sentry.ErrorWithContext(err, ctx.ToErrorContext())
			ctx.ReactWithCross()
			return
		}

		tagExists = tag != ""
	}

	if tagExists {
		ctx.SendEmbedWithFields(utils.Red, "Error", translations.MessageTagCreateAlreadyExists, utils.FieldsToSlice(usageEmbed), id, id)
		ctx.ReactWithCross()
		return
	}

	if err := dbclient.Client.Tag.Set(ctx.GuildId, id, strings.Join(content, " ")); err == nil {
		ctx.ReactWithCheck()
	} else {
		ctx.ReactWithCross()
		sentry.ErrorWithContext(err, ctx.ToErrorContext())
	}
}

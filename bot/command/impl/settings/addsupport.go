package settings

import (
	permcache "github.com/TicketsBot/common/permission"
	"github.com/TicketsBot/common/sentry"
	translations "github.com/TicketsBot/database/translations"
	"github.com/TicketsBot/worker/bot/command"
	"github.com/TicketsBot/worker/bot/dbclient"
	"github.com/TicketsBot/worker/bot/redis"
	"github.com/TicketsBot/worker/bot/utils"
	"github.com/rxdn/gdl/objects/channel"
	"github.com/rxdn/gdl/objects/channel/embed"
	"github.com/rxdn/gdl/permission"
	"github.com/rxdn/gdl/rest"
	"strings"
)

type AddSupportCommand struct {
}

func (AddSupportCommand) Properties() command.Properties {
	return command.Properties{
		Name:            "addsupport",
		Description:     translations.HelpAddSupport,
		Aliases:         []string{"addsuport"},
		PermissionLevel: permcache.Admin,
		Category:        command.Settings,
	}
}

func (AddSupportCommand) Execute(ctx command.CommandContext) {
	usageEmbed := embed.EmbedField{
		Name:   "Usage",
		Value:  "`t!addsupport @User`\n`t!addsupport @Role`\n`t!addsupport role name`",
		Inline: false,
	}

	if len(ctx.Args) == 0 {
		ctx.SendEmbedWithFields(utils.Red, "Error", translations.MessageAddSupportNoMembers, utils.FieldsToSlice(usageEmbed))
		ctx.ReactWithCross()
		return
	}

	user := false
	roles := make([]uint64, 0)

	if len(ctx.Message.Mentions) > 0 {
		user = true
		for _, mention := range ctx.Message.Mentions {
			go func() {
				if err := dbclient.Client.Permissions.AddSupport(ctx.GuildId, mention.Id); err != nil {
					sentry.ErrorWithContext(err, ctx.ToErrorContext())
				}

				if err := permcache.SetCachedPermissionLevel(redis.Client, ctx.GuildId, mention.Id, permcache.Support); err != nil {
					ctx.HandleError(err)
					return
				}
			}()
		}
	} else if len(ctx.Message.MentionRoles) > 0 {
		for _, mention := range ctx.Message.MentionRoles {
			roles = append(roles, mention)
		}
	} else {
		guildRoles, err := ctx.Worker.GetGuildRoles(ctx.GuildId)
		if err != nil {
			sentry.ErrorWithContext(err, ctx.ToErrorContext())
			return
		}

		roleName := strings.ToLower(strings.Join(ctx.Args, " "))

		// Get role ID from name
		valid := false
		for _, role := range guildRoles {
			if strings.ToLower(role.Name) == roleName {
				valid = true
				roles = append(roles, role.Id)
				break
			}
		}

		// Verify a valid role was mentioned
		if !valid {
			ctx.SendEmbedWithFields(utils.Red, "Error", translations.MessageAddSupportNoMembers, utils.FieldsToSlice(usageEmbed))
			ctx.ReactWithCross()
			return
		}
	}

	// Add roles to DB
	for _, role := range roles {
		if err := dbclient.Client.RolePermissions.AddSupport(ctx.GuildId, role); err != nil {
			sentry.ErrorWithContext(err, ctx.ToErrorContext())
		}

		if err := permcache.SetCachedPermissionLevel(redis.Client, ctx.GuildId, role, permcache.Support); err != nil {
			ctx.HandleError(err)
			return
		}
	}

	openTickets, err := dbclient.Client.Tickets.GetGuildOpenTickets(ctx.GuildId)
	if err != nil {
		sentry.ErrorWithContext(err, ctx.ToErrorContext())
	}

	// Update permissions for existing tickets
	for _, ticket := range openTickets {
		if ticket.ChannelId == nil {
			continue
		}

		ch, err := ctx.Worker.GetChannel(*ticket.ChannelId)
		if err != nil {
			continue
		}

		overwrites := ch.PermissionOverwrites

		if user {
			// If adding individual admins, apply each override individually
			for _, mention := range ctx.Message.Mentions {
				overwrites = append(overwrites, channel.PermissionOverwrite{
					Id:    mention.Id,
					Type:  channel.PermissionTypeMember,
					Allow: permission.BuildPermissions(permission.ViewChannel, permission.SendMessages, permission.AddReactions, permission.AttachFiles, permission.ReadMessageHistory, permission.EmbedLinks),
					Deny:  0,
				})
			}
		} else {
			// If adding a role as an admin, apply overrides to role
			for _, role := range roles {
				overwrites = append(overwrites, channel.PermissionOverwrite{
					Id:    role,
					Type:  channel.PermissionTypeRole,
					Allow: permission.BuildPermissions(permission.ViewChannel, permission.SendMessages, permission.AddReactions, permission.AttachFiles, permission.ReadMessageHistory, permission.EmbedLinks),
					Deny:  0,
				})
			}
		}

		data := rest.ModifyChannelData{
			PermissionOverwrites: overwrites,
			Position:             ch.Position,
		}

		if _, err = ctx.Worker.ModifyChannel(*ticket.ChannelId, data); err != nil {
			sentry.ErrorWithContext(err, ctx.ToErrorContext())
		}
	}

	ctx.ReactWithCheck()
}

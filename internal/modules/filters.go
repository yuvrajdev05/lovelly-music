/*
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Arc
 */

package modules

import (
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/config"
	"main/internal/database"
	"main/internal/utils"
)

var (
	superGroupFilter    = tg.Custom(filterSuperGroup)
	adminFilter         = tg.Custom(filterChatAdmins)
	authFilter          = tg.Custom(filterAuthUsers)
	ignoreChannelFilter = tg.Custom(filterChannel)
	sudoOnlyFilter      = tg.Custom(filterSudo)
	ownerFilter         = tg.Custom(filterOwner)
)

func filterSuperGroup(m *tg.NewMessage) bool {
	if !filterChannel(m) {
		return false
	}
	if blocked, err := database.IsBlacklistedChat(m.ChannelID()); err == nil && blocked {
		if m.IsPrivate() {
			m.Reply("🚫 This chat is blocked from using the bot.")
		}
		return false
	}

	switch m.ChatType() {
	case tg.EntityChat:
		// EntityChat can be basic group or supergroup — allow only supergroup
		if m.Channel != nil && !m.Channel.Broadcast {
			database.AddServedChat(m.ChannelID())
			return true // Supergroup
		}
		warnAndLeave(m.Client, m.ChannelID()) // Basic group → leave
		database.RemoveServedChat(m.ChannelID())
		return false

	case tg.EntityChannel:
		return false // Pure channel chat → ignore

	case tg.EntityUser:
		m.Reply(F(m.ChannelID(), "only_supergroup"))
		database.AddServedUser(m.ChannelID())
		return false // Private chat → warn
	}

	return false
}

func filterChatAdmins(m *tg.NewMessage) bool {
	isAdmin, err := utils.IsChatAdmin(m.Client, m.ChannelID(), m.SenderID())
	if err != nil || !isAdmin {
		m.Reply(F(m.ChannelID(), "only_admin"))
		return false
	}
	return true
}

func filterAuthUsers(m *tg.NewMessage) bool {
	isAdmin, err := utils.IsChatAdmin(m.Client, m.ChannelID(), m.SenderID())
	if err == nil && isAdmin {
		return true
	}

	isAuth, err := database.IsAuthorized(m.ChannelID(), m.SenderID())
	if err == nil && isAuth {
		return true
	}

	m.Reply(F(m.ChannelID(), "only_admin_or_auth"))
	return false
}

func filterSudo(m *tg.NewMessage) bool {
	is, _ := database.IsSudo(m.SenderID())

	if config.OwnerID == 0 || (m.SenderID() != config.OwnerID && !is) {
		if m.IsPrivate() ||
			strings.HasSuffix(m.GetCommand(), m.Client.Me().Username) {
			m.Reply(F(m.ChannelID(), "only_sudo"))
		}
		return false
	}

	return true
}

func filterChannel(m *tg.NewMessage) bool {
	if _, ok := m.Message.FromID.(*tg.PeerChannel); ok {
		return false
	}
	// A blacklisted chat is blocked at the common filter layer so every
	// command respects /blchat, not only music/group commands.
	if chatID := m.ChannelID(); chatID != 0 {
		if blocked, err := database.IsBlacklistedChat(chatID); err == nil && blocked {
			return false
		}
	}
	if id := m.SenderID(); id != 0 {
		owner := config.OwnerID != 0 && id == config.OwnerID
		sudo, _ := database.IsSudo(id)
		if !owner && !sudo {
			if blocked, err := database.IsBlockedUser(id); err == nil && blocked {
				return false
			}
		}
	}
	return true
}

func filterOwner(m *tg.NewMessage) bool {
	if config.OwnerID == 0 || m.SenderID() != config.OwnerID {
		if m.IsPrivate() ||
			strings.HasSuffix(m.GetCommand(), m.Client.Me().Username) {
			m.Reply(F(m.ChannelID(), "only_owner"))
		}
		return false
	}
	return true
}

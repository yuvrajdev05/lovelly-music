/*
 * Social/group utilities for ArcMusic.
 *
 * This file is an original Go implementation of group utilities:
 * TagAll/AdminTag, wish tagging, welcome messages and user information.
 */
package modules

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/database"
	"main/internal/utils"
)

var (
	tagMu        sync.Mutex
	activeTags   = map[int64]bool{}
	wishMu       sync.Mutex
	activeWishes = map[int64]bool{}
	welcomeMu    sync.Mutex
	lastWelcome  = map[int64]int64{}
)

var tagEmojis = []string{"🌸", "🦋", "✨", "💫", "🌷", "🌺", "💖", "🌻", "🍀", "🎀", "🌈", "⭐"}

func tagMembers(m *tg.NewMessage, adminsOnly bool, intro string) error {
	chatID := m.ChannelID()
	ok, err := utils.IsChatAdmin(m.Client, chatID, m.SenderID())
	if err != nil || !ok {
		m.Reply("⚠️ Only chat administrators can use this command.")
		return tg.ErrEndGroup
	}

	tagMu.Lock()
	if activeTags[chatID] {
		tagMu.Unlock()
		m.Reply("⚠️ A tagging process is already running. Use /cancel to stop it.")
		return tg.ErrEndGroup
	}
	activeTags[chatID] = true
	tagMu.Unlock()

	go func() {
		defer func() {
			tagMu.Lock()
			delete(activeTags, chatID)
			tagMu.Unlock()
		}()

		var members []*tg.Participant
		var err error
		if adminsOnly {
			members, _, err = m.Client.GetChatMembers(chatID, &tg.ParticipantOptions{Filter: &tg.ChannelParticipantsAdmins{}, Limit: -1, SleepThresholdMs: 3000})
		} else {
			members, _, err = m.Client.GetChatMembers(chatID, &tg.ParticipantOptions{Limit: -1, SleepThresholdMs: 3000})
		}
		if err != nil {
			m.Client.SendMessage(chatID, "❌ Failed to fetch members: "+utils.EscapeHTML(err.Error()))
			return
		}

		total, tagged := len(members), 0
		batch := make([]string, 0, 5)
		for _, member := range members {
			tagMu.Lock()
			running := activeTags[chatID]
			tagMu.Unlock()
			if !running {
				m.Client.SendMessage(chatID, "🛑 Tagging stopped.")
				return
			}
			if member == nil || member.User == nil || member.User.Bot || member.User.Deleted {
				continue
			}
			emoji := tagEmojis[rand.Intn(len(tagEmojis))]
			batch = append(batch, fmt.Sprintf("%s %s", emoji, utils.MentionHTML(member.User)))
			tagged++
			if len(batch) == 5 {
				text := strings.TrimSpace(intro)
				if text != "" {
					text += "\n\n"
				}
				text += strings.Join(batch, " ")
				if _, err := m.Client.SendMessage(chatID, text, &tg.SendOptions{ParseMode: "HTML", LinkPreview: false}); err != nil {
					gologging.ErrorF("tagall send failed in %d: %v", chatID, err)
				}
				batch = batch[:0]
				time.Sleep(2 * time.Second)
			}
		}
		if len(batch) > 0 {
			text := strings.TrimSpace(intro)
			if text != "" {
				text += "\n\n"
			}
			text += strings.Join(batch, " ")
			m.Client.SendMessage(chatID, text, &tg.SendOptions{ParseMode: "HTML", LinkPreview: false})
		}
		label := "members"
		if adminsOnly {
			label = "admins"
		}
		m.Client.SendMessage(chatID, fmt.Sprintf("✅ Tagging completed.\nTotal %s: %d\nTagged: %d", label, total, tagged))
	}()
	return tg.ErrEndGroup
}

func tagAllHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err == nil && reply != nil {
			args = reply.Text()
		}
	}
	if args == "" && !m.IsReply() {
		m.Reply("Usage: <code>/tagall your message</code> or reply to a message with /tagall")
		return tg.ErrEndGroup
	}
	return tagMembers(m, false, args)
}

func adminTagHandler(m *tg.NewMessage) error {
	args := strings.TrimSpace(m.Args())
	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err == nil && reply != nil {
			args = reply.Text()
		}
	}
	if args == "" && !m.IsReply() {
		m.Reply("Usage: <code>/admintag your message</code> or reply to a message")
		return tg.ErrEndGroup
	}
	return tagMembers(m, true, args)
}

func cancelTagHandler(m *tg.NewMessage) error {
	chatID := m.ChannelID()
	ok, _ := utils.IsChatAdmin(m.Client, chatID, m.SenderID())
	if !ok {
		m.Reply("⚠️ Only chat administrators can use this command.")
		return tg.ErrEndGroup
	}
	tagMu.Lock()
	_, running := activeTags[chatID]
	delete(activeTags, chatID)
	tagMu.Unlock()
	if running {
		m.Reply("🛑 Tagging process stopped.")
	} else {
		m.Reply("ℹ️ No tagging process is running.")
	}
	return tg.ErrEndGroup
}

var gmMessages = []string{"🌞 <b>Good Morning</b> 🌼\n\n{mention}", "☕ <b>Rise and Shine!</b>\n\n{mention}", "🌅 <b>New morning, new dreams!</b>\n\n{mention}"}
var gaMessages = []string{"🌞 <b>Good Afternoon</b> ☀️\n\n{mention}", "🍵 <b>Have a lovely afternoon!</b>\n\n{mention}", "🌻 <b>Enjoy your afternoon!</b>\n\n{mention}"}
var gnMessages = []string{"🌙 <b>Good Night</b>\n\n{mention}", "💤 <b>Sweet Dreams!</b> 😴\n\n{mention}", "✨ <b>Good night and sleep well!</b>\n\n{mention}"}

func wishTagHandler(m *tg.NewMessage, messages []string, label string) error {
	chatID := m.ChannelID()
	wishMu.Lock()
	if activeWishes[chatID] {
		wishMu.Unlock()
		m.Reply("⚠️ A wish tagging process is already running.")
		return tg.ErrEndGroup
	}
	activeWishes[chatID] = true
	wishMu.Unlock()
	go func() {
		defer func() { wishMu.Lock(); delete(activeWishes, chatID); wishMu.Unlock() }()
		members, _, err := m.Client.GetChatMembers(chatID, &tg.ParticipantOptions{Limit: -1, SleepThresholdMs: 3000})
		if err != nil {
			m.Client.SendMessage(chatID, "❌ Failed to fetch members: "+utils.EscapeHTML(err.Error()))
			return
		}
		for _, member := range members {
			wishMu.Lock()
			running := activeWishes[chatID]
			wishMu.Unlock()
			if !running {
				m.Client.SendMessage(chatID, "🛑 Wish tagging stopped.")
				return
			}
			if member == nil || member.User == nil || member.User.Bot || member.User.Deleted {
				continue
			}
			text := strings.ReplaceAll(messages[rand.Intn(len(messages))], "{mention}", utils.MentionHTML(member.User))
			m.Client.SendMessage(chatID, text, &tg.SendOptions{ParseMode: "HTML", LinkPreview: false})
			time.Sleep(3 * time.Second)
		}
		m.Client.SendMessage(chatID, "✅ <b>"+label+" tagging completed.</b>", &tg.SendOptions{ParseMode: "HTML"})
	}()
	return tg.ErrEndGroup
}
func gmTagHandler(m *tg.NewMessage) error { return wishTagHandler(m, gmMessages, "Good Morning") }
func gaTagHandler(m *tg.NewMessage) error { return wishTagHandler(m, gaMessages, "Good Afternoon") }
func gnTagHandler(m *tg.NewMessage) error { return wishTagHandler(m, gnMessages, "Good Night") }
func wishStopHandler(m *tg.NewMessage) error {
	wishMu.Lock()
	running := activeWishes[m.ChannelID()]
	delete(activeWishes, m.ChannelID())
	wishMu.Unlock()
	if running {
		m.Reply("🛑 Wish tagging stopped.")
	} else {
		m.Reply("ℹ️ Nothing is running.")
	}
	return tg.ErrEndGroup
}

func welcomeCommandHandler(m *tg.NewMessage) error {
	chatID := m.ChannelID()
	ok, _ := utils.IsChatAdmin(m.Client, chatID, m.SenderID())
	if !ok {
		m.Reply("⚠️ Only chat administrators can use this command.")
		return tg.ErrEndGroup
	}
	arg := strings.TrimSpace(strings.ToLower(m.Args()))
	if arg == "" {
		enabled, _ := database.GetWelcomeEnabled(chatID)
		m.Reply(fmt.Sprintf("Welcome messages are currently <b>%s</b>.\nUsage: <code>/welcome on</code> or <code>/welcome off</code>", utils.IfElse(enabled, "ON", "OFF")))
		return tg.ErrEndGroup
	}
	enabled, err := utils.ParseBool(arg)
	if err != nil {
		m.Reply("⚠️ Use on/off or enable/disable.")
		return tg.ErrEndGroup
	}
	if err := database.SetWelcomeEnabled(chatID, enabled); err != nil {
		m.Reply("❌ Failed to save welcome setting: " + utils.EscapeHTML(err.Error()))
		return tg.ErrEndGroup
	}
	m.Reply(utils.IfElse(enabled, "✅ Welcome messages enabled.", "🚫 Welcome messages disabled."))
	return tg.ErrEndGroup
}

func sendWelcome(m *tg.ParticipantUpdate) {
	chatID := m.ChannelID()
	enabled, err := database.GetWelcomeEnabled(chatID)
	if err != nil || !enabled || m.User == nil || m.User.Bot || m.User.Deleted {
		return
	}
	welcomeMu.Lock()
	if time.Now().Unix()-lastWelcome[chatID] < 2 {
		welcomeMu.Unlock()
		return
	}
	lastWelcome[chatID] = time.Now().Unix()
	welcomeMu.Unlock()
	name := utils.MentionHTML(m.User)
	text := fmt.Sprintf("🌟 <b>Welcome %s!</b>\n\n👥 Welcome to the group.\n🆔 Your ID: <code>%d</code>", name, m.User.ID)
	m.Client.SendMessage(chatID, text, &tg.SendOptions{ParseMode: "HTML", LinkPreview: false})
}

func userInfoHandler(m *tg.NewMessage) error {
	var target *tg.UserObj
	if m.IsReply() {
		r, err := m.GetReplyMessage()
		if err == nil && r != nil {
			target = r.Sender
		}
	}
	if target == nil {
		target = m.Sender
	}
	if target == nil {
		m.Reply("⚠️ User information is unavailable.")
		return tg.ErrEndGroup
	}
	name := strings.TrimSpace(target.FirstName + " " + target.LastName)
	if name == "" {
		name = "Not set"
	}
	username := "Not set"
	if target.Username != "" {
		username = "@" + target.Username
	}
	profile := fmt.Sprintf("https://t.me/%s", target.Username)
	if target.Username == "" {
		profile = fmt.Sprintf("tg://user?id=%d", target.ID)
	}
	text := fmt.Sprintf("<b>👤 User Information</b>\n\n<b>🆔 ID:</b> <code>%d</code>\n<b>👨‍💻 Name:</b> %s\n<b>🏷 Username:</b> %s\n<b>🔗 Mention:</b> %s\n<b>🤖 Bot:</b> %t\n<b>🗑 Deleted:</b> %t\n<b>🔗 Profile:</b> <a href=\"%s\">Open</a>", target.ID, utils.EscapeHTML(name), utils.EscapeHTML(username), utils.MentionHTML(target), target.Bot, target.Deleted, profile)
	m.Reply(text, &tg.SendOptions{ParseMode: "HTML", LinkPreview: false})
	return tg.ErrEndGroup
}

func init() {
	helpTexts["/tagall"] = "<i>Tag non-bot members in batches of five. Admin only.</i>"
	helpTexts["/admintag"] = "<i>Tag chat administrators in batches of five. Admin only.</i>"
	helpTexts["/welcome"] = "<i>Enable or disable automatic welcome messages.</i>"
	helpTexts["/info"] = "<i>Show basic information about yourself or a replied user.</i>"
	helpTexts["/vclogger"] = "<i>Enable or disable voice-chat join/leave logging.</i>"
	helpTexts["/gmtag"] = "<i>Tag members with Good Morning messages.</i>"
	helpTexts["/gatag"] = "<i>Tag members with Good Afternoon messages.</i>"
	helpTexts["/gntag"] = "<i>Tag members with Good Night messages.</i>"
}

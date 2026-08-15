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

var tagEmojis = []string{
	"🌸", "🦋", "✨", "💫", "🌷", "🌺",
	"💖", "🌻", "🍀", "🎀", "🌈", "⭐",
}

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
			members, _, err = m.Client.GetChatMembers(
				chatID,
				&tg.ParticipantOptions{
					Filter:           &tg.ChannelParticipantsAdmins{},
					Limit:            -1,
					SleepThresholdMs: 3000,
				},
			)
		} else {
			members, _, err = m.Client.GetChatMembers(
				chatID,
				&tg.ParticipantOptions{
					Limit:            -1,
					SleepThresholdMs: 3000,
				},
			)
		}

		if err != nil {
			m.Client.SendMessage(
				chatID,
				"❌ Failed to fetch members: "+utils.EscapeHTML(err.Error()),
			)
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

			if member == nil ||
				member.User == nil ||
				member.User.Bot ||
				member.User.Deleted {
				continue
			}

			emoji := tagEmojis[rand.Intn(len(tagEmojis))]

			batch = append(
				batch,
				fmt.Sprintf(
					"%s %s",
					emoji,
					utils.MentionHTML(member.User),
				),
			)

			tagged++

			if len(batch) == 5 {
				text := strings.TrimSpace(intro)

				if text != "" {
					text += "\n\n"
				}

				text += strings.Join(batch, " ")

				if _, err := m.Client.SendMessage(
					chatID,
					text,
					&tg.SendOptions{
						ParseMode:   "HTML",
						LinkPreview: false,
					},
				); err != nil {
					gologging.ErrorF(
						"tagall send failed in %d: %v",
						chatID,
						err,
					)
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

			m.Client.SendMessage(
				chatID,
				text,
				&tg.SendOptions{
					ParseMode:   "HTML",
					LinkPreview: false,
				},
			)
		}

		label := "members"

		if adminsOnly {
			label = "admins"
		}

		m.Client.SendMessage(
			chatID,
			fmt.Sprintf(
				"✅ Tagging completed.\nTotal %s: %d\nTagged: %d",
				label,
				total,
				tagged,
			),
		)
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
		m.Reply(
			"Usage: <code>/tagall your message</code> or reply to a message with /tagall",
		)
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
		m.Reply(
			"Usage: <code>/admintag your message</code> or reply to a message",
		)
		return tg.ErrEndGroup
	}

	return tagMembers(m, true, args)
}

func cancelTagHandler(m *tg.NewMessage) error {
	chatID := m.ChannelID()

	ok, _ := utils.IsChatAdmin(
		m.Client,
		chatID,
		m.SenderID(),
	)

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

// ============================================================
// GOOD MORNING — 30 MESSAGES
// ============================================================

var gmMessages = []string{
	"🌅 <b>Good Morning!</b> ✨\n\nMay your morning begin with peace, positivity and a beautiful smile. 🌸 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nA fresh day brings fresh hopes. Keep smiling and keep shining. ☀️ {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nMay today bring you happiness and wonderful moments. 🌼 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nRise with confidence, walk with purpose, and enjoy every little moment. 🌞 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nStart your day with gratitude and end it with satisfaction. 💛 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nMay your coffee be warm and your day brighter than yesterday. ☕✨ {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nNew morning, new opportunities, new reasons to smile. 🌈 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nWishing you a peaceful heart and a productive day ahead. 🍀 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nLet today be another beautiful page of your story. 📖🌷 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nBelieve in yourself and make today count. 💫 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nMay sunshine fill your day with hope and joy. 🌻 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nWake up, smile, and let positive energy lead the way. 😊✨ {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nEvery sunrise is a reminder that you can begin again. 🌅 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nMay your day be calm, kind, and full of good surprises. 🎀 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nKeep your heart light and your dreams big. 💖 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nSending warm morning vibes and lots of positivity your way. 🦋 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nMay today give you many reasons to be proud of yourself. ⭐ {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nA beautiful morning to a beautiful soul. Stay blessed and happy. 🌺 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nStart small, dream big, and make today meaningful. 🚀 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nMay every step today take you closer to your goals. 🎯 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nLeave yesterday behind and welcome today's possibilities. 🌤️ {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nMay your morning be peaceful and your whole day delightful. 🕊️ {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nSmile—it's a brand-new chance to create something wonderful. 😄🌸 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nKeep spreading kindness wherever you go. 🤍 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nMay your day sparkle with happiness and positive thoughts. ✨ {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nFresh morning, fresh mindset, fresh energy. Let's make it beautiful! 🔥 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nWishing you courage for challenges and joy in every success. 🌷 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nProtect your peace and enjoy your journey. 💫 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nMay this morning bring you closer to everything you wish for. 🌞 {mention}",
	"🌅 <b>Good Morning!</b> ✨\n\nHave a lovely morning and an even lovelier day ahead. 🌸💖 {mention}",
}

// ============================================================
// GOOD AFTERNOON — 30 MESSAGES
// ============================================================

var gaMessages = []string{
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nHope your afternoon is going beautifully. Keep smiling! 🌸 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nTake a little break and enjoy the moment. 🍵 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nMay your afternoon be peaceful, productive and positive. 🌻 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nSending warm afternoon vibes your way. ✨ {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nKeep going—you are doing better than you think. 💪 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nA bright afternoon for a bright soul. 🌼 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nMay the rest of your day be filled with good news. 💫 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nDon't forget to smile and stay hydrated. 💧😊 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nTake a breath, reset your mind, and continue with confidence. 🍀 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nWishing you calm thoughts and happy moments this afternoon. 🦋 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nMay your hard work turn into something beautiful today. 🌷 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nKeep your energy positive and your heart peaceful. 💖 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nEnjoy this little pause and make the rest of today count. ☀️ {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nMay your afternoon bring fresh motivation and new ideas. 💡 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nKeep shining even when the day gets busy. ⭐ {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nOne step at a time, you are moving forward. 🎯 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nWishing you a productive afternoon and a peaceful evening ahead. 🌤️ {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nMay happiness find you in the smallest moments today. 🌸 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nA little positivity can change the whole afternoon. 🌈 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nBelieve that better moments are still ahead. ✨ {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nStay focused, stay kind, and enjoy your journey. 🌺 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nMay your afternoon be lighter than your worries. 🕊️ {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nSending you sunshine, smiles and positive thoughts. 🌞 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nKeep chasing your goals with a happy heart. 🚀 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nMay today surprise you with something wonderful. 🎀 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nRelax your mind, refresh your energy, and keep moving. 🍃 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nWishing you a beautiful afternoon full of peaceful moments. 💛 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nYour efforts today will matter tomorrow. 🌟 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nMay the rest of your day be as wonderful as you deserve. 🌷 {mention}",
	"🌤️ <b>Good Afternoon!</b> ☀️\n\nHave a sweet, peaceful and successful afternoon. ☀️💖 {mention}",
}

// ============================================================
// GOOD NIGHT — 30 MESSAGES
// ============================================================

var gnMessages = []string{
	"🌙 <b>Good Night!</b> 🌌\n\nMay your mind be peaceful and your dreams beautiful. 🌙 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nRest well and let tomorrow bring a fresh beginning. ✨ {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nMay your worries fade away and your heart find peace tonight. 🕊️ {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nYou made it through another day—be proud of yourself. 💖 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nClose your eyes, relax, and welcome a peaceful night. 🌌 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nWishing you sweet dreams and a calm, refreshing sleep. 💤 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nMay tomorrow be brighter, kinder and happier than today. 🌟 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nLeave today's stress behind and rest your mind. 🌙 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nSending peaceful night vibes and warm wishes your way. 🌸 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nMay the stars remind you that beautiful things are always possible. ⭐ {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nRest your heart tonight; tomorrow is another chance to shine. ✨ {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nSleep peacefully and wake up with new energy. 🌅 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nMay your night be quiet, cozy and full of lovely dreams. 🦋 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nTake a deep breath and let the day gently come to an end. 🍃 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nWishing you a peaceful night and a beautiful tomorrow. 💫 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nYour efforts today were enough. Now it's time to rest. 🤍 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nMay sleep bring you comfort and tomorrow bring you happiness. 🌷 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nLet the moonlight carry away every unnecessary worry. 🌙 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nKeep hope in your heart and peace in your soul. 💛 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nMay your dreams be filled with smiles and wonderful moments. 😊✨ {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nRest well—new opportunities are waiting for you tomorrow. 🚀 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nForget the mistakes, remember the lessons, and rest. 🌌 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nMay tonight recharge your spirit and calm your thoughts. 🕊️ {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nSending you a little sparkle before you say good night. ✨ {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nSleep peacefully and wake up ready for a beautiful new day. 🌞 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nTomorrow deserves your best energy, so rest well. 💖 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nMay your pillow be soft, your dreams sweet, and your heart peaceful. 🌙 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nLet today end with gratitude and tomorrow begin with hope. 🌸 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nKeep smiling—better moments are on their way. 🌟 {mention}",
	"🌙 <b>Good Night!</b> 🌌\n\nWishing you a restful night and a wonderful tomorrow. 🌙💫 {mention}",
}

// ============================================================
// WISH TAGGING
// ============================================================

func wishTagHandler(
	m *tg.NewMessage,
	messages []string,
	label string,
) error {
	chatID := m.ChannelID()

	wishMu.Lock()

	if activeWishes[chatID] {
		wishMu.Unlock()

		m.Reply(
			"⚠️ A wish tagging process is already running.",
		)

		return tg.ErrEndGroup
	}

	activeWishes[chatID] = true
	wishMu.Unlock()

	go func() {
		defer func() {
			wishMu.Lock()
			delete(activeWishes, chatID)
			wishMu.Unlock()
		}()

		members, _, err := m.Client.GetChatMembers(
			chatID,
			&tg.ParticipantOptions{
				Limit:            -1,
				SleepThresholdMs: 3000,
			},
		)

		if err != nil {
			m.Client.SendMessage(
				chatID,
				"❌ Failed to fetch members: "+
					utils.EscapeHTML(err.Error()),
			)
			return
		}

		for _, member := range members {
			wishMu.Lock()

			running := activeWishes[chatID]

			wishMu.Unlock()

			if !running {
				m.Client.SendMessage(
					chatID,
					"🛑 Wish tagging stopped.",
				)
				return
			}

			if member == nil ||
				member.User == nil ||
				member.User.Bot ||
				member.User.Deleted {
				continue
			}

			text := strings.ReplaceAll(
				messages[rand.Intn(len(messages))],
				"{mention}",
				utils.MentionHTML(member.User),
			)

			m.Client.SendMessage(
				chatID,
				text,
				&tg.SendOptions{
					ParseMode:   "HTML",
					LinkPreview: false,
				},
			)

			time.Sleep(3 * time.Second)
		}

		m.Client.SendMessage(
			chatID,
			"✅ <b>"+label+" tagging completed.</b>",
			&tg.SendOptions{
				ParseMode: "HTML",
			},
		)
	}()

	return tg.ErrEndGroup
}

func gmTagHandler(m *tg.NewMessage) error {
	return wishTagHandler(
		m,
		gmMessages,
		"Good Morning",
	)
}

func gaTagHandler(m *tg.NewMessage) error {
	return wishTagHandler(
		m,
		gaMessages,
		"Good Afternoon",
	)
}

func gnTagHandler(m *tg.NewMessage) error {
	return wishTagHandler(
		m,
		gnMessages,
		"Good Night",
	)
}

func wishStopHandler(m *tg.NewMessage) error {
	wishMu.Lock()

	running := activeWishes[m.ChannelID()]

	delete(
		activeWishes,
		m.ChannelID(),
	)

	wishMu.Unlock()

	if running {
		m.Reply("🛑 Wish tagging stopped.")
	} else {
		m.Reply("ℹ️ Nothing is running.")
	}

	return tg.ErrEndGroup
}

// ============================================================
// WELCOME
// ============================================================

func welcomeCommandHandler(m *tg.NewMessage) error {
	chatID := m.ChannelID()

	ok, _ := utils.IsChatAdmin(
		m.Client,
		chatID,
		m.SenderID(),
	)

	if !ok {
		m.Reply(
			"⚠️ Only chat administrators can use this command.",
		)
		return tg.ErrEndGroup
	}

	arg := strings.TrimSpace(
		strings.ToLower(m.Args()),
	)

	if arg == "" {
		enabled, _ := database.GetWelcomeEnabled(chatID)

		m.Reply(
			fmt.Sprintf(
				"Welcome messages are currently <b>%s</b>.\nUsage: <code>/welcome on</code> or <code>/welcome off</code>",
				utils.IfElse(
					enabled,
					"ON",
					"OFF",
				),
			),
		)

		return tg.ErrEndGroup
	}

	enabled, err := utils.ParseBool(arg)

	if err != nil {
		m.Reply(
			"⚠️ Use on/off or enable/disable.",
		)
		return tg.ErrEndGroup
	}

	if err := database.SetWelcomeEnabled(
		chatID,
		enabled,
	); err != nil {
		m.Reply(
			"❌ Failed to save welcome setting: "+
				utils.EscapeHTML(err.Error()),
		)
		return tg.ErrEndGroup
	}

	m.Reply(
		utils.IfElse(
			enabled,
			"✅ Welcome messages enabled.",
			"🚫 Welcome messages disabled.",
		),
	)

	return tg.ErrEndGroup
}

func sendWelcome(m *tg.ParticipantUpdate) {
	chatID := m.ChannelID()

	enabled, err := database.GetWelcomeEnabled(chatID)

	if err != nil ||
		!enabled ||
		m.User == nil ||
		m.User.Bot ||
		m.User.Deleted {
		return
	}

	welcomeMu.Lock()

	if time.Now().Unix()-
		lastWelcome[chatID] < 2 {

		welcomeMu.Unlock()
		return
	}

	lastWelcome[chatID] =
		time.Now().Unix()

	welcomeMu.Unlock()

	name := utils.MentionHTML(m.User)

	text := fmt.Sprintf(
		"🌟 <b>Welcome %s!</b>\n\n"+
			"👥 Welcome to the group.\n"+
			"🆔 Your ID: <code>%d</code>",
		name,
		m.User.ID,
	)

	m.Client.SendMessage(
		chatID,
		text,
		&tg.SendOptions{
			ParseMode:   "HTML",
			LinkPreview: false,
		},
	)
}

// ============================================================
// USER INFO
// ============================================================

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
		m.Reply(
			"⚠️ User information is unavailable.",
		)
		return tg.ErrEndGroup
	}

	name := strings.TrimSpace(
		target.FirstName + " " + target.LastName,
	)

	if name == "" {
		name = "Not set"
	}

	username := "Not set"

	if target.Username != "" {
		username = "@" + target.Username
	}

	profile := fmt.Sprintf(
		"https://t.me/%s",
		target.Username,
	)

	if target.Username == "" {
		profile = fmt.Sprintf(
			"tg://user?id=%d",
			target.ID,
		)
	}

	text := fmt.Sprintf(
		"<b>👤 User Information</b>\n\n"+
			"<b>🆔 ID:</b> <code>%d</code>\n"+
			"<b>👨‍💻 Name:</b> %s\n"+
			"<b>🏷 Username:</b> %s\n"+
			"<b>🔗 Mention:</b> %s\n"+
			"<b>🤖 Bot:</b> %t\n"+
			"<b>🗑 Deleted:</b> %t\n"+
			"<b>🔗 Profile:</b> <a href=\"%s\">Open</a>",
		target.ID,
		utils.EscapeHTML(name),
		utils.EscapeHTML(username),
		utils.MentionHTML(target),
		target.Bot,
		target.Deleted,
		profile,
	)

	m.Reply(
		text,
		&tg.SendOptions{
			ParseMode:   "HTML",
			LinkPreview: false,
		},
	)

	return tg.ErrEndGroup
}

// ============================================================
// HELP
// ============================================================

func init() {
	helpTexts["/tagall"] =
		"<i>Tag non-bot members in batches of five. Admin only.</i>"

	helpTexts["/admintag"] =
		"<i>Tag chat administrators in batches of five. Admin only.</i>"

	helpTexts["/welcome"] =
		"<i>Enable or disable automatic welcome messages.</i>"

	helpTexts["/info"] =
		"<i>Show basic information about yourself or a replied user.</i>"

	helpTexts["/vclogger"] =
		"<i>Enable or disable voice-chat join/leave logging.</i>"

	helpTexts["/gmtag"] =
		"<i>Tag members with Good Morning messages.</i>"

	helpTexts["/gatag"] =
		"<i>Tag members with Good Afternoon messages.</i>"

	helpTexts["/gntag"] =
		"<i>Tag members with Good Night messages.</i>"
}

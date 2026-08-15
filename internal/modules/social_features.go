
const welcomeMessage = `╭━━━━━━━━━━━━━━━━━━━━━━╮
        🌷 𝑾𝒆𝒍𝒄𝒐𝒎𝒆 🌷
╰━━━━━━━━━━━━━━━━━━━━━━╯

✨ 𝑯𝒆𝒚 %s 💗

𝑾𝒆𝒍𝒄𝒐𝒎𝒆 𝒕𝒐 𝒐𝒖𝒓 𝒃𝒆𝒂𝒖𝒕𝒊𝒇𝒖𝒍 𝒄𝒐𝒓𝒏𝒆𝒓 🌸

🌙 𝑴𝒂𝒚 𝒚𝒐𝒖𝒓 𝒅𝒂𝒚𝒔 𝒃𝒆 𝒑𝒆𝒂𝒄𝒆𝒇𝒖𝒍
🌷 𝒀𝒐𝒖𝒓 𝒉𝒆𝒂𝒓𝒕 𝒃𝒆 𝒉𝒂𝒑𝒑𝒚
✨ 𝑨𝒏𝒅 𝒚𝒐𝒖𝒓 𝒔𝒎𝒊𝒍𝒆 𝒏𝒆𝒗𝒆𝒓 𝒇𝒂𝒅𝒆

💫 𝑺𝒕𝒂𝒚 𝒄𝒖𝒕𝒆 • 𝑺𝒕𝒂𝒚 𝒉𝒂𝒑𝒑𝒚 • 𝑺𝒕𝒂𝒚 𝒃𝒍𝒆𝒔𝒔𝒆𝒅 💫

          💗 𝑬𝒏𝒋𝒐𝒚 𝒚𝒐𝒖𝒓 𝒔𝒕𝒂𝒚 💗`

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

var gmMessages = []string{
	`🌅 <b>ＧＯＯＤ ＭＯＲＮＩＮＧ</b> 🌸

May your morning begin with peace and end with a reason to smile.
🌸 Wishing you a little more happiness today.

🌷 Keep smiling and keep shining!

{mention}`,
	`☀️ <b>Good Morning, Beautiful Soul!</b> 🌷

Fill your heart with positive thoughts and your day with beautiful memories.
✨ May your heart stay light and hopeful.

🌸 Let your smile brighten someone's day!

{mention}`,
	`🌞 <b>Rise & Shine!</b> ✨

Keep your dreams close, your smile bright and your heart peaceful.
💫 Keep believing in the good that is coming.

💫 Believe in yourself today!

{mention}`,
	`🌻 <b>A Fresh Morning Is Here!</b> 💛

May your efforts turn into achievements and your worries into peace.
🌈 Let positive thoughts guide your next step.

☕ Enjoy your morning with positive vibes!

{mention}`,
	`☕ <b>Morning Vibes Are Calling!</b> 🌼

May every little moment today bring you happiness.
🦋 May your day be filled with peaceful moments.

🌻 Go make some beautiful memories!

{mention}`,
	`🦋 <b>Wake Up With A Smile!</b> 🌈

A new morning means another chance to do something wonderful.
🌸 Wishing you a little more happiness today.

🦋 Spread good vibes wherever you go!

{mention}`,
	`🌤️ <b>New Day, New Energy!</b> 💫

Take a deep breath, smile and let the day begin on a positive note.
✨ May your heart stay light and hopeful.

✨ Make today count!

{mention}`,
	`🌺 <b>Have A Wonderful Morning!</b> ☀️

Leave yesterday behind and welcome today with fresh hope.
💫 Keep believing in the good that is coming.

⭐ Your best moments may be waiting for you!

{mention}`,
	`🍃 <b>Peaceful Morning Wishes!</b> 🌸

May the first light of today bring clarity, courage and calmness.
🌈 Let positive thoughts guide your next step.

❤️ Stay happy, stay blessed!

{mention}`,
	`💖 <b>Start Today With A Happy Heart!</b> 🌅

May today be kinder, brighter and more beautiful than you expected.
🦋 May your day be filled with peaceful moments.

🌈 Have a beautiful day ahead!

{mention}`,
	`🌅 <b>ＧＯＯＤ ＭＯＲＮＩＮＧ</b> 🌸

Leave yesterday behind and welcome today with fresh hope.
🌸 Wishing you a little more happiness today.

✨ Make today count!

{mention}`,
	`☀️ <b>Good Morning, Beautiful Soul!</b> 🌷

May the first light of today bring clarity, courage and calmness.
✨ May your heart stay light and hopeful.

⭐ Your best moments may be waiting for you!

{mention}`,
	`🌞 <b>Rise & Shine!</b> ✨

May today be kinder, brighter and more beautiful than you expected.
💫 Keep believing in the good that is coming.

❤️ Stay happy, stay blessed!

{mention}`,
	`🌻 <b>A Fresh Morning Is Here!</b> 💛

May your morning begin with peace and end with a reason to smile.
🌈 Let positive thoughts guide your next step.

🌈 Have a beautiful day ahead!

{mention}`,
	`☕ <b>Morning Vibes Are Calling!</b> 🌼

Fill your heart with positive thoughts and your day with beautiful memories.
🦋 May your day be filled with peaceful moments.

🌷 Keep smiling and keep shining!

{mention}`,
	`🦋 <b>Wake Up With A Smile!</b> 🌈

Keep your dreams close, your smile bright and your heart peaceful.
🌸 Wishing you a little more happiness today.

🌸 Let your smile brighten someone's day!

{mention}`,
	`🌤️ <b>New Day, New Energy!</b> 💫

May your efforts turn into achievements and your worries into peace.
✨ May your heart stay light and hopeful.

💫 Believe in yourself today!

{mention}`,
	`🌺 <b>Have A Wonderful Morning!</b> ☀️

May every little moment today bring you happiness.
💫 Keep believing in the good that is coming.

☕ Enjoy your morning with positive vibes!

{mention}`,
	`🍃 <b>Peaceful Morning Wishes!</b> 🌸

A new morning means another chance to do something wonderful.
🌈 Let positive thoughts guide your next step.

🌻 Go make some beautiful memories!

{mention}`,
	`💖 <b>Start Today With A Happy Heart!</b> 🌅

Take a deep breath, smile and let the day begin on a positive note.
🦋 May your day be filled with peaceful moments.

🦋 Spread good vibes wherever you go!

{mention}`,
	`🌅 <b>ＧＯＯＤ ＭＯＲＮＩＮＧ</b> 🌸

May every little moment today bring you happiness.
🌸 Wishing you a little more happiness today.

💫 Believe in yourself today!

{mention}`,
	`☀️ <b>Good Morning, Beautiful Soul!</b> 🌷

A new morning means another chance to do something wonderful.
✨ May your heart stay light and hopeful.

☕ Enjoy your morning with positive vibes!

{mention}`,
	`🌞 <b>Rise & Shine!</b> ✨

Take a deep breath, smile and let the day begin on a positive note.
💫 Keep believing in the good that is coming.

🌻 Go make some beautiful memories!

{mention}`,
	`🌻 <b>A Fresh Morning Is Here!</b> 💛

Leave yesterday behind and welcome today with fresh hope.
🌈 Let positive thoughts guide your next step.

🦋 Spread good vibes wherever you go!

{mention}`,
	`☕ <b>Morning Vibes Are Calling!</b> 🌼

May the first light of today bring clarity, courage and calmness.
🦋 May your day be filled with peaceful moments.

✨ Make today count!

{mention}`,
	`🦋 <b>Wake Up With A Smile!</b> 🌈

May today be kinder, brighter and more beautiful than you expected.
🌸 Wishing you a little more happiness today.

⭐ Your best moments may be waiting for you!

{mention}`,
	`🌤️ <b>New Day, New Energy!</b> 💫

May your morning begin with peace and end with a reason to smile.
✨ May your heart stay light and hopeful.

❤️ Stay happy, stay blessed!

{mention}`,
	`🌺 <b>Have A Wonderful Morning!</b> ☀️

Fill your heart with positive thoughts and your day with beautiful memories.
💫 Keep believing in the good that is coming.

🌈 Have a beautiful day ahead!

{mention}`,
	`🍃 <b>Peaceful Morning Wishes!</b> 🌸

Keep your dreams close, your smile bright and your heart peaceful.
🌈 Let positive thoughts guide your next step.

🌷 Keep smiling and keep shining!

{mention}`,
	`💖 <b>Start Today With A Happy Heart!</b> 🌅

May your efforts turn into achievements and your worries into peace.
🦋 May your day be filled with peaceful moments.

🌸 Let your smile brighten someone's day!

{mention}`,
	`🌅 <b>ＧＯＯＤ ＭＯＲＮＩＮＧ</b> 🌸

Fill your heart with positive thoughts and your day with beautiful memories.
🌸 Wishing you a little more happiness today.

❤️ Stay happy, stay blessed!

{mention}`,
	`☀️ <b>Good Morning, Beautiful Soul!</b> 🌷

Keep your dreams close, your smile bright and your heart peaceful.
✨ May your heart stay light and hopeful.

🌈 Have a beautiful day ahead!

{mention}`,
	`🌞 <b>Rise & Shine!</b> ✨

May your efforts turn into achievements and your worries into peace.
💫 Keep believing in the good that is coming.

🌷 Keep smiling and keep shining!

{mention}`,
	`🌻 <b>A Fresh Morning Is Here!</b> 💛

May every little moment today bring you happiness.
🌈 Let positive thoughts guide your next step.

🌸 Let your smile brighten someone's day!

{mention}`,
	`☕ <b>Morning Vibes Are Calling!</b> 🌼

A new morning means another chance to do something wonderful.
🦋 May your day be filled with peaceful moments.

💫 Believe in yourself today!

{mention}`,
	`🦋 <b>Wake Up With A Smile!</b> 🌈

Take a deep breath, smile and let the day begin on a positive note.
🌸 Wishing you a little more happiness today.

☕ Enjoy your morning with positive vibes!

{mention}`,
	`🌤️ <b>New Day, New Energy!</b> 💫

Leave yesterday behind and welcome today with fresh hope.
✨ May your heart stay light and hopeful.

🌻 Go make some beautiful memories!

{mention}`,
	`🌺 <b>Have A Wonderful Morning!</b> ☀️

May the first light of today bring clarity, courage and calmness.
💫 Keep believing in the good that is coming.

🦋 Spread good vibes wherever you go!

{mention}`,
	`🍃 <b>Peaceful Morning Wishes!</b> 🌸

May today be kinder, brighter and more beautiful than you expected.
🌈 Let positive thoughts guide your next step.

✨ Make today count!

{mention}`,
	`💖 <b>Start Today With A Happy Heart!</b> 🌅

May your morning begin with peace and end with a reason to smile.
🦋 May your day be filled with peaceful moments.

⭐ Your best moments may be waiting for you!

{mention}`,
	`🌅 <b>ＧＯＯＤ ＭＯＲＮＩＮＧ</b> 🌸

May the first light of today bring clarity, courage and calmness.
🌸 Wishing you a little more happiness today.

🌻 Go make some beautiful memories!

{mention}`,
	`☀️ <b>Good Morning, Beautiful Soul!</b> 🌷

May today be kinder, brighter and more beautiful than you expected.
✨ May your heart stay light and hopeful.

🦋 Spread good vibes wherever you go!

{mention}`,
	`🌞 <b>Rise & Shine!</b> ✨

May your morning begin with peace and end with a reason to smile.
💫 Keep believing in the good that is coming.

✨ Make today count!

{mention}`,
	`🌻 <b>A Fresh Morning Is Here!</b> 💛

Fill your heart with positive thoughts and your day with beautiful memories.
🌈 Let positive thoughts guide your next step.

⭐ Your best moments may be waiting for you!

{mention}`,
	`☕ <b>Morning Vibes Are Calling!</b> 🌼

Keep your dreams close, your smile bright and your heart peaceful.
🦋 May your day be filled with peaceful moments.

❤️ Stay happy, stay blessed!

{mention}`,
	`🦋 <b>Wake Up With A Smile!</b> 🌈

May your efforts turn into achievements and your worries into peace.
🌸 Wishing you a little more happiness today.

🌈 Have a beautiful day ahead!

{mention}`,
	`🌤️ <b>New Day, New Energy!</b> 💫

May every little moment today bring you happiness.
✨ May your heart stay light and hopeful.

🌷 Keep smiling and keep shining!

{mention}`,
	`🌺 <b>Have A Wonderful Morning!</b> ☀️

A new morning means another chance to do something wonderful.
💫 Keep believing in the good that is coming.

🌸 Let your smile brighten someone's day!

{mention}`,
	`🍃 <b>Peaceful Morning Wishes!</b> 🌸

Take a deep breath, smile and let the day begin on a positive note.
🌈 Let positive thoughts guide your next step.

💫 Believe in yourself today!

{mention}`,
	`💖 <b>Start Today With A Happy Heart!</b> 🌅

Leave yesterday behind and welcome today with fresh hope.
🦋 May your day be filled with peaceful moments.

☕ Enjoy your morning with positive vibes!

{mention}`,
	`🌅 <b>ＧＯＯＤ ＭＯＲＮＩＮＧ</b> 🌸

A new morning means another chance to do something wonderful.
🌸 Wishing you a little more happiness today.

🌷 Keep smiling and keep shining!

{mention}`,
	`☀️ <b>Good Morning, Beautiful Soul!</b> 🌷

Take a deep breath, smile and let the day begin on a positive note.
✨ May your heart stay light and hopeful.

🌸 Let your smile brighten someone's day!

{mention}`,
	`🌞 <b>Rise & Shine!</b> ✨

Leave yesterday behind and welcome today with fresh hope.
💫 Keep believing in the good that is coming.

💫 Believe in yourself today!

{mention}`,
	`🌻 <b>A Fresh Morning Is Here!</b> 💛

May the first light of today bring clarity, courage and calmness.
🌈 Let positive thoughts guide your next step.

☕ Enjoy your morning with positive vibes!

{mention}`,
	`☕ <b>Morning Vibes Are Calling!</b> 🌼

May today be kinder, brighter and more beautiful than you expected.
🦋 May your day be filled with peaceful moments.

🌻 Go make some beautiful memories!

{mention}`,
	`🦋 <b>Wake Up With A Smile!</b> 🌈

May your morning begin with peace and end with a reason to smile.
🌸 Wishing you a little more happiness today.

🦋 Spread good vibes wherever you go!

{mention}`,
	`🌤️ <b>New Day, New Energy!</b> 💫

Fill your heart with positive thoughts and your day with beautiful memories.
✨ May your heart stay light and hopeful.

✨ Make today count!

{mention}`,
	`🌺 <b>Have A Wonderful Morning!</b> ☀️

Keep your dreams close, your smile bright and your heart peaceful.
💫 Keep believing in the good that is coming.

⭐ Your best moments may be waiting for you!

{mention}`,
	`🍃 <b>Peaceful Morning Wishes!</b> 🌸

May your efforts turn into achievements and your worries into peace.
🌈 Let positive thoughts guide your next step.

❤️ Stay happy, stay blessed!

{mention}`,
	`💖 <b>Start Today With A Happy Heart!</b> 🌅

May every little moment today bring you happiness.
🦋 May your day be filled with peaceful moments.

🌈 Have a beautiful day ahead!

{mention}`,
	`🌅 <b>ＧＯＯＤ ＭＯＲＮＩＮＧ</b> 🌸

Keep your dreams close, your smile bright and your heart peaceful.
🌸 Wishing you a little more happiness today.

✨ Make today count!

{mention}`,
	`☀️ <b>Good Morning, Beautiful Soul!</b> 🌷

May your efforts turn into achievements and your worries into peace.
✨ May your heart stay light and hopeful.

⭐ Your best moments may be waiting for you!

{mention}`,
	`🌞 <b>Rise & Shine!</b> ✨

May every little moment today bring you happiness.
💫 Keep believing in the good that is coming.

❤️ Stay happy, stay blessed!

{mention}`,
	`🌻 <b>A Fresh Morning Is Here!</b> 💛

A new morning means another chance to do something wonderful.
🌈 Let positive thoughts guide your next step.

🌈 Have a beautiful day ahead!

{mention}`,
	`☕ <b>Morning Vibes Are Calling!</b> 🌼

Take a deep breath, smile and let the day begin on a positive note.
🦋 May your day be filled with peaceful moments.

🌷 Keep smiling and keep shining!

{mention}`,
	`🦋 <b>Wake Up With A Smile!</b> 🌈

Leave yesterday behind and welcome today with fresh hope.
🌸 Wishing you a little more happiness today.

🌸 Let your smile brighten someone's day!

{mention}`,
	`🌤️ <b>New Day, New Energy!</b> 💫

May the first light of today bring clarity, courage and calmness.
✨ May your heart stay light and hopeful.

💫 Believe in yourself today!

{mention}`,
	`🌺 <b>Have A Wonderful Morning!</b> ☀️

May today be kinder, brighter and more beautiful than you expected.
💫 Keep believing in the good that is coming.

☕ Enjoy your morning with positive vibes!

{mention}`,
	`🍃 <b>Peaceful Morning Wishes!</b> 🌸

May your morning begin with peace and end with a reason to smile.
🌈 Let positive thoughts guide your next step.

🌻 Go make some beautiful memories!

{mention}`,
	`💖 <b>Start Today With A Happy Heart!</b> 🌅

Fill your heart with positive thoughts and your day with beautiful memories.
🦋 May your day be filled with peaceful moments.

🦋 Spread good vibes wherever you go!

{mention}`,
	`🌅 <b>ＧＯＯＤ ＭＯＲＮＩＮＧ</b> 🌸

May today be kinder, brighter and more beautiful than you expected.
🌸 Wishing you a little more happiness today.

💫 Believe in yourself today!

{mention}`,
	`☀️ <b>Good Morning, Beautiful Soul!</b> 🌷

May your morning begin with peace and end with a reason to smile.
✨ May your heart stay light and hopeful.

☕ Enjoy your morning with positive vibes!

{mention}`,
	`🌞 <b>Rise & Shine!</b> ✨

Fill your heart with positive thoughts and your day with beautiful memories.
💫 Keep believing in the good that is coming.

🌻 Go make some beautiful memories!

{mention}`,
	`🌻 <b>A Fresh Morning Is Here!</b> 💛

Keep your dreams close, your smile bright and your heart peaceful.
🌈 Let positive thoughts guide your next step.

🦋 Spread good vibes wherever you go!

{mention}`,
	`☕ <b>Morning Vibes Are Calling!</b> 🌼

May your efforts turn into achievements and your worries into peace.
🦋 May your day be filled with peaceful moments.

✨ Make today count!

{mention}`,
	`🦋 <b>Wake Up With A Smile!</b> 🌈

May every little moment today bring you happiness.
🌸 Wishing you a little more happiness today.

⭐ Your best moments may be waiting for you!

{mention}`,
	`🌤️ <b>New Day, New Energy!</b> 💫

A new morning means another chance to do something wonderful.
✨ May your heart stay light and hopeful.

❤️ Stay happy, stay blessed!

{mention}`,
	`🌺 <b>Have A Wonderful Morning!</b> ☀️

Take a deep breath, smile and let the day begin on a positive note.
💫 Keep believing in the good that is coming.

🌈 Have a beautiful day ahead!

{mention}`,
	`🍃 <b>Peaceful Morning Wishes!</b> 🌸

Leave yesterday behind and welcome today with fresh hope.
🌈 Let positive thoughts guide your next step.

🌷 Keep smiling and keep shining!

{mention}`,
	`💖 <b>Start Today With A Happy Heart!</b> 🌅

May the first light of today bring clarity, courage and calmness.
🦋 May your day be filled with peaceful moments.

🌸 Let your smile brighten someone's day!

{mention}`,
	`🌅 <b>ＧＯＯＤ ＭＯＲＮＩＮＧ</b> 🌸

Take a deep breath, smile and let the day begin on a positive note.
🌸 Wishing you a little more happiness today.

❤️ Stay happy, stay blessed!

{mention}`,
	`☀️ <b>Good Morning, Beautiful Soul!</b> 🌷

Leave yesterday behind and welcome today with fresh hope.
✨ May your heart stay light and hopeful.

🌈 Have a beautiful day ahead!

{mention}`,
	`🌞 <b>Rise & Shine!</b> ✨

May the first light of today bring clarity, courage and calmness.
💫 Keep believing in the good that is coming.

🌷 Keep smiling and keep shining!

{mention}`,
	`🌻 <b>A Fresh Morning Is Here!</b> 💛

May today be kinder, brighter and more beautiful than you expected.
🌈 Let positive thoughts guide your next step.

🌸 Let your smile brighten someone's day!

{mention}`,
	`☕ <b>Morning Vibes Are Calling!</b> 🌼

May your morning begin with peace and end with a reason to smile.
🦋 May your day be filled with peaceful moments.

💫 Believe in yourself today!

{mention}`,
	`🦋 <b>Wake Up With A Smile!</b> 🌈

Fill your heart with positive thoughts and your day with beautiful memories.
🌸 Wishing you a little more happiness today.

☕ Enjoy your morning with positive vibes!

{mention}`,
	`🌤️ <b>New Day, New Energy!</b> 💫

Keep your dreams close, your smile bright and your heart peaceful.
✨ May your heart stay light and hopeful.

🌻 Go make some beautiful memories!

{mention}`,
	`🌺 <b>Have A Wonderful Morning!</b> ☀️

May your efforts turn into achievements and your worries into peace.
💫 Keep believing in the good that is coming.

🦋 Spread good vibes wherever you go!

{mention}`,
	`🍃 <b>Peaceful Morning Wishes!</b> 🌸

May every little moment today bring you happiness.
🌈 Let positive thoughts guide your next step.

✨ Make today count!

{mention}`,
	`💖 <b>Start Today With A Happy Heart!</b> 🌅

A new morning means another chance to do something wonderful.
🦋 May your day be filled with peaceful moments.

⭐ Your best moments may be waiting for you!

{mention}`,
	`🌅 <b>ＧＯＯＤ ＭＯＲＮＩＮＧ</b> 🌸

May your efforts turn into achievements and your worries into peace.
🌸 Wishing you a little more happiness today.

🌻 Go make some beautiful memories!

{mention}`,
	`☀️ <b>Good Morning, Beautiful Soul!</b> 🌷

May every little moment today bring you happiness.
✨ May your heart stay light and hopeful.

🦋 Spread good vibes wherever you go!

{mention}`,
	`🌞 <b>Rise & Shine!</b> ✨

A new morning means another chance to do something wonderful.
💫 Keep believing in the good that is coming.

✨ Make today count!

{mention}`,
	`🌻 <b>A Fresh Morning Is Here!</b> 💛

Take a deep breath, smile and let the day begin on a positive note.
🌈 Let positive thoughts guide your next step.

⭐ Your best moments may be waiting for you!

{mention}`,
	`☕ <b>Morning Vibes Are Calling!</b> 🌼

Leave yesterday behind and welcome today with fresh hope.
🦋 May your day be filled with peaceful moments.

❤️ Stay happy, stay blessed!

{mention}`,
	`🦋 <b>Wake Up With A Smile!</b> 🌈

May the first light of today bring clarity, courage and calmness.
🌸 Wishing you a little more happiness today.

🌈 Have a beautiful day ahead!

{mention}`,
	`🌤️ <b>New Day, New Energy!</b> 💫

May today be kinder, brighter and more beautiful than you expected.
✨ May your heart stay light and hopeful.

🌷 Keep smiling and keep shining!

{mention}`,
	`🌺 <b>Have A Wonderful Morning!</b> ☀️

May your morning begin with peace and end with a reason to smile.
💫 Keep believing in the good that is coming.

🌸 Let your smile brighten someone's day!

{mention}`,
	`🍃 <b>Peaceful Morning Wishes!</b> 🌸

Fill your heart with positive thoughts and your day with beautiful memories.
🌈 Let positive thoughts guide your next step.

💫 Believe in yourself today!

{mention}`,
	`💖 <b>Start Today With A Happy Heart!</b> 🌅

Keep your dreams close, your smile bright and your heart peaceful.
🦋 May your day be filled with peaceful moments.

☕ Enjoy your morning with positive vibes!

{mention}`,
}

var gaMessages = []string{
	`☀️ <b>ＧＯＯＤ ＡＦＴＥＲＮＯＯＮ</b> 🌻

Take a small break, refresh your mind and continue your day with confidence.
🌸 Wishing you a little more happiness today.

🌻 Keep smiling through the rest of your day!

{mention}`,
	`🌤️ <b>Good Afternoon, Sunshine!</b> ✨

Don't let a busy day steal your smile; keep a little happiness for yourself.
✨ May your heart stay light and hopeful.

🦋 Keep your heart light and your mind calm!

{mention}`,
	`🍵 <b>Afternoon Vibes!</b> 🌸

May your hard work today bring you closer to the dreams you are chasing.
💫 Keep believing in the good that is coming.

💫 Stay strong and keep shining!

{mention}`,
	`🌞 <b>Have A Beautiful Afternoon!</b> 💛

May this afternoon bring fresh energy, good thoughts and pleasant surprises.
🌈 Let positive thoughts guide your next step.

☕ Take care and enjoy your afternoon!

{mention}`,
	`🌼 <b>Hope Your Afternoon Is Going Great!</b> ☀️

May your afternoon be calm, productive and filled with good moments.
🦋 May your day be filled with peaceful moments.

🌞 Make the rest of today beautiful!

{mention}`,
	`🦋 <b>Pause, Breathe & Smile!</b> 🌈

Keep moving forward, one positive step at a time.
🌸 Wishing you a little more happiness today.

🌸 Sending peaceful vibes your way!

{mention}`,
	`🌿 <b>A Little Afternoon Positivity!</b> 💫

The day is still yours—make the remaining hours meaningful.
✨ May your heart stay light and hopeful.

✨ Keep the positive energy going!

{mention}`,
	`🌺 <b>Sending Warm Afternoon Wishes!</b> ☕

Half the day may be gone, but there is still plenty of time for something amazing.
💫 Keep believing in the good that is coming.

⭐ Something good may be just around the corner!

{mention}`,
	`💖 <b>Make This Afternoon Special!</b> 🌤️

A peaceful mind can make even an ordinary afternoon feel special.
🌈 Let positive thoughts guide your next step.

❤️ Have a lovely rest of the day!

{mention}`,
	`✨ <b>Keep Going, You're Doing Great!</b> 🌻

Let go of unnecessary stress and enjoy the little things around you.
🦋 May your day be filled with peaceful moments.

🌈 Wishing you many happy moments today!

{mention}`,
	`☀️ <b>ＧＯＯＤ ＡＦＴＥＲＮＯＯＮ</b> 🌻

Half the day may be gone, but there is still plenty of time for something amazing.
🌸 Wishing you a little more happiness today.

✨ Keep the positive energy going!

{mention}`,
	`🌤️ <b>Good Afternoon, Sunshine!</b> ✨

A peaceful mind can make even an ordinary afternoon feel special.
✨ May your heart stay light and hopeful.

⭐ Something good may be just around the corner!

{mention}`,
	`🍵 <b>Afternoon Vibes!</b> 🌸

Let go of unnecessary stress and enjoy the little things around you.
💫 Keep believing in the good that is coming.

❤️ Have a lovely rest of the day!

{mention}`,
	`🌞 <b>Have A Beautiful Afternoon!</b> 💛

Take a small break, refresh your mind and continue your day with confidence.
🌈 Let positive thoughts guide your next step.

🌈 Wishing you many happy moments today!

{mention}`,
	`🌼 <b>Hope Your Afternoon Is Going Great!</b> ☀️

Don't let a busy day steal your smile; keep a little happiness for yourself.
🦋 May your day be filled with peaceful moments.

🌻 Keep smiling through the rest of your day!

{mention}`,
	`🦋 <b>Pause, Breathe & Smile!</b> 🌈

May your hard work today bring you closer to the dreams you are chasing.
🌸 Wishing you a little more happiness today.

🦋 Keep your heart light and your mind calm!

{mention}`,
	`🌿 <b>A Little Afternoon Positivity!</b> 💫

May this afternoon bring fresh energy, good thoughts and pleasant surprises.
✨ May your heart stay light and hopeful.

💫 Stay strong and keep shining!

{mention}`,
	`🌺 <b>Sending Warm Afternoon Wishes!</b> ☕

May your afternoon be calm, productive and filled with good moments.
💫 Keep believing in the good that is coming.

☕ Take care and enjoy your afternoon!

{mention}`,
	`💖 <b>Make This Afternoon Special!</b> 🌤️

Keep moving forward, one positive step at a time.
🌈 Let positive thoughts guide your next step.

🌞 Make the rest of today beautiful!

{mention}`,
	`✨ <b>Keep Going, You're Doing Great!</b> 🌻

The day is still yours—make the remaining hours meaningful.
🦋 May your day be filled with peaceful moments.

🌸 Sending peaceful vibes your way!

{mention}`,
	`☀️ <b>ＧＯＯＤ ＡＦＴＥＲＮＯＯＮ</b> 🌻

May your afternoon be calm, productive and filled with good moments.
🌸 Wishing you a little more happiness today.

💫 Stay strong and keep shining!

{mention}`,
	`🌤️ <b>Good Afternoon, Sunshine!</b> ✨

Keep moving forward, one positive step at a time.
✨ May your heart stay light and hopeful.

☕ Take care and enjoy your afternoon!

{mention}`,
	`🍵 <b>Afternoon Vibes!</b> 🌸

The day is still yours—make the remaining hours meaningful.
💫 Keep believing in the good that is coming.

🌞 Make the rest of today beautiful!

{mention}`,
	`🌞 <b>Have A Beautiful Afternoon!</b> 💛

Half the day may be gone, but there is still plenty of time for something amazing.
🌈 Let positive thoughts guide your next step.

🌸 Sending peaceful vibes your way!

{mention}`,
	`🌼 <b>Hope Your Afternoon Is Going Great!</b> ☀️

A peaceful mind can make even an ordinary afternoon feel special.
🦋 May your day be filled with peaceful moments.

✨ Keep the positive energy going!

{mention}`,
	`🦋 <b>Pause, Breathe & Smile!</b> 🌈

Let go of unnecessary stress and enjoy the little things around you.
🌸 Wishing you a little more happiness today.

⭐ Something good may be just around the corner!

{mention}`,
	`🌿 <b>A Little Afternoon Positivity!</b> 💫

Take a small break, refresh your mind and continue your day with confidence.
✨ May your heart stay light and hopeful.

❤️ Have a lovely rest of the day!

{mention}`,
	`🌺 <b>Sending Warm Afternoon Wishes!</b> ☕

Don't let a busy day steal your smile; keep a little happiness for yourself.
💫 Keep believing in the good that is coming.

🌈 Wishing you many happy moments today!

{mention}`,
	`💖 <b>Make This Afternoon Special!</b> 🌤️

May your hard work today bring you closer to the dreams you are chasing.
🌈 Let positive thoughts guide your next step.

🌻 Keep smiling through the rest of your day!

{mention}`,
	`✨ <b>Keep Going, You're Doing Great!</b> 🌻

May this afternoon bring fresh energy, good thoughts and pleasant surprises.
🦋 May your day be filled with peaceful moments.

🦋 Keep your heart light and your mind calm!

{mention}`,
	`☀️ <b>ＧＯＯＤ ＡＦＴＥＲＮＯＯＮ</b> 🌻

Don't let a busy day steal your smile; keep a little happiness for yourself.
🌸 Wishing you a little more happiness today.

❤️ Have a lovely rest of the day!

{mention}`,
	`🌤️ <b>Good Afternoon, Sunshine!</b> ✨

May your hard work today bring you closer to the dreams you are chasing.
✨ May your heart stay light and hopeful.

🌈 Wishing you many happy moments today!

{mention}`,
	`🍵 <b>Afternoon Vibes!</b> 🌸

May this afternoon bring fresh energy, good thoughts and pleasant surprises.
💫 Keep believing in the good that is coming.

🌻 Keep smiling through the rest of your day!

{mention}`,
	`🌞 <b>Have A Beautiful Afternoon!</b> 💛

May your afternoon be calm, productive and filled with good moments.
🌈 Let positive thoughts guide your next step.

🦋 Keep your heart light and your mind calm!

{mention}`,
	`🌼 <b>Hope Your Afternoon Is Going Great!</b> ☀️

Keep moving forward, one positive step at a time.
🦋 May your day be filled with peaceful moments.

💫 Stay strong and keep shining!

{mention}`,
	`🦋 <b>Pause, Breathe & Smile!</b> 🌈

The day is still yours—make the remaining hours meaningful.
🌸 Wishing you a little more happiness today.

☕ Take care and enjoy your afternoon!

{mention}`,
	`🌿 <b>A Little Afternoon Positivity!</b> 💫

Half the day may be gone, but there is still plenty of time for something amazing.
✨ May your heart stay light and hopeful.

🌞 Make the rest of today beautiful!

{mention}`,
	`🌺 <b>Sending Warm Afternoon Wishes!</b> ☕

A peaceful mind can make even an ordinary afternoon feel special.
💫 Keep believing in the good that is coming.

🌸 Sending peaceful vibes your way!

{mention}`,
	`💖 <b>Make This Afternoon Special!</b> 🌤️

Let go of unnecessary stress and enjoy the little things around you.
🌈 Let positive thoughts guide your next step.

✨ Keep the positive energy going!

{mention}`,
	`✨ <b>Keep Going, You're Doing Great!</b> 🌻

Take a small break, refresh your mind and continue your day with confidence.
🦋 May your day be filled with peaceful moments.

⭐ Something good may be just around the corner!

{mention}`,
	`☀️ <b>ＧＯＯＤ ＡＦＴＥＲＮＯＯＮ</b> 🌻

A peaceful mind can make even an ordinary afternoon feel special.
🌸 Wishing you a little more happiness today.

🌞 Make the rest of today beautiful!

{mention}`,
	`🌤️ <b>Good Afternoon, Sunshine!</b> ✨

Let go of unnecessary stress and enjoy the little things around you.
✨ May your heart stay light and hopeful.

🌸 Sending peaceful vibes your way!

{mention}`,
	`🍵 <b>Afternoon Vibes!</b> 🌸

Take a small break, refresh your mind and continue your day with confidence.
💫 Keep believing in the good that is coming.

✨ Keep the positive energy going!

{mention}`,
	`🌞 <b>Have A Beautiful Afternoon!</b> 💛

Don't let a busy day steal your smile; keep a little happiness for yourself.
🌈 Let positive thoughts guide your next step.

⭐ Something good may be just around the corner!

{mention}`,
	`🌼 <b>Hope Your Afternoon Is Going Great!</b> ☀️

May your hard work today bring you closer to the dreams you are chasing.
🦋 May your day be filled with peaceful moments.

❤️ Have a lovely rest of the day!

{mention}`,
	`🦋 <b>Pause, Breathe & Smile!</b> 🌈

May this afternoon bring fresh energy, good thoughts and pleasant surprises.
🌸 Wishing you a little more happiness today.

🌈 Wishing you many happy moments today!

{mention}`,
	`🌿 <b>A Little Afternoon Positivity!</b> 💫

May your afternoon be calm, productive and filled with good moments.
✨ May your heart stay light and hopeful.

🌻 Keep smiling through the rest of your day!

{mention}`,
	`🌺 <b>Sending Warm Afternoon Wishes!</b> ☕

Keep moving forward, one positive step at a time.
💫 Keep believing in the good that is coming.

🦋 Keep your heart light and your mind calm!

{mention}`,
	`💖 <b>Make This Afternoon Special!</b> 🌤️

The day is still yours—make the remaining hours meaningful.
🌈 Let positive thoughts guide your next step.

💫 Stay strong and keep shining!

{mention}`,
	`✨ <b>Keep Going, You're Doing Great!</b> 🌻

Half the day may be gone, but there is still plenty of time for something amazing.
🦋 May your day be filled with peaceful moments.

☕ Take care and enjoy your afternoon!

{mention}`,
	`☀️ <b>ＧＯＯＤ ＡＦＴＥＲＮＯＯＮ</b> 🌻

Keep moving forward, one positive step at a time.
🌸 Wishing you a little more happiness today.

🌻 Keep smiling through the rest of your day!

{mention}`,
	`🌤️ <b>Good Afternoon, Sunshine!</b> ✨

The day is still yours—make the remaining hours meaningful.
✨ May your heart stay light and hopeful.

🦋 Keep your heart light and your mind calm!

{mention}`,
	`🍵 <b>Afternoon Vibes!</b> 🌸

Half the day may be gone, but there is still plenty of time for something amazing.
💫 Keep believing in the good that is coming.

💫 Stay strong and keep shining!

{mention}`,
	`🌞 <b>Have A Beautiful Afternoon!</b> 💛

A peaceful mind can make even an ordinary afternoon feel special.
🌈 Let positive thoughts guide your next step.

☕ Take care and enjoy your afternoon!

{mention}`,
	`🌼 <b>Hope Your Afternoon Is Going Great!</b> ☀️

Let go of unnecessary stress and enjoy the little things around you.
🦋 May your day be filled with peaceful moments.

🌞 Make the rest of today beautiful!

{mention}`,
	`🦋 <b>Pause, Breathe & Smile!</b> 🌈

Take a small break, refresh your mind and continue your day with confidence.
🌸 Wishing you a little more happiness today.

🌸 Sending peaceful vibes your way!

{mention}`,
	`🌿 <b>A Little Afternoon Positivity!</b> 💫

Don't let a busy day steal your smile; keep a little happiness for yourself.
✨ May your heart stay light and hopeful.

✨ Keep the positive energy going!

{mention}`,
	`🌺 <b>Sending Warm Afternoon Wishes!</b> ☕

May your hard work today bring you closer to the dreams you are chasing.
💫 Keep believing in the good that is coming.

⭐ Something good may be just around the corner!

{mention}`,
	`💖 <b>Make This Afternoon Special!</b> 🌤️

May this afternoon bring fresh energy, good thoughts and pleasant surprises.
🌈 Let positive thoughts guide your next step.

❤️ Have a lovely rest of the day!

{mention}`,
	`✨ <b>Keep Going, You're Doing Great!</b> 🌻

May your afternoon be calm, productive and filled with good moments.
🦋 May your day be filled with peaceful moments.

🌈 Wishing you many happy moments today!

{mention}`,
	`☀️ <b>ＧＯＯＤ ＡＦＴＥＲＮＯＯＮ</b> 🌻

May your hard work today bring you closer to the dreams you are chasing.
🌸 Wishing you a little more happiness today.

✨ Keep the positive energy going!

{mention}`,
	`🌤️ <b>Good Afternoon, Sunshine!</b> ✨

May this afternoon bring fresh energy, good thoughts and pleasant surprises.
✨ May your heart stay light and hopeful.

⭐ Something good may be just around the corner!

{mention}`,
	`🍵 <b>Afternoon Vibes!</b> 🌸

May your afternoon be calm, productive and filled with good moments.
💫 Keep believing in the good that is coming.

❤️ Have a lovely rest of the day!

{mention}`,
	`🌞 <b>Have A Beautiful Afternoon!</b> 💛

Keep moving forward, one positive step at a time.
🌈 Let positive thoughts guide your next step.

🌈 Wishing you many happy moments today!

{mention}`,
	`🌼 <b>Hope Your Afternoon Is Going Great!</b> ☀️

The day is still yours—make the remaining hours meaningful.
🦋 May your day be filled with peaceful moments.

🌻 Keep smiling through the rest of your day!

{mention}`,
	`🦋 <b>Pause, Breathe & Smile!</b> 🌈

Half the day may be gone, but there is still plenty of time for something amazing.
🌸 Wishing you a little more happiness today.

🦋 Keep your heart light and your mind calm!

{mention}`,
	`🌿 <b>A Little Afternoon Positivity!</b> 💫

A peaceful mind can make even an ordinary afternoon feel special.
✨ May your heart stay light and hopeful.

💫 Stay strong and keep shining!

{mention}`,
	`🌺 <b>Sending Warm Afternoon Wishes!</b> ☕

Let go of unnecessary stress and enjoy the little things around you.
💫 Keep believing in the good that is coming.

☕ Take care and enjoy your afternoon!

{mention}`,
	`💖 <b>Make This Afternoon Special!</b> 🌤️

Take a small break, refresh your mind and continue your day with confidence.
🌈 Let positive thoughts guide your next step.

🌞 Make the rest of today beautiful!

{mention}`,
	`✨ <b>Keep Going, You're Doing Great!</b> 🌻

Don't let a busy day steal your smile; keep a little happiness for yourself.
🦋 May your day be filled with peaceful moments.

🌸 Sending peaceful vibes your way!

{mention}`,
	`☀️ <b>ＧＯＯＤ ＡＦＴＥＲＮＯＯＮ</b> 🌻

Let go of unnecessary stress and enjoy the little things around you.
🌸 Wishing you a little more happiness today.

💫 Stay strong and keep shining!

{mention}`,
	`🌤️ <b>Good Afternoon, Sunshine!</b> ✨

Take a small break, refresh your mind and continue your day with confidence.
✨ May your heart stay light and hopeful.

☕ Take care and enjoy your afternoon!

{mention}`,
	`🍵 <b>Afternoon Vibes!</b> 🌸

Don't let a busy day steal your smile; keep a little happiness for yourself.
💫 Keep believing in the good that is coming.

🌞 Make the rest of today beautiful!

{mention}`,
	`🌞 <b>Have A Beautiful Afternoon!</b> 💛

May your hard work today bring you closer to the dreams you are chasing.
🌈 Let positive thoughts guide your next step.

🌸 Sending peaceful vibes your way!

{mention}`,
	`🌼 <b>Hope Your Afternoon Is Going Great!</b> ☀️

May this afternoon bring fresh energy, good thoughts and pleasant surprises.
🦋 May your day be filled with peaceful moments.

✨ Keep the positive energy going!

{mention}`,
	`🦋 <b>Pause, Breathe & Smile!</b> 🌈

May your afternoon be calm, productive and filled with good moments.
🌸 Wishing you a little more happiness today.

⭐ Something good may be just around the corner!

{mention}`,
	`🌿 <b>A Little Afternoon Positivity!</b> 💫

Keep moving forward, one positive step at a time.
✨ May your heart stay light and hopeful.

❤️ Have a lovely rest of the day!

{mention}`,
	`🌺 <b>Sending Warm Afternoon Wishes!</b> ☕

The day is still yours—make the remaining hours meaningful.
💫 Keep believing in the good that is coming.

🌈 Wishing you many happy moments today!

{mention}`,
	`💖 <b>Make This Afternoon Special!</b> 🌤️

Half the day may be gone, but there is still plenty of time for something amazing.
🌈 Let positive thoughts guide your next step.

🌻 Keep smiling through the rest of your day!

{mention}`,
	`✨ <b>Keep Going, You're Doing Great!</b> 🌻

A peaceful mind can make even an ordinary afternoon feel special.
🦋 May your day be filled with peaceful moments.

🦋 Keep your heart light and your mind calm!

{mention}`,
	`☀️ <b>ＧＯＯＤ ＡＦＴＥＲＮＯＯＮ</b> 🌻

The day is still yours—make the remaining hours meaningful.
🌸 Wishing you a little more happiness today.

❤️ Have a lovely rest of the day!

{mention}`,
	`🌤️ <b>Good Afternoon, Sunshine!</b> ✨

Half the day may be gone, but there is still plenty of time for something amazing.
✨ May your heart stay light and hopeful.

🌈 Wishing you many happy moments today!

{mention}`,
	`🍵 <b>Afternoon Vibes!</b> 🌸

A peaceful mind can make even an ordinary afternoon feel special.
💫 Keep believing in the good that is coming.

🌻 Keep smiling through the rest of your day!

{mention}`,
	`🌞 <b>Have A Beautiful Afternoon!</b> 💛

Let go of unnecessary stress and enjoy the little things around you.
🌈 Let positive thoughts guide your next step.

🦋 Keep your heart light and your mind calm!

{mention}`,
	`🌼 <b>Hope Your Afternoon Is Going Great!</b> ☀️

Take a small break, refresh your mind and continue your day with confidence.
🦋 May your day be filled with peaceful moments.

💫 Stay strong and keep shining!

{mention}`,
	`🦋 <b>Pause, Breathe & Smile!</b> 🌈

Don't let a busy day steal your smile; keep a little happiness for yourself.
🌸 Wishing you a little more happiness today.

☕ Take care and enjoy your afternoon!

{mention}`,
	`🌿 <b>A Little Afternoon Positivity!</b> 💫

May your hard work today bring you closer to the dreams you are chasing.
✨ May your heart stay light and hopeful.

🌞 Make the rest of today beautiful!

{mention}`,
	`🌺 <b>Sending Warm Afternoon Wishes!</b> ☕

May this afternoon bring fresh energy, good thoughts and pleasant surprises.
💫 Keep believing in the good that is coming.

🌸 Sending peaceful vibes your way!

{mention}`,
	`💖 <b>Make This Afternoon Special!</b> 🌤️

May your afternoon be calm, productive and filled with good moments.
🌈 Let positive thoughts guide your next step.

✨ Keep the positive energy going!

{mention}`,
	`✨ <b>Keep Going, You're Doing Great!</b> 🌻

Keep moving forward, one positive step at a time.
🦋 May your day be filled with peaceful moments.

⭐ Something good may be just around the corner!

{mention}`,
	`☀️ <b>ＧＯＯＤ ＡＦＴＥＲＮＯＯＮ</b> 🌻

May this afternoon bring fresh energy, good thoughts and pleasant surprises.
🌸 Wishing you a little more happiness today.

🌞 Make the rest of today beautiful!

{mention}`,
	`🌤️ <b>Good Afternoon, Sunshine!</b> ✨

May your afternoon be calm, productive and filled with good moments.
✨ May your heart stay light and hopeful.

🌸 Sending peaceful vibes your way!

{mention}`,
	`🍵 <b>Afternoon Vibes!</b> 🌸

Keep moving forward, one positive step at a time.
💫 Keep believing in the good that is coming.

✨ Keep the positive energy going!

{mention}`,
	`🌞 <b>Have A Beautiful Afternoon!</b> 💛

The day is still yours—make the remaining hours meaningful.
🌈 Let positive thoughts guide your next step.

⭐ Something good may be just around the corner!

{mention}`,
	`🌼 <b>Hope Your Afternoon Is Going Great!</b> ☀️

Half the day may be gone, but there is still plenty of time for something amazing.
🦋 May your day be filled with peaceful moments.

❤️ Have a lovely rest of the day!

{mention}`,
	`🦋 <b>Pause, Breathe & Smile!</b> 🌈

A peaceful mind can make even an ordinary afternoon feel special.
🌸 Wishing you a little more happiness today.

🌈 Wishing you many happy moments today!

{mention}`,
	`🌿 <b>A Little Afternoon Positivity!</b> 💫

Let go of unnecessary stress and enjoy the little things around you.
✨ May your heart stay light and hopeful.

🌻 Keep smiling through the rest of your day!

{mention}`,
	`🌺 <b>Sending Warm Afternoon Wishes!</b> ☕

Take a small break, refresh your mind and continue your day with confidence.
💫 Keep believing in the good that is coming.

🦋 Keep your heart light and your mind calm!

{mention}`,
	`💖 <b>Make This Afternoon Special!</b> 🌤️

Don't let a busy day steal your smile; keep a little happiness for yourself.
🌈 Let positive thoughts guide your next step.

💫 Stay strong and keep shining!

{mention}`,
	`✨ <b>Keep Going, You're Doing Great!</b> 🌻

May your hard work today bring you closer to the dreams you are chasing.
🦋 May your day be filled with peaceful moments.

☕ Take care and enjoy your afternoon!

{mention}`,
}

var gnMessages = []string{
	`🌙 <b>ＧＯＯＤ ＮＩＧＨＴ</b> ✨

Leave today's worries behind and give your mind the peaceful rest it deserves.
🌸 Wishing you a little more happiness today.

🌙 Sleep peacefully and dream beautifully!

{mention}`,
	`🌌 <b>Good Night, Beautiful Soul!</b> 🌷

May your heart feel lighter and your dreams feel brighter tonight.
✨ May your heart stay light and hopeful.

🦋 Let your heart rest tonight!

{mention}`,
	`💤 <b>Sweet Dreams!</b> 🌙

Forget the little problems and remember the beautiful moments of today.
💫 Keep believing in the good that is coming.

🌷 Rest well and wake up stronger!

{mention}`,
	`🌠 <b>Time To Rest & Recharge!</b> 💖

Tomorrow is a new page—rest peacefully and get ready to write something beautiful.
🌈 Let positive thoughts guide your next step.

💤 Sweet dreams and peaceful sleep!

{mention}`,
	`🌃 <b>Peaceful Night Wishes!</b> ✨

Close your eyes, breathe slowly and let the stress of the day fade away.
🦋 May your day be filled with peaceful moments.

🌠 See you in a brighter tomorrow!

{mention}`,
	`🦋 <b>Let The Day Rest Now!</b> 🌙

May the quiet of the night bring peace to your thoughts.
🌸 Wishing you a little more happiness today.

💖 Sending peaceful night vibes!

{mention}`,
	`⭐ <b>Dream Big Tonight!</b> 🌌

May every worry become smaller and every good hope become stronger.
✨ May your heart stay light and hopeful.

✨ May tomorrow bring you more reasons to smile!

{mention}`,
	`🌙 <b>A Calm Night For A Calm Heart!</b> 🌸

Whatever happened today, tomorrow gives you another chance to begin again.
💫 Keep believing in the good that is coming.

❤️ Good night, take care!

{mention}`,
	`💫 <b>Good Night & Stay Blessed!</b> 💤

You did enough for today; now it's time to relax and recharge.
🌈 Let positive thoughts guide your next step.

⭐ Dream big and wake up shining!

{mention}`,
	`❤️ <b>End The Day With A Smile!</b> 🌙

Rest well tonight so you can wake up with fresh energy tomorrow.
🦋 May your day be filled with peaceful moments.

🌌 Good night, stay happy and blessed!

{mention}`,
	`🌙 <b>ＧＯＯＤ ＮＩＧＨＴ</b> ✨

Whatever happened today, tomorrow gives you another chance to begin again.
🌸 Wishing you a little more happiness today.

✨ May tomorrow bring you more reasons to smile!

{mention}`,
	`🌌 <b>Good Night, Beautiful Soul!</b> 🌷

You did enough for today; now it's time to relax and recharge.
✨ May your heart stay light and hopeful.

❤️ Good night, take care!

{mention}`,
	`💤 <b>Sweet Dreams!</b> 🌙

Rest well tonight so you can wake up with fresh energy tomorrow.
💫 Keep believing in the good that is coming.

⭐ Dream big and wake up shining!

{mention}`,
	`🌠 <b>Time To Rest & Recharge!</b> 💖

Leave today's worries behind and give your mind the peaceful rest it deserves.
🌈 Let positive thoughts guide your next step.

🌌 Good night, stay happy and blessed!

{mention}`,
	`🌃 <b>Peaceful Night Wishes!</b> ✨

May your heart feel lighter and your dreams feel brighter tonight.
🦋 May your day be filled with peaceful moments.

🌙 Sleep peacefully and dream beautifully!

{mention}`,
	`🦋 <b>Let The Day Rest Now!</b> 🌙

Forget the little problems and remember the beautiful moments of today.
🌸 Wishing you a little more happiness today.

🦋 Let your heart rest tonight!

{mention}`,
	`⭐ <b>Dream Big Tonight!</b> 🌌

Tomorrow is a new page—rest peacefully and get ready to write something beautiful.
✨ May your heart stay light and hopeful.

🌷 Rest well and wake up stronger!

{mention}`,
	`🌙 <b>A Calm Night For A Calm Heart!</b> 🌸

Close your eyes, breathe slowly and let the stress of the day fade away.
💫 Keep believing in the good that is coming.

💤 Sweet dreams and peaceful sleep!

{mention}`,
	`💫 <b>Good Night & Stay Blessed!</b> 💤

May the quiet of the night bring peace to your thoughts.
🌈 Let positive thoughts guide your next step.

🌠 See you in a brighter tomorrow!

{mention}`,
	`❤️ <b>End The Day With A Smile!</b> 🌙

May every worry become smaller and every good hope become stronger.
🦋 May your day be filled with peaceful moments.

💖 Sending peaceful night vibes!

{mention}`,
	`🌙 <b>ＧＯＯＤ ＮＩＧＨＴ</b> ✨

Close your eyes, breathe slowly and let the stress of the day fade away.
🌸 Wishing you a little more happiness today.

🌷 Rest well and wake up stronger!

{mention}`,
	`🌌 <b>Good Night, Beautiful Soul!</b> 🌷

May the quiet of the night bring peace to your thoughts.
✨ May your heart stay light and hopeful.

💤 Sweet dreams and peaceful sleep!

{mention}`,
	`💤 <b>Sweet Dreams!</b> 🌙

May every worry become smaller and every good hope become stronger.
💫 Keep believing in the good that is coming.

🌠 See you in a brighter tomorrow!

{mention}`,
	`🌠 <b>Time To Rest & Recharge!</b> 💖

Whatever happened today, tomorrow gives you another chance to begin again.
🌈 Let positive thoughts guide your next step.

💖 Sending peaceful night vibes!

{mention}`,
	`🌃 <b>Peaceful Night Wishes!</b> ✨

You did enough for today; now it's time to relax and recharge.
🦋 May your day be filled with peaceful moments.

✨ May tomorrow bring you more reasons to smile!

{mention}`,
	`🦋 <b>Let The Day Rest Now!</b> 🌙

Rest well tonight so you can wake up with fresh energy tomorrow.
🌸 Wishing you a little more happiness today.

❤️ Good night, take care!

{mention}`,
	`⭐ <b>Dream Big Tonight!</b> 🌌

Leave today's worries behind and give your mind the peaceful rest it deserves.
✨ May your heart stay light and hopeful.

⭐ Dream big and wake up shining!

{mention}`,
	`🌙 <b>A Calm Night For A Calm Heart!</b> 🌸

May your heart feel lighter and your dreams feel brighter tonight.
💫 Keep believing in the good that is coming.

🌌 Good night, stay happy and blessed!

{mention}`,
	`💫 <b>Good Night & Stay Blessed!</b> 💤

Forget the little problems and remember the beautiful moments of today.
🌈 Let positive thoughts guide your next step.

🌙 Sleep peacefully and dream beautifully!

{mention}`,
	`❤️ <b>End The Day With A Smile!</b> 🌙

Tomorrow is a new page—rest peacefully and get ready to write something beautiful.
🦋 May your day be filled with peaceful moments.

🦋 Let your heart rest tonight!

{mention}`,
	`🌙 <b>ＧＯＯＤ ＮＩＧＨＴ</b> ✨

May your heart feel lighter and your dreams feel brighter tonight.
🌸 Wishing you a little more happiness today.

⭐ Dream big and wake up shining!

{mention}`,
	`🌌 <b>Good Night, Beautiful Soul!</b> 🌷

Forget the little problems and remember the beautiful moments of today.
✨ May your heart stay light and hopeful.

🌌 Good night, stay happy and blessed!

{mention}`,
	`💤 <b>Sweet Dreams!</b> 🌙

Tomorrow is a new page—rest peacefully and get ready to write something beautiful.
💫 Keep believing in the good that is coming.

🌙 Sleep peacefully and dream beautifully!

{mention}`,
	`🌠 <b>Time To Rest & Recharge!</b> 💖

Close your eyes, breathe slowly and let the stress of the day fade away.
🌈 Let positive thoughts guide your next step.

🦋 Let your heart rest tonight!

{mention}`,
	`🌃 <b>Peaceful Night Wishes!</b> ✨

May the quiet of the night bring peace to your thoughts.
🦋 May your day be filled with peaceful moments.

🌷 Rest well and wake up stronger!

{mention}`,
	`🦋 <b>Let The Day Rest Now!</b> 🌙

May every worry become smaller and every good hope become stronger.
🌸 Wishing you a little more happiness today.

💤 Sweet dreams and peaceful sleep!

{mention}`,
	`⭐ <b>Dream Big Tonight!</b> 🌌

Whatever happened today, tomorrow gives you another chance to begin again.
✨ May your heart stay light and hopeful.

🌠 See you in a brighter tomorrow!

{mention}`,
	`🌙 <b>A Calm Night For A Calm Heart!</b> 🌸

You did enough for today; now it's time to relax and recharge.
💫 Keep believing in the good that is coming.

💖 Sending peaceful night vibes!

{mention}`,
	`💫 <b>Good Night & Stay Blessed!</b> 💤

Rest well tonight so you can wake up with fresh energy tomorrow.
🌈 Let positive thoughts guide your next step.

✨ May tomorrow bring you more reasons to smile!

{mention}`,
	`❤️ <b>End The Day With A Smile!</b> 🌙

Leave today's worries behind and give your mind the peaceful rest it deserves.
🦋 May your day be filled with peaceful moments.

❤️ Good night, take care!

{mention}`,
	`🌙 <b>ＧＯＯＤ ＮＩＧＨＴ</b> ✨

You did enough for today; now it's time to relax and recharge.
🌸 Wishing you a little more happiness today.

🌠 See you in a brighter tomorrow!

{mention}`,
	`🌌 <b>Good Night, Beautiful Soul!</b> 🌷

Rest well tonight so you can wake up with fresh energy tomorrow.
✨ May your heart stay light and hopeful.

💖 Sending peaceful night vibes!

{mention}`,
	`💤 <b>Sweet Dreams!</b> 🌙

Leave today's worries behind and give your mind the peaceful rest it deserves.
💫 Keep believing in the good that is coming.

✨ May tomorrow bring you more reasons to smile!

{mention}`,
	`🌠 <b>Time To Rest & Recharge!</b> 💖

May your heart feel lighter and your dreams feel brighter tonight.
🌈 Let positive thoughts guide your next step.

❤️ Good night, take care!

{mention}`,
	`🌃 <b>Peaceful Night Wishes!</b> ✨

Forget the little problems and remember the beautiful moments of today.
🦋 May your day be filled with peaceful moments.

⭐ Dream big and wake up shining!

{mention}`,
	`🦋 <b>Let The Day Rest Now!</b> 🌙

Tomorrow is a new page—rest peacefully and get ready to write something beautiful.
🌸 Wishing you a little more happiness today.

🌌 Good night, stay happy and blessed!

{mention}`,
	`⭐ <b>Dream Big Tonight!</b> 🌌

Close your eyes, breathe slowly and let the stress of the day fade away.
✨ May your heart stay light and hopeful.

🌙 Sleep peacefully and dream beautifully!

{mention}`,
	`🌙 <b>A Calm Night For A Calm Heart!</b> 🌸

May the quiet of the night bring peace to your thoughts.
💫 Keep believing in the good that is coming.

🦋 Let your heart rest tonight!

{mention}`,
	`💫 <b>Good Night & Stay Blessed!</b> 💤

May every worry become smaller and every good hope become stronger.
🌈 Let positive thoughts guide your next step.

🌷 Rest well and wake up stronger!

{mention}`,
	`❤️ <b>End The Day With A Smile!</b> 🌙

Whatever happened today, tomorrow gives you another chance to begin again.
🦋 May your day be filled with peaceful moments.

💤 Sweet dreams and peaceful sleep!

{mention}`,
	`🌙 <b>ＧＯＯＤ ＮＩＧＨＴ</b> ✨

May the quiet of the night bring peace to your thoughts.
🌸 Wishing you a little more happiness today.

🌙 Sleep peacefully and dream beautifully!

{mention}`,
	`🌌 <b>Good Night, Beautiful Soul!</b> 🌷

May every worry become smaller and every good hope become stronger.
✨ May your heart stay light and hopeful.

🦋 Let your heart rest tonight!

{mention}`,
	`💤 <b>Sweet Dreams!</b> 🌙

Whatever happened today, tomorrow gives you another chance to begin again.
💫 Keep believing in the good that is coming.

🌷 Rest well and wake up stronger!

{mention}`,
	`🌠 <b>Time To Rest & Recharge!</b> 💖

You did enough for today; now it's time to relax and recharge.
🌈 Let positive thoughts guide your next step.

💤 Sweet dreams and peaceful sleep!

{mention}`,
	`🌃 <b>Peaceful Night Wishes!</b> ✨

Rest well tonight so you can wake up with fresh energy tomorrow.
🦋 May your day be filled with peaceful moments.

🌠 See you in a brighter tomorrow!

{mention}`,
	`🦋 <b>Let The Day Rest Now!</b> 🌙

Leave today's worries behind and give your mind the peaceful rest it deserves.
🌸 Wishing you a little more happiness today.

💖 Sending peaceful night vibes!

{mention}`,
	`⭐ <b>Dream Big Tonight!</b> 🌌

May your heart feel lighter and your dreams feel brighter tonight.
✨ May your heart stay light and hopeful.

✨ May tomorrow bring you more reasons to smile!

{mention}`,
	`🌙 <b>A Calm Night For A Calm Heart!</b> 🌸

Forget the little problems and remember the beautiful moments of today.
💫 Keep believing in the good that is coming.

❤️ Good night, take care!

{mention}`,
	`💫 <b>Good Night & Stay Blessed!</b> 💤

Tomorrow is a new page—rest peacefully and get ready to write something beautiful.
🌈 Let positive thoughts guide your next step.

⭐ Dream big and wake up shining!

{mention}`,
	`❤️ <b>End The Day With A Smile!</b> 🌙

Close your eyes, breathe slowly and let the stress of the day fade away.
🦋 May your day be filled with peaceful moments.

🌌 Good night, stay happy and blessed!

{mention}`,
	`🌙 <b>ＧＯＯＤ ＮＩＧＨＴ</b> ✨

Forget the little problems and remember the beautiful moments of today.
🌸 Wishing you a little more happiness today.

✨ May tomorrow bring you more reasons to smile!

{mention}`,
	`🌌 <b>Good Night, Beautiful Soul!</b> 🌷

Tomorrow is a new page—rest peacefully and get ready to write something beautiful.
✨ May your heart stay light and hopeful.

❤️ Good night, take care!

{mention}`,
	`💤 <b>Sweet Dreams!</b> 🌙

Close your eyes, breathe slowly and let the stress of the day fade away.
💫 Keep believing in the good that is coming.

⭐ Dream big and wake up shining!

{mention}`,
	`🌠 <b>Time To Rest & Recharge!</b> 💖

May the quiet of the night bring peace to your thoughts.
🌈 Let positive thoughts guide your next step.

🌌 Good night, stay happy and blessed!

{mention}`,
	`🌃 <b>Peaceful Night Wishes!</b> ✨

May every worry become smaller and every good hope become stronger.
🦋 May your day be filled with peaceful moments.

🌙 Sleep peacefully and dream beautifully!

{mention}`,
	`🦋 <b>Let The Day Rest Now!</b> 🌙

Whatever happened today, tomorrow gives you another chance to begin again.
🌸 Wishing you a little more happiness today.

🦋 Let your heart rest tonight!

{mention}`,
	`⭐ <b>Dream Big Tonight!</b> 🌌

You did enough for today; now it's time to relax and recharge.
✨ May your heart stay light and hopeful.

🌷 Rest well and wake up stronger!

{mention}`,
	`🌙 <b>A Calm Night For A Calm Heart!</b> 🌸

Rest well tonight so you can wake up with fresh energy tomorrow.
💫 Keep believing in the good that is coming.

💤 Sweet dreams and peaceful sleep!

{mention}`,
	`💫 <b>Good Night & Stay Blessed!</b> 💤

Leave today's worries behind and give your mind the peaceful rest it deserves.
🌈 Let positive thoughts guide your next step.

🌠 See you in a brighter tomorrow!

{mention}`,
	`❤️ <b>End The Day With A Smile!</b> 🌙

May your heart feel lighter and your dreams feel brighter tonight.
🦋 May your day be filled with peaceful moments.

💖 Sending peaceful night vibes!

{mention}`,
	`🌙 <b>ＧＯＯＤ ＮＩＧＨＴ</b> ✨

Rest well tonight so you can wake up with fresh energy tomorrow.
🌸 Wishing you a little more happiness today.

🌷 Rest well and wake up stronger!

{mention}`,
	`🌌 <b>Good Night, Beautiful Soul!</b> 🌷

Leave today's worries behind and give your mind the peaceful rest it deserves.
✨ May your heart stay light and hopeful.

💤 Sweet dreams and peaceful sleep!

{mention}`,
	`💤 <b>Sweet Dreams!</b> 🌙

May your heart feel lighter and your dreams feel brighter tonight.
💫 Keep believing in the good that is coming.

🌠 See you in a brighter tomorrow!

{mention}`,
	`🌠 <b>Time To Rest & Recharge!</b> 💖

Forget the little problems and remember the beautiful moments of today.
🌈 Let positive thoughts guide your next step.

💖 Sending peaceful night vibes!

{mention}`,
	`🌃 <b>Peaceful Night Wishes!</b> ✨

Tomorrow is a new page—rest peacefully and get ready to write something beautiful.
🦋 May your day be filled with peaceful moments.

✨ May tomorrow bring you more reasons to smile!

{mention}`,
	`🦋 <b>Let The Day Rest Now!</b> 🌙

Close your eyes, breathe slowly and let the stress of the day fade away.
🌸 Wishing you a little more happiness today.

❤️ Good night, take care!

{mention}`,
	`⭐ <b>Dream Big Tonight!</b> 🌌

May the quiet of the night bring peace to your thoughts.
✨ May your heart stay light and hopeful.

⭐ Dream big and wake up shining!

{mention}`,
	`🌙 <b>A Calm Night For A Calm Heart!</b> 🌸

May every worry become smaller and every good hope become stronger.
💫 Keep believing in the good that is coming.

🌌 Good night, stay happy and blessed!

{mention}`,
	`💫 <b>Good Night & Stay Blessed!</b> 💤

Whatever happened today, tomorrow gives you another chance to begin again.
🌈 Let positive thoughts guide your next step.

🌙 Sleep peacefully and dream beautifully!

{mention}`,
	`❤️ <b>End The Day With A Smile!</b> 🌙

You did enough for today; now it's time to relax and recharge.
🦋 May your day be filled with peaceful moments.

🦋 Let your heart rest tonight!

{mention}`,
	`🌙 <b>ＧＯＯＤ ＮＩＧＨＴ</b> ✨

May every worry become smaller and every good hope become stronger.
🌸 Wishing you a little more happiness today.

⭐ Dream big and wake up shining!

{mention}`,
	`🌌 <b>Good Night, Beautiful Soul!</b> 🌷

Whatever happened today, tomorrow gives you another chance to begin again.
✨ May your heart stay light and hopeful.

🌌 Good night, stay happy and blessed!

{mention}`,
	`💤 <b>Sweet Dreams!</b> 🌙

You did enough for today; now it's time to relax and recharge.
💫 Keep believing in the good that is coming.

🌙 Sleep peacefully and dream beautifully!

{mention}`,
	`🌠 <b>Time To Rest & Recharge!</b> 💖

Rest well tonight so you can wake up with fresh energy tomorrow.
🌈 Let positive thoughts guide your next step.

🦋 Let your heart rest tonight!

{mention}`,
	`🌃 <b>Peaceful Night Wishes!</b> ✨

Leave today's worries behind and give your mind the peaceful rest it deserves.
🦋 May your day be filled with peaceful moments.

🌷 Rest well and wake up stronger!

{mention}`,
	`🦋 <b>Let The Day Rest Now!</b> 🌙

May your heart feel lighter and your dreams feel brighter tonight.
🌸 Wishing you a little more happiness today.

💤 Sweet dreams and peaceful sleep!

{mention}`,
	`⭐ <b>Dream Big Tonight!</b> 🌌

Forget the little problems and remember the beautiful moments of today.
✨ May your heart stay light and hopeful.

🌠 See you in a brighter tomorrow!

{mention}`,
	`🌙 <b>A Calm Night For A Calm Heart!</b> 🌸

Tomorrow is a new page—rest peacefully and get ready to write something beautiful.
💫 Keep believing in the good that is coming.

💖 Sending peaceful night vibes!

{mention}`,
	`💫 <b>Good Night & Stay Blessed!</b> 💤

Close your eyes, breathe slowly and let the stress of the day fade away.
🌈 Let positive thoughts guide your next step.

✨ May tomorrow bring you more reasons to smile!

{mention}`,
	`❤️ <b>End The Day With A Smile!</b> 🌙

May the quiet of the night bring peace to your thoughts.
🦋 May your day be filled with peaceful moments.

❤️ Good night, take care!

{mention}`,
	`🌙 <b>ＧＯＯＤ ＮＩＧＨＴ</b> ✨

Tomorrow is a new page—rest peacefully and get ready to write something beautiful.
🌸 Wishing you a little more happiness today.

🌠 See you in a brighter tomorrow!

{mention}`,
	`🌌 <b>Good Night, Beautiful Soul!</b> 🌷

Close your eyes, breathe slowly and let the stress of the day fade away.
✨ May your heart stay light and hopeful.

💖 Sending peaceful night vibes!

{mention}`,
	`💤 <b>Sweet Dreams!</b> 🌙

May the quiet of the night bring peace to your thoughts.
💫 Keep believing in the good that is coming.

✨ May tomorrow bring you more reasons to smile!

{mention}`,
	`🌠 <b>Time To Rest & Recharge!</b> 💖

May every worry become smaller and every good hope become stronger.
🌈 Let positive thoughts guide your next step.

❤️ Good night, take care!

{mention}`,
	`🌃 <b>Peaceful Night Wishes!</b> ✨

Whatever happened today, tomorrow gives you another chance to begin again.
🦋 May your day be filled with peaceful moments.

⭐ Dream big and wake up shining!

{mention}`,
	`🦋 <b>Let The Day Rest Now!</b> 🌙

You did enough for today; now it's time to relax and recharge.
🌸 Wishing you a little more happiness today.

🌌 Good night, stay happy and blessed!

{mention}`,
	`⭐ <b>Dream Big Tonight!</b> 🌌

Rest well tonight so you can wake up with fresh energy tomorrow.
✨ May your heart stay light and hopeful.

🌙 Sleep peacefully and dream beautifully!

{mention}`,
	`🌙 <b>A Calm Night For A Calm Heart!</b> 🌸

Leave today's worries behind and give your mind the peaceful rest it deserves.
💫 Keep believing in the good that is coming.

🦋 Let your heart rest tonight!

{mention}`,
	`💫 <b>Good Night & Stay Blessed!</b> 💤

May your heart feel lighter and your dreams feel brighter tonight.
🌈 Let positive thoughts guide your next step.

🌷 Rest well and wake up stronger!

{mention}`,
	`❤️ <b>End The Day With A Smile!</b> 🌙

Forget the little problems and remember the beautiful moments of today.
🦋 May your day be filled with peaceful moments.

💤 Sweet dreams and peaceful sleep!

{mention}`,
}

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

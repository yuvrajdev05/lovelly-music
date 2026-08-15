package modules

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/utils"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))
var funMu sync.Mutex
var funCoins = map[int64]int{}
var funGifts = map[int64][]string{}

var safeTruths = []string{
	"What is a skill you would like to learn this year?",
	"What is your favorite song right now?",
	"What is one habit you are proud of?",
	"What is the funniest harmless mistake you have made?",
	"Which subject do you enjoy most?",
	"What is one place you would like to visit someday?",
}
var safeDares = []string{
	"Send a message using only emojis.",
	"Write a two-line funny poem about your group.",
	"Say your favorite movie title using three emojis.",
	"Type your next message with alternating uppercase and lowercase letters.",
	"Give the group a genuine compliment.",
	"Describe your day in exactly five words.",
}
var boredActivities = []string{
	"Learn one new keyboard shortcut.",
	"Organize your downloads folder for five minutes.",
	"Write down three small goals for tomorrow.",
	"Try a short coding problem you have never solved before.",
	"Take a screen break and stretch for a few minutes.",
}
var cleanRoasts = []string{
	"Your Wi-Fi has more direction than your plan 😄",
	"You bring main-character energy to a group project 😂",
	"Even autocorrect needs a moment to understand you 😅",
	"Your confidence loaded before the facts did 🤣",
	"You are not late; you are running on premium timing 😎",
}
var gifts = map[string]int{"🌹": 10, "🍫": 20, "🧸": 30, "⭐": 15, "🎁": 40, "💎": 100}

func init() {
	for _, c := range []string{"dice", "ludo", "dart", "basket", "basketball", "football", "bowling"} {
		helpTexts["/"+c] = "<i>Play a Telegram mini-game.</i>"
	}
	helpTexts["/truth"] = "<i>Get a safe random truth question.</i>"
	helpTexts["/dare"] = "<i>Get a safe random dare.</i>"
	helpTexts["/bored"] = "<i>Get a random harmless activity suggestion.</i>"
	helpTexts["/gali"] = "<i>Send a clean, playful roast. Group use is admin-only.</i>"
	helpTexts["/balance"] = "<i>Show your virtual coin balance.</i>"
	helpTexts["/bal"] = helpTexts["/balance"]
	helpTexts["/gifts"] = "<i>Show the virtual gift catalogue.</i>"
	helpTexts["/sendgift"] = "<i>Send a virtual gift using /sendgift @username 🎁.</i>"
	helpTexts["/mygifts"] = "<i>Show your received virtual gifts.</i>"
	helpTexts["/received"] = helpTexts["/mygifts"]
	helpTexts["/top"] = "<i>Show the virtual coin leaderboard.</i>"
	helpTexts["/story"] = "<i>Create a harmless friendship/adventure story from two names.</i>"
}

func diceGameHandler(m *tg.NewMessage) error {
	emoji := "🎲"
	switch strings.ToLower(strings.TrimPrefix(m.GetCommand(), "/")) {
	case "dart":
		emoji = "🎯"
	case "basket", "basketball":
		emoji = "🏀"
	case "football":
		emoji = "⚽"
	case "bowling":
		emoji = "🎳"
	}
	value, err := botDice(m.ChannelID(), emoji)
	if err != nil {
		m.Reply("❌ Game could not be started: " + err.Error())
		return tg.ErrEndGroup
	}
	m.Reply(fmt.Sprintf("🎮 <b>%s</b> result: <b>%d</b>", emoji, value), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}
func truthHandler(m *tg.NewMessage) error {
	funMu.Lock()
	q := safeTruths[rng.Intn(len(safeTruths))]
	funMu.Unlock()
	m.Reply("🧠 <b>Truth</b>\n\n"+q, &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}
func dareHandler(m *tg.NewMessage) error {
	funMu.Lock()
	q := safeDares[rng.Intn(len(safeDares))]
	funMu.Unlock()
	m.Reply("🎯 <b>Dare</b>\n\n"+q, &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}
func boredHandler(m *tg.NewMessage) error {
	funMu.Lock()
	q := boredActivities[rng.Intn(len(boredActivities))]
	funMu.Unlock()
	m.Reply("😴 <b>Try this:</b> "+q, &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}
func galiHandler(m *tg.NewMessage) error {
	if !m.IsPrivate() {
		if ok, _ := utils.IsChatAdmin(m.Client, m.ChannelID(), m.SenderID()); !ok {
			m.Reply("🚫 This fun command is admin-only in groups.")
			return tg.ErrEndGroup
		}
	}
	funMu.Lock()
	text := cleanRoasts[rng.Intn(len(cleanRoasts))]
	funMu.Unlock()
	m.Reply(text)
	return tg.ErrEndGroup
}

func storyHandler(m *tg.NewMessage) error {
	parts := strings.Fields(m.Args())
	if len(parts) < 2 {
		m.Reply("Usage: <code>/story Name1 Name2</code>")
		return tg.ErrEndGroup
	}
	name1, name2 := utils.EscapeHTML(parts[0]), utils.EscapeHTML(strings.Join(parts[1:], " "))
	stories := []string{
		fmt.Sprintf("📚 <b>Mini Story</b>\n\n%s and %s met while working on a group project. They solved the tricky part together and ended the day with a good laugh. ✨", name1, name2),
		fmt.Sprintf("🌧️ <b>Mini Story</b>\n\n%s forgot an umbrella, and %s shared one on the way to the library. They reached their destination with a new friendship and a funny memory. ☔📖", name1, name2),
		fmt.Sprintf("🎮 <b>Mini Story</b>\n\n%s challenged %s to a friendly game. After a very close match, they decided the rematch would settle everything. 🏆", name1, name2),
	}
	funMu.Lock()
	text := stories[rng.Intn(len(stories))]
	funMu.Unlock()
	m.Reply(text, &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}

func ensureCoins(id int64) {
	if _, ok := funCoins[id]; !ok {
		funCoins[id] = 50
	}
}
func balanceHandler(m *tg.NewMessage) error {
	funMu.Lock()
	ensureCoins(m.SenderID())
	c := funCoins[m.SenderID()]
	funMu.Unlock()
	m.Reply(fmt.Sprintf("💰 <b>Balance:</b> %d coins\n🎁 <b>Received:</b> %d", c, len(funGifts[m.SenderID()])), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}
func giftsHandler(m *tg.NewMessage) error {
	var b strings.Builder
	b.WriteString("🎁 <b>Virtual Gifts</b>\n\n")
	keys := make([]string, 0, len(gifts))
	for k := range gifts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return gifts[keys[i]] < gifts[keys[j]] })
	for _, k := range keys {
		fmt.Fprintf(&b, "%s — %d coins\n", k, gifts[k])
	}
	b.WriteString("\nUsage: <code>/sendgift @username 🎁</code>")
	m.Reply(b.String(), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}
func sendGiftHandler(m *tg.NewMessage) error {
	parts := strings.Fields(m.Args())
	if len(parts) < 2 {
		m.Reply("Usage: <code>/sendgift @username 🎁</code>")
		return tg.ErrEndGroup
	}
	gift := parts[1]
	cost, ok := gifts[gift]
	if !ok {
		m.Reply("⚠️ Unknown gift. Use /gifts.")
		return tg.ErrEndGroup
	}
	funMu.Lock()
	ensureCoins(m.SenderID())
	if funCoins[m.SenderID()] < cost {
		have := funCoins[m.SenderID()]
		funMu.Unlock()
		m.Reply(fmt.Sprintf("😕 You need %d coins but have %d.", cost, have))
		return tg.ErrEndGroup
	}
	funCoins[m.SenderID()] -= cost
	funGifts[m.SenderID()] = append(funGifts[m.SenderID()], fmt.Sprintf("%s from %d", gift, m.SenderID()))
	funMu.Unlock()
	m.Reply(fmt.Sprintf("🎉 Virtual gift %s sent to <b>%s</b>!", gift, parts[0]), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}
func myGiftsHandler(m *tg.NewMessage) error {
	funMu.Lock()
	list := append([]string(nil), funGifts[m.SenderID()]...)
	funMu.Unlock()
	if len(list) == 0 {
		m.Reply("🎁 You have no virtual gifts yet.")
		return tg.ErrEndGroup
	}
	var b strings.Builder
	b.WriteString("🎁 <b>Your virtual gifts</b>\n\n")
	for i, v := range list {
		fmt.Fprintf(&b, "%d. %s\n", i+1, v)
	}
	m.Reply(b.String(), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}
func topHandler(m *tg.NewMessage) error {
	funMu.Lock()
	type pair struct {
		id int64
		c  int
	}
	rows := make([]pair, 0, len(funCoins))
	for id, c := range funCoins {
		rows = append(rows, pair{id, c})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].c > rows[j].c })
	funMu.Unlock()
	var b strings.Builder
	b.WriteString("🏆 <b>Virtual Coin Leaderboard</b>\n\n")
	for i, r := range rows {
		if i >= 10 {
			break
		}
		fmt.Fprintf(&b, "%d. <code>%d</code> — %d coins\n", i+1, r.id, r.c)
	}
	if len(rows) == 0 {
		b.WriteString("No users yet.")
	}
	m.Reply(b.String(), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}

/* Original VC participant logger for ArcMusic. */
package modules

import (
	"fmt"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"main/internal/core"
	"main/internal/database"
	"main/internal/utils"
)

var (
	vcMu      sync.Mutex
	vcRunning = map[int64]bool{}
	vcUsers   = map[int64]map[int64]bool{}
)

func vcLoggerHandler(m *tg.NewMessage) error {
	chatID := m.ChannelID()
	ok, _ := utils.IsChatAdmin(m.Client, chatID, m.SenderID())
	if !ok {
		m.Reply("⚠️ Only chat administrators can use this command.")
		return tg.ErrEndGroup
	}
	arg := m.Args()
	if arg == "" {
		enabled, _ := database.GetVCLoggerEnabled(chatID)
		m.Reply(fmt.Sprintf("🎙 <b>VC Logger:</b> %s\nUsage: <code>/vclogger on</code> or <code>/vclogger off</code>", utils.IfElse(enabled, "ON", "OFF")))
		return tg.ErrEndGroup
	}
	enabled, err := utils.ParseBool(arg)
	if err != nil {
		m.Reply("⚠️ Use on/off or enable/disable.")
		return tg.ErrEndGroup
	}
	if err := database.SetVCLoggerEnabled(chatID, enabled); err != nil {
		m.Reply("❌ Failed to save setting: " + utils.EscapeHTML(err.Error()))
		return tg.ErrEndGroup
	}
	vcMu.Lock()
	if enabled {
		vcRunning[chatID] = true
		if vcUsers[chatID] == nil {
			vcUsers[chatID] = map[int64]bool{}
		}
	} else {
		delete(vcRunning, chatID)
		delete(vcUsers, chatID)
	}
	vcMu.Unlock()
	if enabled {
		go monitorVC(chatID)
	}
	m.Reply(utils.IfElse(enabled, "✅ VC logging enabled.", "🚫 VC logging disabled."))
	return tg.ErrEndGroup
}

func monitorVC(chatID int64) {
	vcMu.Lock()
	if vcUsers[chatID] == nil {
		vcUsers[chatID] = map[int64]bool{}
	}
	vcMu.Unlock()
	for {
		vcMu.Lock()
		running := vcRunning[chatID]
		previous := vcUsers[chatID]
		vcMu.Unlock()
		enabled, _ := database.GetVCLoggerEnabled(chatID)
		if !running || !enabled {
			return
		}
		ass, err := core.Assistants.ForChat(chatID)
		if err != nil {
			time.Sleep(8 * time.Second)
			continue
		}
		participants, err := ass.Ntg.GetParticipants(chatID)
		if err != nil {
			time.Sleep(8 * time.Second)
			continue
		}
		current := map[int64]bool{}
		for _, p := range participants {
			if p == nil || p.Peer == nil {
				continue
			}
			if u, ok := p.Peer.(*tg.PeerUser); ok {
				current[u.UserID] = true
			}
		}
		for id := range current {
			if !previous[id] {
				sendVCLog(chatID, id, true)
			}
		}
		for id := range previous {
			if !current[id] {
				sendVCLog(chatID, id, false)
			}
		}
		vcMu.Lock()
		vcUsers[chatID] = current
		vcMu.Unlock()
		time.Sleep(5 * time.Second)
	}
}

func sendVCLog(chatID, userID int64, joined bool) {
	if joined {
		core.Bot.SendMessage(chatID, fmt.Sprintf("🎤 <a href=\"tg://user?id=%d\"><b>User %d</b></a> joined the VC.", userID, userID), &tg.SendOptions{ParseMode: "HTML", LinkPreview: false})
		return
	}
	core.Bot.SendMessage(chatID, fmt.Sprintf("👋 <a href=\"tg://user?id=%d\"><b>User %d</b></a> left the VC.", userID, userID), &tg.SendOptions{ParseMode: "HTML", LinkPreview: false})
}

func initVCLoggers() {
	chats, err := database.ServedChats()
	if err != nil {
		return
	}
	for _, chatID := range chats {
		enabled, _ := database.GetVCLoggerEnabled(chatID)
		if enabled {
			vcMu.Lock()
			vcRunning[chatID] = true
			vcUsers[chatID] = map[int64]bool{}
			vcMu.Unlock()
			go monitorVC(chatID)
		}
	}
}

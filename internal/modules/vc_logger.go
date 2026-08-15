/*
 * Original VC participant logger for ArcMusic.
 */

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

// ============================================================
// VC LOGGER COMMAND
// ============================================================

func vcLoggerHandler(m *tg.NewMessage) error {

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

	arg := m.Args()

	// --------------------------------------------------------
	// STATUS
	// --------------------------------------------------------

	if arg == "" {

		enabled, _ := database.GetVCLoggerEnabled(chatID)

		m.Reply(
			fmt.Sprintf(
				"🎙 <b>VC Logger:</b> %s\nUsage: <code>/vclogger on</code> or <code>/vclogger off</code>",
				utils.IfElse(
					enabled,
					"ON",
					"OFF",
				),
			),
		)

		return tg.ErrEndGroup
	}

	// --------------------------------------------------------
	// PARSE ON/OFF
	// --------------------------------------------------------

	enabled, err := utils.ParseBool(arg)

	if err != nil {

		m.Reply(
			"⚠️ Use on/off or enable/disable.",
		)

		return tg.ErrEndGroup
	}

	// --------------------------------------------------------
	// SAVE SETTING
	// --------------------------------------------------------

	if err := database.SetVCLoggerEnabled(
		chatID,
		enabled,
	); err != nil {

		m.Reply(
			"❌ Failed to save setting: "+
				utils.EscapeHTML(err.Error()),
		)

		return tg.ErrEndGroup
	}

	// --------------------------------------------------------
	// UPDATE RUNNING STATE
	// --------------------------------------------------------

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

	// --------------------------------------------------------
	// START MONITOR
	// --------------------------------------------------------

	if enabled {
		go monitorVC(chatID)
	}

	// --------------------------------------------------------
	// RESPONSE
	// --------------------------------------------------------

	m.Reply(
		utils.IfElse(
			enabled,
			"✅ VC logging enabled.",
			"🚫 VC logging disabled.",
		),
	)

	return tg.ErrEndGroup
}

// ============================================================
// VC MONITOR
// ============================================================

func monitorVC(chatID int64) {

	vcMu.Lock()

	if vcUsers[chatID] == nil {
		vcUsers[chatID] = map[int64]bool{}
	}

	vcMu.Unlock()

	for {

		// ----------------------------------------------------
		// CHECK RUNNING
		// ----------------------------------------------------

		vcMu.Lock()

		running := vcRunning[chatID]
		previous := vcUsers[chatID]

		vcMu.Unlock()

		enabled, _ := database.GetVCLoggerEnabled(chatID)

		if !running || !enabled {
			return
		}

		// ----------------------------------------------------
		// GET ASSISTANT
		// ----------------------------------------------------

		ass, err := core.Assistants.ForChat(chatID)

		if err != nil {

			time.Sleep(
				8 * time.Second,
			)

			continue
		}

		// ----------------------------------------------------
		// GET VC PARTICIPANTS
		// ----------------------------------------------------

		participants, err := ass.Ntg.GetParticipants(chatID)

		if err != nil {

			time.Sleep(
				8 * time.Second,
			)

			continue
		}

		current := map[int64]bool{}

		// ----------------------------------------------------
		// BUILD CURRENT USER LIST
		// ----------------------------------------------------

		for _, p := range participants {

			if p == nil || p.Peer == nil {
				continue
			}

			if u, ok := p.Peer.(*tg.PeerUser); ok {
				current[u.UserID] = true
			}
		}

		// ----------------------------------------------------
		// USERS JOINED
		// ----------------------------------------------------

		for id := range current {

			if !previous[id] {

				go sendVCLog(
					chatID,
					id,
					true,
				)
			}
		}

		// ----------------------------------------------------
		// USERS LEFT
		// ----------------------------------------------------

		for id := range previous {

			if !current[id] {

				go sendVCLog(
					chatID,
					id,
					false,
				)
			}
		}

		// ----------------------------------------------------
		// UPDATE USERS
		// ----------------------------------------------------

		vcMu.Lock()

		vcUsers[chatID] = current

		vcMu.Unlock()

		// ----------------------------------------------------
		// NEXT CHECK
		// ----------------------------------------------------

		time.Sleep(
			5 * time.Second,
		)
	}
}

// ============================================================
// SEND VC LOG + AUTO DELETE AFTER 3 SECONDS
// ============================================================

func sendVCLog(
	chatID int64,
	userID int64,
	joined bool,
) {

	var text string

	if joined {

		text = fmt.Sprintf(
			"🎤 <a href=\"tg://user?id=%d\"><b>User %d</b></a> joined the VC.",
			userID,
			userID,
		)

	} else {

		text = fmt.Sprintf(
			"👋 <a href=\"tg://user?id=%d\"><b>User %d</b></a> left the VC.",
			userID,
			userID,
		)
	}

	// --------------------------------------------------------
	// SEND MESSAGE
	// --------------------------------------------------------

	msg, err := core.Bot.SendMessage(
		chatID,
		text,
		&tg.SendOptions{
			ParseMode:   "HTML",
			LinkPreview: false,
		},
	)

	if err != nil {
		return
	}

	if msg == nil {
		return
	}

	// --------------------------------------------------------
	// DELETE AFTER 3 SECONDS
	// --------------------------------------------------------

	time.Sleep(
		3 * time.Second,
	)

	_, err = msg.Delete()

	if err != nil {
		return
	}
}

// ============================================================
// INITIALIZE VC LOGGERS
// ============================================================

func initVCLoggers() {

	chats, err := database.ServedChats()

	if err != nil {
		return
	}

	for _, chatID := range chats {

		enabled, _ := database.GetVCLoggerEnabled(
			chatID,
		)

		if enabled {

			vcMu.Lock()

			vcRunning[chatID] = true
			vcUsers[chatID] = map[int64]bool{}

			vcMu.Unlock()

			go monitorVC(chatID)
		}
	}
}

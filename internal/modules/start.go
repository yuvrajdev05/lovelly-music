/*
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Arc
 */

package modules

import (
	"time"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/config"
	"main/internal/core"
	"main/internal/database"
	"main/internal/locales"
	"main/internal/utils"
)

func init() {
	helpTexts["/start"] = `<i>Start the bot and show main menu.</i>`
}

// InnoMusic sticker Bot API file ID.
const startStickerID = "CAACAgUAAxkBAAEf9h1qRSPKlYwtAAFFrNVl-JWpojGWyT8AAjAJAAKJsAFXMkMs6wG9EwY8BA"

func startHandler(m *tg.NewMessage) error {
	if m.ChatType() != tg.EntityUser {
		database.AddServedChat(m.ChannelID())

		m.Reply(
			F(m.ChannelID(), "start_group"),
		)

		return tg.ErrEndGroup
	}

	arg := m.Args()

	database.AddServedUser(m.ChannelID())

	if arg != "" {
		gologging.Info(
			"Got Start parameter: "+arg+" in ChatID: "+utils.IntToStr(
				m.ChannelID(),
			),
		)
	}

	switch arg {

	case "pm_help":
		gologging.Info("User requested help via start param")

		helpHandler(m)

	default:

		// ---------------------------------------------------------
		// START ANIMATION
		// ---------------------------------------------------------

		msg1, _ := m.Respond(
			"🌟 𝑾𝒆𝒍𝒄𝒐𝒎𝒆 𝑻𝒐 𝑳𝒐𝒗𝒆𝒍𝒍𝒚 𝑴𝒖𝒔𝒊𝒄 🌟",
			nil,
		)

		time.Sleep(1 * time.Second)

		if msg1 != nil {
			_, _ = msg1.Delete()
		}

		msg2, _ := m.Respond(
			"𓆩🎵𓆪 𝑩𝒆𝒔𝒕 𝑴𝒖𝒔𝒊𝒄 𝑬𝒙𝒑𝒆𝒓𝒊𝒆𝒏𝒄𝒆 𓆩🎵𓆪",
			nil,
		)

		time.Sleep(1 * time.Second)

		if msg2 != nil {
			_, _ = msg2.Delete()
		}

		msg3, _ := m.Respond(
			"💖𓆩🎶𓆪 𝑯𝒊𝒈𝒉 𝑸𝒖𝒂𝒍𝒊𝒕𝒚 𝑺𝒕𝒓𝒆𝒂𝒎𝒊𝒏𝒈 𓆩🎶𓆪💖",
			nil,
		)

		time.Sleep(1 * time.Second)

		if msg3 != nil {
			_, _ = msg3.Delete()
		}

		msg4, _ := m.Respond(
			`✨ ✦ 𝑷𝒐𝒘𝒆𝒓𝒆𝒅 𝑩𝒚 ✦ <a href="https://t.me/x_yuvii">𖹭 𝒀𝒖𝒗𝒊𝒊 𖹭</a> ✨`,
			&tg.SendOptions{
				ParseMode: "HTML",
			},
		)

		time.Sleep(1 * time.Second)

		if msg4 != nil {
			_, _ = msg4.Delete()
		}

		// ---------------------------------------------------------
		// START STICKER
		// ---------------------------------------------------------

		stickerMedia, stickerErr := tg.ResolveBotFileID(startStickerID)

		if stickerErr != nil {
			gologging.Error(
				"[start] Sticker file ID resolve failed: " +
					stickerErr.Error(),
			)
		} else {

			stickerMsg, sendErr := m.ReplyMedia(stickerMedia)

			if sendErr != nil {
				gologging.Error(
					"[start] Sticker reply failed: " +
						sendErr.Error(),
				)
			} else if stickerMsg != nil {

				// Keep sticker for 2.5 seconds, then delete it.
				time.Sleep(2500 * time.Millisecond)

				_, _ = stickerMsg.Delete()
			}
		}

		// ---------------------------------------------------------
		// FINAL START MESSAGE
		// ---------------------------------------------------------

		time.Sleep(1 * time.Second)

		caption := F(
			m.ChannelID(),
			"start_private",
			locales.Arg{
				"user": utils.MentionHTML(m.Sender),
				"bot":  utils.MentionHTML(m.Client.Me()),
			},
		)

		_, err := m.RespondMedia(
			&tg.InputMediaWebPage{
				URL:             config.StartImage,
				ForceLargeMedia: true,
			},
			&tg.MediaOptions{
				Caption:     caption,
				NoForwards:  true,
				ReplyMarkup: core.GetStartMarkup(m.ChannelID()),
			},
		)

		if err != nil {

			gologging.Error(
				"[start] InputMediaWebPage Reply failed: " +
					err.Error(),
			)

			// Fallback: send StartImage directly.
			_, err = m.RespondMedia(
				config.StartImage,
				&tg.MediaOptions{
					Caption:     caption,
					NoForwards:  true,
					ReplyMarkup: core.GetStartMarkup(m.ChannelID()),
				},
			)

			if err != nil {

				gologging.Error(
					"[start] URL media reply failed: " +
						err.Error(),
				)

				// Final fallback: text only.
				_, err = m.Respond(
					caption,
					&tg.SendOptions{
						NoForwards:  true,
						ReplyMarkup: core.GetStartMarkup(m.ChannelID()),
					},
				)

				return err
			}
		}
	}

	// -------------------------------------------------------------
	// LOGGER
	// -------------------------------------------------------------

	if config.LoggerID != 0 && isLoggerEnabled() {

		uName := "N/A"

		if m.Sender.Username != "" {
			uName = "@" + m.Sender.Username
		}

		msg := F(
			m.ChannelID(),
			"logger_bot_started",
			locales.Arg{
				"mention":        utils.MentionHTML(m.Sender),
				"user_id":        m.SenderID(),
				"user_username": uName,
			},
		)

		_, err := m.Client.SendMessage(
			config.LoggerID,
			msg,
		)

		if err != nil {
			gologging.Error(
				"Failed to send logger_bot_started msg, Err: " +
					err.Error(),
			)
		}
	}

	return tg.ErrEndGroup
}

// -------------------------------------------------------------
// START CALLBACK
// -------------------------------------------------------------

func startCB(cb *tg.CallbackQuery) error {

	cb.Answer("")

	caption := F(
		cb.ChannelID(),
		"start_private",
		locales.Arg{
			"user": utils.MentionHTML(cb.Sender),
			"bot":  utils.MentionHTML(cb.Client.Me()),
		},
	)

	sendOpt := &tg.SendOptions{
		ReplyMarkup: core.GetStartMarkup(cb.ChannelID()),
		NoForwards:  true,
	}

	if config.StartImage != "" {

		sendOpt.Media = &tg.InputMediaWebPage{
			URL:             config.StartImage,
			ForceLargeMedia: true,
		}
	}

	cb.Edit(
		caption,
		sendOpt,
	)

	return tg.ErrEndGroup
}

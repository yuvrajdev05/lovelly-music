package modules

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/config"
	"main/internal/database"
	"main/internal/utils"
)

func init() {
	helpTexts["/ban"] = "<i>Ban a user. Reply to a message or use /ban &lt;user_id/@username&gt; [reason].</i>"
	helpTexts["/unban"] = "<i>Unban a user.</i>"
	helpTexts["/tban"] = "<i>Temporarily ban a user, e.g. /tban 10m spam.</i>"
	helpTexts["/kick"] = "<i>Remove a user and immediately allow them to rejoin.</i>"
	helpTexts["/mute"] = "<i>Mute a user until manually unmuted.</i>"
	helpTexts["/tmute"] = "<i>Temporarily mute a user, e.g. /tmute 30m.</i>"
	helpTexts["/unmute"] = "<i>Restore a user's normal chat permissions.</i>"
	helpTexts["/warn"] = "<i>Add a warning to a user. Three warnings reset the counter and temporarily mute.</i>"
	helpTexts["/warns"] = "<i>Show a user's warning count.</i>"
	helpTexts["/rmwarns"] = "<i>Clear a user's warnings.</i>"
	helpTexts["/del"] = "<i>Delete the replied message.</i>"
	helpTexts["/purge"] = "<i>Delete the replied message and the command message.</i>"
	helpTexts["/pin"] = "<i>Pin the replied message.</i>"
	helpTexts["/unpin"] = "<i>Unpin the replied message.</i>"
	helpTexts["/promote"] = "<i>Promote a member with basic admin permissions.</i>"
	helpTexts["/demote"] = "<i>Remove admin privileges from a member.</i>"
	helpTexts["/report"] = "<i>Report a replied message to the chat administrators.</i>"
	helpTexts["/block"] = "<i>Sudo-only global user block list.</i>"
	helpTexts["/unblock"] = "<i>Remove a user from the global block list.</i>"
	helpTexts["/blocked"] = "<i>Show globally blocked users.</i>"
	helpTexts["/gban"] = "<i>Sudo-only global ban across served chats.</i>"
	helpTexts["/ungban"] = "<i>Remove a global ban.</i>"
	helpTexts["/gbanlist"] = "<i>Show globally blocked users.</i>"
	helpTexts["/zombies"] = "<i>Find and remove deleted accounts from the current group.</i>"
	helpTexts["/banall"] = "<i>Admin-only mass ban. Requires the exact confirmation phrase: /banall CONFIRM.</i>"
	helpTexts["/unpinall"] = "<i>Unpin all pinned messages in the current chat.</i>"
	helpTexts["/blchat"] = "<i>Block a chat from using the bot. Sudo only.</i>"
	helpTexts["/unblchat"] = "<i>Remove a chat from the bot block list.</i>"
	helpTexts["/blchats"] = "<i>Show chats blocked from using the bot.</i>"
}

func targetUser(m *tg.NewMessage) (int64, error) {
	return utils.ExtractUser(m)
}

func parseFeatureDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("duration is required")
	}
	if strings.HasSuffix(s, "d") {
		n, err := strconv.ParseInt(strings.TrimSuffix(s, "d"), 10, 64)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("invalid duration")
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("use values like 10m, 2h or 1d")
	}
	return d, nil
}

func doBan(m *tg.NewMessage, temporary bool, kick bool) error {
	if m.IsPrivate() {
		m.Reply("⚠️ This command works in groups only.")
		return tg.ErrEndGroup
	}
	uid, err := targetUser(m)
	if err != nil {
		m.Reply("⚠️ I couldn't identify that user.")
		return tg.ErrEndGroup
	}
	if uid == m.Client.Me().ID {
		m.Reply("⚠️ I can't moderate myself.")
		return tg.ErrEndGroup
	}
	if config.OwnerID != 0 && uid == config.OwnerID {
		m.Reply("⚠️ I won't moderate the bot owner.")
		return tg.ErrEndGroup
	}
	until := int64(0)
	args := strings.Fields(m.Args())
	if temporary {
		if len(args) == 0 {
			m.Reply("Usage: <code>/tban 10m reason</code>")
			return tg.ErrEndGroup
		}
		d, e := parseFeatureDuration(args[0])
		if e != nil {
			m.Reply("⚠️ " + e.Error())
			return tg.ErrEndGroup
		}
		until = time.Now().Add(d).Unix()
	}
	if err := botAPIInt("banChatMember", m.ChannelID(), uid, until); err != nil {
		m.Reply("❌ Ban failed: " + utils.EscapeHTML(err.Error()))
		return tg.ErrEndGroup
	}
	if kick {
		_ = botAPIInt("unbanChatMember", m.ChannelID(), uid, 0)
	}
	action := "banned"
	if kick {
		action = "kicked"
	}
	if temporary {
		action = "temporarily banned"
	}
	m.Reply(fmt.Sprintf("✅ User <code>%d</code> %s.", uid, action))
	return tg.ErrEndGroup
}

func banFeatureHandler(m *tg.NewMessage) error  { return doBan(m, false, false) }
func tbanFeatureHandler(m *tg.NewMessage) error { return doBan(m, true, false) }
func kickFeatureHandler(m *tg.NewMessage) error { return doBan(m, false, true) }

func unbanFeatureHandler(m *tg.NewMessage) error {
	uid, err := targetUser(m)
	if err != nil {
		m.Reply("⚠️ I couldn't identify that user.")
		return tg.ErrEndGroup
	}
	if err := botAPIInt("unbanChatMember", m.ChannelID(), uid, 0); err != nil {
		m.Reply("❌ Unban failed: " + utils.EscapeHTML(err.Error()))
		return tg.ErrEndGroup
	}
	m.Reply(fmt.Sprintf("✅ User <code>%d</code> unbanned.", uid))
	return tg.ErrEndGroup
}

func restrictUser(m *tg.NewMessage, temporary bool, enable bool) error {
	uid, err := targetUser(m)
	if err != nil {
		m.Reply("⚠️ I couldn't identify that user.")
		return tg.ErrEndGroup
	}
	until := int64(0)
	if temporary {
		args := strings.Fields(m.Args())
		if len(args) == 0 {
			m.Reply("Usage: <code>/tmute 30m reason</code>")
			return tg.ErrEndGroup
		}
		d, e := parseFeatureDuration(args[0])
		if e != nil {
			m.Reply("⚠️ " + e.Error())
			return tg.ErrEndGroup
		}
		until = time.Now().Add(d).Unix()
	}
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(m.ChannelID(), 10))
	form.Set("user_id", strconv.FormatInt(uid, 10))
	if until > 0 {
		form.Set("until_date", strconv.FormatInt(until, 10))
	}
	if enable {
		form.Set("permissions", `{"can_send_messages":true,"can_send_audios":true,"can_send_documents":true,"can_send_photos":true,"can_send_videos":true,"can_send_video_notes":true,"can_send_voice_notes":true,"can_send_polls":true,"can_send_other_messages":true,"can_add_web_page_previews":true}`)
	} else {
		form.Set("permissions", `{"can_send_messages":false,"can_send_audios":false,"can_send_documents":false,"can_send_photos":false,"can_send_videos":false,"can_send_video_notes":false,"can_send_voice_notes":false,"can_send_polls":false,"can_send_other_messages":false,"can_add_web_page_previews":false}`)
	}
	if err := botAPI("restrictChatMember", form, nil); err != nil {
		m.Reply("❌ Permission update failed: " + utils.EscapeHTML(err.Error()))
		return tg.ErrEndGroup
	}
	if enable {
		m.Reply(fmt.Sprintf("🔊 User <code>%d</code> unmuted.", uid))
	} else if temporary {
		m.Reply(fmt.Sprintf("🔇 User <code>%d</code> muted temporarily.", uid))
	} else {
		m.Reply(fmt.Sprintf("🔇 User <code>%d</code> muted.", uid))
	}
	return tg.ErrEndGroup
}
func muteFeatureHandler(m *tg.NewMessage) error   { return restrictUser(m, false, false) }
func tmuteFeatureHandler(m *tg.NewMessage) error  { return restrictUser(m, true, false) }
func unmuteFeatureHandler(m *tg.NewMessage) error { return restrictUser(m, false, true) }

func warnFeatureHandler(m *tg.NewMessage) error {
	uid, err := targetUser(m)
	if err != nil {
		m.Reply("⚠️ I couldn't identify that user.")
		return tg.ErrEndGroup
	}
	count, err := database.GetWarn(m.ChannelID(), uid)
	if err != nil {
		m.Reply("❌ Database error.")
		return tg.ErrEndGroup
	}
	count++
	if count >= 3 {
		_ = database.SetWarn(m.ChannelID(), uid, 0)
		_ = restrictUser(m, false, false)
		m.Reply(fmt.Sprintf("⚠️ User <code>%d</code> reached 3/3 warnings and was muted.", uid))
		return tg.ErrEndGroup
	}
	_ = database.SetWarn(m.ChannelID(), uid, count)
	m.Reply(fmt.Sprintf("⚠️ Warning added to <code>%d</code>.\nWarnings: <b>%d/3</b>", uid, count))
	return tg.ErrEndGroup
}
func warnsFeatureHandler(m *tg.NewMessage) error {
	uid, err := targetUser(m)
	if err != nil {
		m.Reply("⚠️ I couldn't identify that user.")
		return tg.ErrEndGroup
	}
	count, err := database.GetWarn(m.ChannelID(), uid)
	if err != nil {
		m.Reply("❌ Database error.")
		return tg.ErrEndGroup
	}
	m.Reply(fmt.Sprintf("⚠️ User <code>%d</code> has <b>%d/3</b> warnings.", uid, count))
	return tg.ErrEndGroup
}
func rmWarnsFeatureHandler(m *tg.NewMessage) error {
	uid, err := targetUser(m)
	if err != nil {
		m.Reply("⚠️ I couldn't identify that user.")
		return tg.ErrEndGroup
	}
	if err := database.SetWarn(m.ChannelID(), uid, 0); err != nil {
		m.Reply("❌ Database error.")
		return tg.ErrEndGroup
	}
	m.Reply(fmt.Sprintf("✅ Warnings cleared for <code>%d</code>.", uid))
	return tg.ErrEndGroup
}

func deleteFeatureHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("⚠️ Reply to a message to delete it.")
		return tg.ErrEndGroup
	}
	r, err := m.GetReplyMessage()
	if err != nil || r == nil {
		m.Reply("⚠️ Couldn't load the replied message.")
		return tg.ErrEndGroup
	}
	_, _ = r.Delete()
	_, _ = m.Delete()
	return tg.ErrEndGroup
}
func purgeFeatureHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("⚠️ Reply to the first message you want to delete.")
		return tg.ErrEndGroup
	}
	r, err := m.GetReplyMessage()
	if err != nil || r == nil {
		return tg.ErrEndGroup
	}
	_ = botDelete(m.ChannelID(), r.ID)
	_, _ = m.Delete()
	return tg.ErrEndGroup
}

func pinFeatureHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("⚠️ Reply to a message to pin/unpin it.")
		return tg.ErrEndGroup
	}
	r, err := m.GetReplyMessage()
	if err != nil || r == nil {
		return tg.ErrEndGroup
	}
	if strings.EqualFold(m.GetCommand(), "/unpin") {
		_ = botUnpin(m.ChannelID(), r.ID)
		m.Reply("📌 Message unpinned.")
	} else {
		_ = botPin(m.ChannelID(), r.ID, false)
		m.Reply("📌 Message pinned.")
	}
	return tg.ErrEndGroup
}

func promoteFeatureHandler(m *tg.NewMessage) error {
	uid, err := targetUser(m)
	if err != nil {
		m.Reply("⚠️ I couldn't identify that user.")
		return tg.ErrEndGroup
	}
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(m.ChannelID(), 10))
	form.Set("user_id", strconv.FormatInt(uid, 10))
	form.Set("can_manage_chat", "true")
	form.Set("can_change_info", "false")
	form.Set("can_post_messages", "false")
	form.Set("can_edit_messages", "false")
	form.Set("can_delete_messages", "true")
	form.Set("can_invite_users", "true")
	form.Set("can_restrict_members", "true")
	form.Set("can_pin_messages", "true")
	form.Set("can_promote_members", "false")
	form.Set("can_manage_video_chats", "true")
	if err := botAPI("promoteChatMember", form, nil); err != nil {
		m.Reply("❌ Promote failed: " + utils.EscapeHTML(err.Error()))
		return tg.ErrEndGroup
	}
	m.Reply(fmt.Sprintf("✅ User <code>%d</code> promoted.", uid))
	return tg.ErrEndGroup
}
func demoteFeatureHandler(m *tg.NewMessage) error {
	uid, err := targetUser(m)
	if err != nil {
		m.Reply("⚠️ I couldn't identify that user.")
		return tg.ErrEndGroup
	}
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(m.ChannelID(), 10))
	form.Set("user_id", strconv.FormatInt(uid, 10))
	for _, k := range []string{"can_manage_chat", "can_change_info", "can_post_messages", "can_edit_messages", "can_delete_messages", "can_invite_users", "can_restrict_members", "can_pin_messages", "can_promote_members", "can_manage_video_chats"} {
		form.Set(k, "false")
	}
	if err := botAPI("promoteChatMember", form, nil); err != nil {
		m.Reply("❌ Demote failed: " + utils.EscapeHTML(err.Error()))
		return tg.ErrEndGroup
	}
	m.Reply(fmt.Sprintf("✅ User <code>%d</code> demoted.", uid))
	return tg.ErrEndGroup
}

func unpinAllFeatureHandler(m *tg.NewMessage) error {
	if err := botUnpinAll(m.ChannelID()); err != nil {
		m.Reply("❌ Unpin all failed: " + utils.EscapeHTML(err.Error()))
		return tg.ErrEndGroup
	}
	m.Reply("✅ All pinned messages were unpinned.")
	return tg.ErrEndGroup
}

func zombiesFeatureHandler(m *tg.NewMessage) error {
	members, _, err := m.Client.GetChatMembers(m.ChannelID(), &tg.ParticipantOptions{Limit: -1, SleepThresholdMs: 3000})
	if err != nil {
		m.Reply("❌ Couldn't fetch chat members: " + utils.EscapeHTML(err.Error()))
		return tg.ErrEndGroup
	}
	removed := 0
	found := 0
	for _, member := range members {
		if member == nil || member.User == nil || !member.User.Deleted {
			continue
		}
		found++
		if err := botAPIInt("banChatMember", m.ChannelID(), member.User.ID, 0); err == nil {
			removed++
		}
	}
	m.Reply(fmt.Sprintf("🧟 Deleted accounts found: <b>%d</b>\n✅ Removed: <b>%d</b>", found, removed), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}

func banAllFeatureHandler(m *tg.NewMessage) error {
	if !strings.EqualFold(strings.TrimSpace(m.Args()), "CONFIRM") {
		m.Reply("⚠️ This affects many members. If you really intend to do it, use <code>/banall CONFIRM</code>.", &tg.SendOptions{ParseMode: "HTML"})
		return tg.ErrEndGroup
	}
	members, _, err := m.Client.GetChatMembers(m.ChannelID(), &tg.ParticipantOptions{Limit: -1, SleepThresholdMs: 3000})
	if err != nil {
		m.Reply("❌ Couldn't fetch members: " + utils.EscapeHTML(err.Error()))
		return tg.ErrEndGroup
	}
	admins, _, _ := m.Client.GetChatMembers(m.ChannelID(), &tg.ParticipantOptions{Filter: &tg.ChannelParticipantsAdmins{}, Limit: -1, SleepThresholdMs: 3000})
	adminIDs := make(map[int64]struct{}, len(admins))
	for _, a := range admins {
		if a != nil && a.User != nil {
			adminIDs[a.User.ID] = struct{}{}
		}
	}
	banned := 0
	for _, member := range members {
		if member == nil || member.User == nil || member.User.Bot || member.User.Deleted {
			continue
		}
		if _, isAdmin := adminIDs[member.User.ID]; isAdmin {
			continue
		}
		if member.User.ID == m.Client.Me().ID || (config.OwnerID != 0 && member.User.ID == config.OwnerID) {
			continue
		}
		if err := botAPIInt("banChatMember", m.ChannelID(), member.User.ID, 0); err == nil {
			banned++
		}
	}
	m.Reply(fmt.Sprintf("⚠️ Mass moderation finished. Users banned: <b>%d</b>", banned), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}

func reportFeatureHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("⚠️ Reply to the message you want to report.")
		return tg.ErrEndGroup
	}
	admins, _, err := m.Client.GetChatMembers(m.ChannelID(), &tg.ParticipantOptions{Filter: &tg.ChannelParticipantsAdmins{}, Limit: -1})
	if err != nil {
		m.Reply("❌ Couldn't fetch administrators.")
		return tg.ErrEndGroup
	}
	lines := []string{"🚨 <b>Admin Report</b>", fmt.Sprintf("Chat: <code>%d</code>", m.ChannelID())}
	for _, p := range admins {
		if p != nil && p.User != nil && !p.User.Bot && !p.User.Deleted {
			lines = append(lines, utils.MentionHTML(p.User))
		}
	}
	lines = append(lines, fmt.Sprintf("\nReported by: <code>%d</code>", m.SenderID()))
	m.Reply(strings.Join(lines, "\n"), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}

func blockFeatureHandler(m *tg.NewMessage) error {
	uid, err := targetUser(m)
	if err != nil {
		m.Reply("⚠️ I couldn't identify that user.")
		return tg.ErrEndGroup
	}
	if err := database.AddBlockedUser(uid); err != nil {
		m.Reply("❌ Database error.")
		return tg.ErrEndGroup
	}
	m.Reply(fmt.Sprintf("🚫 User <code>%d</code> added to the global block list.", uid))
	return tg.ErrEndGroup
}
func unblockFeatureHandler(m *tg.NewMessage) error {
	uid, err := targetUser(m)
	if err != nil {
		m.Reply("⚠️ I couldn't identify that user.")
		return tg.ErrEndGroup
	}
	if err := database.RemoveBlockedUser(uid); err != nil {
		m.Reply("❌ Database error.")
		return tg.ErrEndGroup
	}
	m.Reply(fmt.Sprintf("✅ User <code>%d</code> removed from the global block list.", uid))
	return tg.ErrEndGroup
}
func blockedFeatureHandler(m *tg.NewMessage) error {
	ids, err := database.BlockedUsers()
	if err != nil {
		m.Reply("❌ Database error.")
		return tg.ErrEndGroup
	}
	if len(ids) == 0 {
		m.Reply("ℹ️ No globally blocked users.")
		return tg.ErrEndGroup
	}
	var b strings.Builder
	b.WriteString("🚫 <b>Blocked Users</b>\n\n")
	for i, id := range ids {
		fmt.Fprintf(&b, "%d. <code>%d</code>\n", i+1, id)
	}
	m.Reply(b.String(), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}
func gbanFeatureHandler(m *tg.NewMessage) error {
	uid, err := targetUser(m)
	if err != nil {
		m.Reply("⚠️ I couldn't identify that user.")
		return tg.ErrEndGroup
	}
	if err := database.AddBlockedUser(uid); err != nil {
		m.Reply("❌ Database error.")
		return tg.ErrEndGroup
	}
	chats, err := database.ServedChats()
	if err == nil {
		for _, chat := range chats {
			_ = botAPIInt("banChatMember", chat, uid, 0)
		}
	}
	m.Reply(fmt.Sprintf("🌐 User <code>%d</code> globally banned from served chats.", uid))
	return tg.ErrEndGroup
}
func ungbanFeatureHandler(m *tg.NewMessage) error {
	uid, err := targetUser(m)
	if err != nil {
		m.Reply("⚠️ I couldn't identify that user.")
		return tg.ErrEndGroup
	}
	_ = database.RemoveBlockedUser(uid)
	chats, err := database.ServedChats()
	if err == nil {
		for _, chat := range chats {
			_ = botAPIInt("unbanChatMember", chat, uid, 0)
		}
	}
	m.Reply(fmt.Sprintf("✅ Global ban removed for <code>%d</code>.", uid))
	return tg.ErrEndGroup
}
func gbanListFeatureHandler(m *tg.NewMessage) error { return blockedFeatureHandler(m) }

func blockChatFeatureHandler(m *tg.NewMessage) error {
	args := strings.Fields(m.Args())
	if len(args) < 1 {
		m.Reply("Usage: <code>/blchat -1001234567890</code>")
		return tg.ErrEndGroup
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		m.Reply("⚠️ Invalid chat ID.")
		return tg.ErrEndGroup
	}
	if err := database.AddBlacklistedChat(id); err != nil {
		m.Reply("❌ Database error.")
		return tg.ErrEndGroup
	}
	// Match the Python bot behavior: once a chat is blacklisted, leave it.
	if id == m.ChannelID() {
		_ = m.Client.LeaveChannel(id)
	}
	m.Reply(fmt.Sprintf("🚫 Chat <code>%d</code> is now blocked.", id))
	return tg.ErrEndGroup
}
func unblockChatFeatureHandler(m *tg.NewMessage) error {
	args := strings.Fields(m.Args())
	if len(args) < 1 {
		m.Reply("Usage: <code>/unblchat -1001234567890</code>")
		return tg.ErrEndGroup
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		m.Reply("⚠️ Invalid chat ID.")
		return tg.ErrEndGroup
	}
	if err := database.RemoveBlacklistedChat(id); err != nil {
		m.Reply("❌ Database error.")
		return tg.ErrEndGroup
	}
	m.Reply(fmt.Sprintf("✅ Chat <code>%d</code> removed from the block list.", id))
	return tg.ErrEndGroup
}
func blockedChatsFeatureHandler(m *tg.NewMessage) error {
	ids, err := database.BlacklistedChats()
	if err != nil {
		m.Reply("❌ Database error.")
		return tg.ErrEndGroup
	}
	if len(ids) == 0 {
		m.Reply("ℹ️ No blocked chats.")
		return tg.ErrEndGroup
	}
	var b strings.Builder
	b.WriteString("🚫 <b>Blocked Chats</b>\n\n")
	for i, id := range ids {
		fmt.Fprintf(&b, "%d. <code>%d</code>\n", i+1, id)
	}
	m.Reply(b.String(), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}

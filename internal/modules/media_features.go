package modules

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func init() {
	helpTexts["/tgm"] = "<i>Upload replied media to Catbox and return a public link. Maximum 200 MB.</i>"
	helpTexts["/tts"] = "<i>Convert text to Hindi speech audio.</i>"
}

func tgmHandler(m *tg.NewMessage) error {
	if !m.IsReply() {
		m.Reply("⚠️ Reply to a photo, video, document or audio with /tgm.")
		return tg.ErrEndGroup
	}
	reply, err := m.GetReplyMessage()
	if err != nil || reply == nil {
		m.Reply("⚠️ Couldn't load the replied media.")
		return tg.ErrEndGroup
	}
	status, _ := m.Reply("⏳ Downloading media...")
	dir := "downloads/tgm"
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, fmt.Sprintf("tgm_%d_%d", m.ChannelID(), time.Now().UnixNano()))
	path, err = reply.Download(&tg.DownloadOptions{FileName: path})
	if err != nil {
		if status != nil {
			status.Edit("❌ Download failed: " + err.Error())
		}
		return tg.ErrEndGroup
	}
	defer os.Remove(path)
	if info, e := os.Stat(path); e == nil && info.Size() > 200*1024*1024 {
		if status != nil {
			status.Edit("⚠️ File is larger than 200 MB.")
		}
		return tg.ErrEndGroup
	}
	if status != nil {
		status.Edit("📤 Uploading...")
	}
	link, err := uploadCatbox(path)
	if err != nil {
		if status != nil {
			status.Edit("❌ Upload failed: " + err.Error())
		}
		return tg.ErrEndGroup
	}
	if status != nil {
		status.Edit(fmt.Sprintf("🌐 <a href=\"%s\">Open uploaded file</a>", link), &tg.SendOptions{ParseMode: "HTML", LinkPreview: true})
	}
	return tg.ErrEndGroup
}

func uploadCatbox(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		defer pw.Close()
		defer mw.Close()
		_ = mw.WriteField("reqtype", "fileupload")
		part, e := mw.CreateFormFile("fileToUpload", filepath.Base(path))
		if e != nil {
			return
		}
		_, _ = io.Copy(part, file)
	}()
	req, err := http.NewRequest(http.MethodPost, "https://catbox.moe/user/api.php", pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := (&http.Client{Timeout: 90 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("upload returned HTTP %d", resp.StatusCode)
	}
	link := strings.TrimSpace(string(data))
	if link == "" {
		return "", fmt.Errorf("empty upload response")
	}
	return link, nil
}

func ttsHandler(m *tg.NewMessage) error {
	text := strings.TrimSpace(m.Args())
	if text == "" {
		m.Reply("Usage: <code>/tts your text</code>")
		return tg.ErrEndGroup
	}
	if len([]rune(text)) > 300 {
		m.Reply("⚠️ Please keep TTS text under 300 characters.")
		return tg.ErrEndGroup
	}
	status, _ := m.Reply("🔊 Generating Hindi speech...")
	q := url.QueryEscape(text)
	endpoint := "https://translate.google.com/translate_tts?ie=UTF-8&client=tw-ob&tl=hi&q=" + q
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		if status != nil {
			status.Edit("❌ TTS request failed.")
		}
		return tg.ErrEndGroup
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		if status != nil {
			status.Edit("❌ TTS service unavailable.")
		}
		return tg.ErrEndGroup
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if status != nil {
			status.Edit("❌ TTS service returned an error.")
		}
		return tg.ErrEndGroup
	}
	path := filepath.Join("downloads", fmt.Sprintf("tts_%d.mp3", time.Now().UnixNano()))
	_ = os.MkdirAll("downloads", 0o755)
	f, err := os.Create(path)
	if err != nil {
		if status != nil {
			status.Edit("❌ Could not create audio file.")
		}
		return tg.ErrEndGroup
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(path)
		if status != nil {
			status.Edit("❌ Could not save audio.")
		}
		return tg.ErrEndGroup
	}
	defer os.Remove(path)
	if err := botSendAudio(m.ChannelID(), path, ""); err != nil {
		if status != nil {
			status.Edit("❌ Could not send audio: " + err.Error())
		}
		return tg.ErrEndGroup
	}
	if status != nil {
		status.Delete()
	}
	return tg.ErrEndGroup
}

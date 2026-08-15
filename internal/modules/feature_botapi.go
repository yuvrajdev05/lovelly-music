package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"main/internal/config"
)

type botAPIResponse[T any] struct {
	OK          bool   `json:"ok"`
	Result      T      `json:"result"`
	Description string `json:"description"`
}

type diceResult struct {
	Dice struct {
		Value int `json:"value"`
	} `json:"dice"`
}

func botAPI(method string, form url.Values, out any) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/%s", config.Token, method)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	var raw struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("telegram api invalid response: %w", err)
	}
	if !raw.OK {
		return fmt.Errorf("telegram api: %s", raw.Description)
	}
	if out != nil && len(raw.Result) > 0 {
		return json.Unmarshal(raw.Result, out)
	}
	return nil
}

func botAPIInt(method string, chatID, userID int64, until int64) error {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	form.Set("user_id", strconv.FormatInt(userID, 10))
	if until > 0 {
		form.Set("until_date", strconv.FormatInt(until, 10))
	}
	return botAPI(method, form, nil)
}

func botDelete(chatID int64, messageID int32) error {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	form.Set("message_id", strconv.FormatInt(int64(messageID), 10))
	return botAPI("deleteMessage", form, nil)
}

func botPin(chatID int64, messageID int32, notify bool) error {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	form.Set("message_id", strconv.FormatInt(int64(messageID), 10))
	form.Set("disable_notification", strconv.FormatBool(!notify))
	return botAPI("pinChatMessage", form, nil)
}

func botUnpin(chatID int64, messageID int32) error {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	if messageID > 0 {
		form.Set("message_id", strconv.FormatInt(int64(messageID), 10))
	}
	return botAPI("unpinChatMessage", form, nil)
}

func botUnpinAll(chatID int64) error {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	return botAPI("unpinAllChatMessages", form, nil)
}

func botDice(chatID int64, emoji string) (int, error) {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	form.Set("emoji", emoji)
	var result struct {
		Message struct {
			Dice struct {
				Value int `json:"value"`
			} `json:"dice"`
		} `json:"message"`
	}
	if err := botAPI("sendDice", form, &result); err != nil {
		return 0, err
	}
	return result.Message.Dice.Value, nil
}

func botSendAudio(chatID int64, path string, caption string) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	if caption != "" {
		_ = mw.WriteField("caption", caption)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	part, err := mw.CreateFormFile("audio", filepathBase(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendAudio", config.Token)
	req, err := http.NewRequest(http.MethodPost, endpoint, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	var raw struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	if !raw.OK {
		return fmt.Errorf("telegram api: %s", raw.Description)
	}
	return nil
}

func filepathBase(path string) string {
	if i := strings.LastIndexAny(path, "/\\"); i >= 0 {
		return path[i+1:]
	}
	return path
}

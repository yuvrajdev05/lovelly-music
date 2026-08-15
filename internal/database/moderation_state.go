package database

import (
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ModerationState keeps persistent global moderation lists and per-chat warnings.
// It is stored in the existing bot_settings collection so no new Mongo setup is required.
type ModerationState struct {
	ID               string         `bson:"_id"`
	BlockedUsers     []int64        `bson:"blocked_users"`
	BlacklistedChats []int64        `bson:"blacklisted_chats"`
	Warns            map[string]int `bson:"warns"`
}

func getModerationState() (*ModerationState, error) {
	const key = "moderation"
	ctx, cancel := ctx()
	defer cancel()
	var s ModerationState
	err := settingsColl.FindOne(ctx, bson.M{"_id": key}).Decode(&s)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &ModerationState{ID: key, BlockedUsers: []int64{}, BlacklistedChats: []int64{}, Warns: map[string]int{}}, nil
		}
		return nil, fmt.Errorf("failed to get moderation state: %w", err)
	}
	if s.BlockedUsers == nil {
		s.BlockedUsers = []int64{}
	}
	if s.BlacklistedChats == nil {
		s.BlacklistedChats = []int64{}
	}
	if s.Warns == nil {
		s.Warns = map[string]int{}
	}
	return &s, nil
}

func saveModerationState(s *ModerationState) error {
	ctx, cancel := ctx()
	defer cancel()
	_, err := settingsColl.UpdateOne(ctx, bson.M{"_id": s.ID}, bson.M{"$set": s}, upsertOpt)
	if err != nil {
		return fmt.Errorf("failed to save moderation state: %w", err)
	}
	return nil
}

func AddBlockedUser(id int64) error {
	s, err := getModerationState()
	if err != nil {
		return err
	}
	s.BlockedUsers, _ = addUnique(s.BlockedUsers, id)
	return saveModerationState(s)
}
func RemoveBlockedUser(id int64) error {
	s, err := getModerationState()
	if err != nil {
		return err
	}
	s.BlockedUsers, _ = removeElement(s.BlockedUsers, id)
	return saveModerationState(s)
}
func BlockedUsers() ([]int64, error) {
	s, err := getModerationState()
	if err != nil {
		return nil, err
	}
	return append([]int64(nil), s.BlockedUsers...), nil
}
func IsBlockedUser(id int64) (bool, error) {
	s, err := getModerationState()
	if err != nil {
		return false, err
	}
	return contains(s.BlockedUsers, id), nil
}

func AddBlacklistedChat(id int64) error {
	s, err := getModerationState()
	if err != nil {
		return err
	}
	s.BlacklistedChats, _ = addUnique(s.BlacklistedChats, id)
	return saveModerationState(s)
}
func RemoveBlacklistedChat(id int64) error {
	s, err := getModerationState()
	if err != nil {
		return err
	}
	s.BlacklistedChats, _ = removeElement(s.BlacklistedChats, id)
	return saveModerationState(s)
}
func BlacklistedChats() ([]int64, error) {
	s, err := getModerationState()
	if err != nil {
		return nil, err
	}
	return append([]int64(nil), s.BlacklistedChats...), nil
}
func IsBlacklistedChat(id int64) (bool, error) {
	s, err := getModerationState()
	if err != nil {
		return false, err
	}
	return contains(s.BlacklistedChats, id), nil
}

func GetWarn(chatID, userID int64) (int, error) {
	s, err := getModerationState()
	if err != nil {
		return 0, err
	}
	return s.Warns[fmt.Sprintf("%d:%d", chatID, userID)], nil
}
func SetWarn(chatID, userID int64, count int) error {
	s, err := getModerationState()
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%d:%d", chatID, userID)
	if count <= 0 {
		delete(s.Warns, key)
	} else {
		s.Warns[key] = count
	}
	return saveModerationState(s)
}

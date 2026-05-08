package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/yup/server/internal/model"
)

type Store struct {
	mu              sync.RWMutex
	users           map[string]*model.User
	devices         map[string]*model.Device         // deviceID -> device
	keyBundles      map[string]*model.KeyBundle       // username -> latest bundle
	messages        map[string]*model.Message         // messageID -> message
	pendingEnvelopes map[string][]string              // username -> pending messageIDs
	sentMessages    map[string][]string               // username -> sent messageIDs
}

func NewStore() *Store {
	return &Store{
		users:            make(map[string]*model.User),
		devices:          make(map[string]*model.Device),
		keyBundles:       make(map[string]*model.KeyBundle),
		messages:         make(map[string]*model.Message),
		pendingEnvelopes: make(map[string][]string),
		sentMessages:     make(map[string][]string),
	}
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func newToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) RegisterUser(username string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		return nil, fmt.Errorf("username already exists")
	}

	user := &model.User{
		Username:  username,
		AuthToken: newToken(),
		CreatedAt: time.Now().UTC(),
	}
	s.users[username] = user
	return user, nil
}

func (s *Store) GetUser(username string) (*model.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[username]
	return user, ok
}

func (s *Store) ValidateToken(username, token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[username]
	if !ok {
		return false
	}
	return user.AuthToken == token
}

func (s *Store) UploadKeyBundle(username string, bundle *model.KeyBundle) (*model.KeyBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; !exists {
		return nil, fmt.Errorf("user not found")
	}

	bundle.DeviceID = newID()
	s.keyBundles[username] = bundle

	device := &model.Device{
		DeviceID:       bundle.DeviceID,
		UserID:         username,
		PublicCurveKey: bundle.CurveKey,
		PublicEdKey:    bundle.EdKey,
		CreatedAt:      time.Now().UTC(),
	}
	s.devices[bundle.DeviceID] = device

	return bundle, nil
}

func (s *Store) GetKeyBundle(username string) (*model.KeyBundle, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bundle, ok := s.keyBundles[username]
	return bundle, ok
}

func (s *Store) StoreMessage(sender, recipient, ciphertext string, msgType int, senderKey string) (*model.Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[recipient]; !exists {
		return nil, fmt.Errorf("recipient not found")
	}

	msgID := newID()
	now := time.Now().UTC()

	msg := &model.Message{
		ID:             msgID,
		SenderUsername: sender,
		RecipientUser:  recipient,
		Ciphertext:     ciphertext,
		MessageType:    msgType,
		SenderCurveKey: senderKey,
		Status:         "pending",
		CreatedAt:      now,
	}
	s.messages[msgID] = msg

	s.pendingEnvelopes[recipient] = append(s.pendingEnvelopes[recipient], msgID)
	s.sentMessages[sender] = append(s.sentMessages[sender], msgID)

	return &model.Envelope{
		ID:             msgID,
		SenderUsername: sender,
		Ciphertext:     ciphertext,
		MessageType:    msgType,
		SenderCurveKey: senderKey,
		Status:         "pending",
		CreatedAt:      now,
	}, nil
}

func (s *Store) GetPendingEnvelopes(username string) []*model.Envelope {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := s.pendingEnvelopes[username]
	delete(s.pendingEnvelopes, username)

	envs := make([]*model.Envelope, 0, len(ids))
	now := time.Now().UTC()
	for _, id := range ids {
		msg, ok := s.messages[id]
		if !ok {
			continue
		}
		msg.Status = "delivered"
		msg.DeliveredAt = &now
		envs = append(envs, &model.Envelope{
			ID:             msg.ID,
			SenderUsername: msg.SenderUsername,
			Ciphertext:     msg.Ciphertext,
			MessageType:    msg.MessageType,
			SenderCurveKey: msg.SenderCurveKey,
			Status:         "delivered",
			CreatedAt:      msg.CreatedAt,
		})
	}
	return envs
}

func (s *Store) AckMessage(messageID, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg, ok := s.messages[messageID]
	if !ok {
		return fmt.Errorf("message not found")
	}
	if msg.RecipientUser != username {
		return fmt.Errorf("not the recipient of this message")
	}
	if msg.Status != "delivered" && msg.Status != "pending" {
		return fmt.Errorf("message already acknowledged")
	}
	now := time.Now().UTC()
	msg.Status = "received"
	msg.DeliveredAt = &now
	return nil
}

func (s *Store) GetSentMessages(username string) []*model.Envelope {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.sentMessages[username]
	envs := make([]*model.Envelope, 0, len(ids))
	for _, id := range ids {
		msg, ok := s.messages[id]
		if !ok {
			continue
		}
		envs = append(envs, &model.Envelope{
			ID:             msg.ID,
			SenderUsername: msg.SenderUsername,
			Ciphertext:     msg.Ciphertext,
			MessageType:    msg.MessageType,
			SenderCurveKey: msg.SenderCurveKey,
			Status:         msg.Status,
			CreatedAt:      msg.CreatedAt,
		})
	}
	return envs
}

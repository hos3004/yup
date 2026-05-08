package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/yup/server/internal/model"
)

type Store struct {
	mu               sync.RWMutex
	users            map[string]*model.User
	tokens           map[string]string            // token -> username
	devices          map[string]*model.Device      // deviceID -> device
	keyBundles       map[string]*model.KeyBundle   // username -> latest bundle
	consumedOtk      map[string]map[string]bool    // username -> consumed OTK set
	messages         map[string]*model.Message     // messageID -> message
	pendingEnvelopes map[string][]string           // username -> pending messageIDs
	sentMessages     map[string][]string           // username -> sent messageIDs
}

func NewStore() *Store {
	return &Store{
		users:            make(map[string]*model.User),
		tokens:           make(map[string]string),
		devices:          make(map[string]*model.Device),
		keyBundles:       make(map[string]*model.KeyBundle),
		consumedOtk:      make(map[string]map[string]bool),
		messages:         make(map[string]*model.Message),
		pendingEnvelopes: make(map[string][]string),
		sentMessages:     make(map[string][]string),
	}
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (s *Store) RegisterUser(username string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		return nil, fmt.Errorf("username already exists")
	}

	token, err := newToken()
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:  username,
		AuthToken: token,
		CreatedAt: time.Now().UTC(),
	}
	s.users[username] = user
	s.tokens[token] = username
	return user, nil
}

func (s *Store) GetUser(username string) (*model.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[username]
	return user, ok
}

// ValidateToken looks up the user by token and returns the username.
// Uses constant-time comparison for token verification.
func (s *Store) ValidateToken(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	username, ok := s.tokens[token]
	if !ok {
		return "", false
	}
	user, ok := s.users[username]
	if !ok {
		return "", false
	}
	// Constant-time comparison
	if subtle.ConstantTimeCompare([]byte(token), []byte(user.AuthToken)) == 1 {
		return username, true
	}
	return "", false
}

func (s *Store) UploadKeyBundle(username string, bundle *model.KeyBundle) (*model.KeyBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; !exists {
		return nil, fmt.Errorf("user not found")
	}

	deviceID, err := newID()
	if err != nil {
		return nil, err
	}
	bundle.DeviceID = deviceID
	s.keyBundles[username] = bundle
	s.consumedOtk[username] = make(map[string]bool)

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

func (s *Store) GetKeyBundle(username string) (*model.KeyBundle, bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bundle, ok := s.keyBundles[username]
	if !ok {
		return nil, false, ""
	}

	// Consume exactly one available OTK
	var chosenOTK string
	consumed := s.consumedOtk[username]
	if consumed == nil {
		consumed = make(map[string]bool)
		s.consumedOtk[username] = consumed
	}

	for _, otk := range bundle.OneTimeKeys {
		if !consumed[otk] {
			chosenOTK = otk
			consumed[otk] = true
			break
		}
	}

	responseBundle := &model.KeyBundle{
		DeviceID:    bundle.DeviceID,
		CurveKey:    bundle.CurveKey,
		EdKey:       bundle.EdKey,
		OneTimeKeys: []string{},
		Signature:   bundle.Signature,
	}

	remaining := ""
	if chosenOTK != "" {
		responseBundle.OneTimeKeys = []string{chosenOTK}
	} else {
		remaining = "no_otk_available"
	}

	return responseBundle, true, remaining
}

func (s *Store) AvailableOTKCount(username string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bundle, ok := s.keyBundles[username]
	if !ok {
		return 0
	}
	consumed := s.consumedOtk[username]
	count := 0
	for _, otk := range bundle.OneTimeKeys {
		if !consumed[otk] {
			count++
		}
	}
	return count
}

func (s *Store) StoreMessage(sender, recipient, ciphertext string, msgType int, senderKey string) (*model.Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[recipient]; !exists {
		return nil, fmt.Errorf("recipient not found")
	}

	msgID, err := newID()
	if err != nil {
		return nil, err
	}
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

func (s *Store) DeleteAllUserData(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; !exists {
		return fmt.Errorf("user not found")
	}

	token := s.users[username].AuthToken
	delete(s.tokens, token)
	delete(s.users, username)
	delete(s.keyBundles, username)
	delete(s.consumedOtk, username)
	delete(s.pendingEnvelopes, username)
	delete(s.sentMessages, username)

	return nil
}

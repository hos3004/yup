package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/yup/server/internal/model"
)

// DataStore defines the persistence interface for the YUP server.
type DataStore interface {
	RegisterUser(username string) (*model.User, error)
	GetUser(username string) (*model.User, bool)
	ValidateToken(token string) (string, bool)
	UploadKeyBundle(username string, bundle *model.KeyBundle) (*model.KeyBundle, error)
	GetKeyBundle(username string) (*model.KeyBundle, bool, string)
	GetCurveKey(username string) (string, bool)
	AvailableOTKCount(username string) int
	StoreMessage(sender, recipient, ciphertext string, msgType int, senderKey string) (*model.Envelope, error)
	GetPendingEnvelopes(username string) []*model.Envelope
	AckMessage(messageID, username string) error
	GetSentMessages(username string) []*model.Envelope
	DeleteAllUserData(username string) error
	PurgeExpiredMessages(maxAge time.Duration) error
	RegisterDeviceToken(username, token, platform string) error
	GetDeviceTokens(username string) ([]string, error)
}

type InMemoryStore struct {
	mu               sync.RWMutex
	users            map[string]*model.User
	tokenHashes      map[string]string            // sha256(token) -> username
	devices          map[string]*model.Device      // deviceID -> device
	keyBundles       map[string]*model.KeyBundle   // username -> latest bundle
	consumedOtk      map[string]map[string]bool    // username -> consumed OTK set
	messages         map[string]*model.Message     // messageID -> message
	pendingEnvelopes map[string][]string           // username -> pending messageIDs
	sentMessages     map[string][]string           // username -> sent messageIDs
	deviceTokens     map[string]map[string]*model.DeviceToken  // username -> token -> DeviceToken
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		users:            make(map[string]*model.User),
		tokenHashes:      make(map[string]string),
		devices:          make(map[string]*model.Device),
		keyBundles:       make(map[string]*model.KeyBundle),
		consumedOtk:      make(map[string]map[string]bool),
		messages:         make(map[string]*model.Message),
		pendingEnvelopes: make(map[string][]string),
		sentMessages:     make(map[string][]string),
		deviceTokens:     make(map[string]map[string]*model.DeviceToken),
	}
}

func (s *InMemoryStore) RegisterUser(username string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		return nil, fmt.Errorf("username already exists")
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	tokenHash := sha256Hex(token)
	user := &model.User{
		Username:  username,
		AuthToken: token,
		CreatedAt: time.Now().UTC(),
	}
	s.users[username] = user
	s.tokenHashes[tokenHash] = username
	return user, nil
}

func (s *InMemoryStore) GetUser(username string) (*model.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[username]
	return user, ok
}

// ValidateToken looks up the user by token hash and returns the username.
func (s *InMemoryStore) ValidateToken(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tokenHash := sha256Hex(token)
	username, ok := s.tokenHashes[tokenHash]
	if !ok {
		return "", false
	}
	return username, true
}

func (s *InMemoryStore) UploadKeyBundle(username string, bundle *model.KeyBundle) (*model.KeyBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; !exists {
		return nil, fmt.Errorf("user not found")
	}

	deviceID, err := generateID()
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

func (s *InMemoryStore) GetKeyBundle(username string) (*model.KeyBundle, bool, string) {
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

func (s *InMemoryStore) GetCurveKey(username string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bundle, ok := s.keyBundles[username]
	if !ok {
		return "", false
	}
	return bundle.CurveKey, true
}

func (s *InMemoryStore) AvailableOTKCount(username string) int {
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

func (s *InMemoryStore) StoreMessage(sender, recipient, ciphertext string, msgType int, senderKey string) (*model.Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[recipient]; !exists {
		return nil, fmt.Errorf("recipient not found")
	}

	msgID, err := generateID()
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

func (s *InMemoryStore) GetPendingEnvelopes(username string) []*model.Envelope {
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

func (s *InMemoryStore) AckMessage(messageID, username string) error {
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

func (s *InMemoryStore) GetSentMessages(username string) []*model.Envelope {
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

func (s *InMemoryStore) PurgeExpiredMessages(maxAge time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().UTC().Add(-maxAge)
	for id, msg := range s.messages {
		if msg.CreatedAt.Before(cutoff) {
			delete(s.messages, id)
		}
	}
	return nil
}

func (s *InMemoryStore) DeleteAllUserData(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; !exists {
		return fmt.Errorf("user not found")
	}

	tokenHash := sha256Hex(s.users[username].AuthToken)
	delete(s.tokenHashes, tokenHash)
	delete(s.users, username)
	delete(s.keyBundles, username)
	delete(s.consumedOtk, username)
	delete(s.pendingEnvelopes, username)
	delete(s.sentMessages, username)
	delete(s.deviceTokens, username)

	return nil
}

func (s *InMemoryStore) RegisterDeviceToken(username, token, platform string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; !exists {
		return fmt.Errorf("user not found")
	}

	if s.deviceTokens[username] == nil {
		s.deviceTokens[username] = make(map[string]*model.DeviceToken)
	}

	now := time.Now().UTC()
	if existing, ok := s.deviceTokens[username][token]; ok {
		existing.Platform = platform
		existing.UpdatedAt = now
	} else {
		s.deviceTokens[username][token] = &model.DeviceToken{
			Username:  username,
			Token:     token,
			Platform:  platform,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	return nil
}

func (s *InMemoryStore) GetDeviceTokens(username string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokens := s.deviceTokens[username]
	result := make([]string, 0, len(tokens))
	for token := range tokens {
		result = append(result, token)
	}
	return result, nil
}

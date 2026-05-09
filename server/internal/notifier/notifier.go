package notifier

import (
	"context"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// Notifier sends push notifications to device tokens.
type Notifier interface {
	// SendPush sends a data-only push notification to the given tokens.
	// Returns the number of successfully sent notifications.
	SendPush(ctx context.Context, tokens []string, data map[string]string) (int, error)
}

type noopNotifier struct{}

func (n *noopNotifier) SendPush(_ context.Context, tokens []string, data map[string]string) (int, error) {
	return 0, nil
}

type fcmNotifier struct {
	client *messaging.Client
}

func (n *fcmNotifier) SendPush(ctx context.Context, tokens []string, data map[string]string) (int, error) {
	if len(tokens) == 0 {
		return 0, nil
	}

	msg := &messaging.MulticastMessage{
		Data: data,
		Tokens: tokens,
	}

	br, err := n.client.SendEachForMulticast(ctx, msg)
	if err != nil {
		return 0, err
	}

	return br.SuccessCount, nil
}

// New creates a Notifier backed by Firebase Cloud Messaging if
// GOOGLE_APPLICATION_CREDENTIALS is set, otherwise a no-op notifier.
func New() Notifier {
	creds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if creds == "" {
		log.Println("FCM notifier: GOOGLE_APPLICATION_CREDENTIALS not set, using no-op notifier")
		return &noopNotifier{}
	}

	opt := option.WithCredentialsFile(creds)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Printf("FCM notifier: failed to initialize Firebase app: %v, using no-op notifier", err)
		return &noopNotifier{}
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Printf("FCM notifier: failed to create messaging client: %v, using no-op notifier", err)
		return &noopNotifier{}
	}

	log.Println("FCM notifier: initialized successfully")
	return &fcmNotifier{client: client}
}

package push

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"

	"repwire/internal/domain"
)

type Client struct {
	publicKey  string
	privateKey string
	subject    string
}

func NewClient(publicKey, privateKey, subject string) *Client {
	return &Client{publicKey: publicKey, privateKey: privateKey, subject: subject}
}

func (c *Client) Enabled() bool { return c.publicKey != "" && c.privateKey != "" }

func (c *Client) Send(ctx context.Context, sub domain.PushSubscription, title, body, targetURL string) error {
	if !c.Enabled() {
		return fmt.Errorf("web push not configured")
	}
	payload, _ := json.Marshal(map[string]string{"title": title, "body": body, "url": targetURL})
	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.P256DH, Auth: sub.Auth},
	}, &webpush.Options{
		Subscriber: c.subject, VAPIDPublicKey: c.publicKey, VAPIDPrivateKey: c.privateKey, TTL: 300,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("web push http %d", resp.StatusCode)
	}
	return nil
}

package main

// walletconnect_relay.go — transport to a WalletConnect v2 relay.
//
// The relay is a message broker. A wallet publishes to a topic, subscribes to
// that topic, and receives whatever other participants published. The relay
// sees only sealed envelopes; it cannot read the JSON-RPC payloads inside.
//
// The protocol is: open a WebSocket, receive a `relay_attestation` frame, send
// a `relay_subscribe` request, then exchange `relay_publish` and `relay_irn`
// frames. Everything published is sealed with the topic's symmetric key.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// relayTopic is the publish/subscribe address derived from a symmetric key.
type relayTopic struct {
	Topic string
	Key   []byte
}

// RelayClient owns one WebSocket to a relay and multiplexes subscriptions.
type RelayClient struct {
	url   string
	conn  *websocket.Conn
	write sync.Mutex

	mu     sync.RWMutex
	topics map[string]*relayTopic

	// handler is called with the plaintext of every message received on a
	// subscribed topic. It runs on the read goroutine, so it must not block.
	handler func(topic string, payload []byte)

	closed chan struct{}
	once   sync.Once
}

// relayFrame is the envelope every relay WebSocket message uses.
type relayFrame struct {
	ID      int64           `json:"id,omitempty"`
	JSONRPC string          `json:"jsonrpc,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// RelayPublishParams is the payload of relay_publish.
type RelayPublishParams struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Tag     int    `json:"tag,omitempty"`
}

// RelaySubscribeParams is the payload of relay_subscribe.
type RelaySubscribeParams struct {
	Topic string `json:"topic"`
}

// NewRelayClient dials the relay and completes the subscribe handshake.
//
// A WalletConnect relay requires a project id, and its WebSocket endpoint
// carries that id in the query string. An empty project id is a configuration
// error: dialling without one yields a socket the relay rejects, so it is
// refused up front rather than surfacing later as a silent no-op.
func NewRelayClient(ctx context.Context, relayURL, projectID string, handler func(string, []byte)) (*RelayClient, error) {
	if strings.TrimSpace(relayURL) == "" {
		return nil, errors.New("relay URL is required")
	}
	if strings.TrimSpace(projectID) == "" {
		return nil, errors.New("WalletConnect project id is required to reach the relay")
	}

	endpoint := relayURL
	if !strings.Contains(endpoint, "projectId=") {
		sep := "?"
		if strings.Contains(endpoint, "?") {
			sep = "&"
		}
		endpoint += sep + "projectId=" + projectID
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}
	conn, resp, err := dialer.DialContext(ctx, endpoint, nil)
	if err != nil {
		status := ""
		if resp != nil {
			status = resp.Status
		}
		return nil, fmt.Errorf("dial relay %s: %w %s", endpoint, err, status)
	}

	c := &RelayClient{
		url:     endpoint,
		conn:    conn,
		topics:  make(map[string]*relayTopic),
		handler: handler,
		closed:  make(chan struct{}),
	}

	// The relay greets every socket with a relay_attestation frame carrying the
	// token that authorises subscriptions.
	if err := c.awaitAttestation(); err != nil {
		conn.Close()
		return nil, err
	}

	go c.readLoop()
	return c, nil
}

// awaitAttestation reads the opening frame and verifies the relay's token.
func (c *RelayClient) awaitAttestation() error {
	c.conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	defer c.conn.SetReadDeadline(time.Time{})

	_, raw, err := c.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read relay attestation: %w", err)
	}

	var frame relayFrame
	if err := json.Unmarshal(raw, &frame); err != nil {
		return fmt.Errorf("relay attestation is not JSON: %w", err)
	}
	if frame.Method != "relay_attestation" {
		return fmt.Errorf("relay sent %q before attesting", frame.Method)
	}

	var params struct {
		Attestation string `json:"attestation"`
	}
	if err := json.Unmarshal(frame.Params, &params); err != nil {
		return fmt.Errorf("relay attestation has no payload: %w", err)
	}

	if _, err := VerifyRelayAuthJWT(params.Attestation, "", time.Now().Unix()); err != nil {
		return fmt.Errorf("relay attestation rejected: %w", err)
	}
	return nil
}

// Subscribe registers interest in a topic and its symmetric key.
func (c *RelayClient) Subscribe(ctx context.Context, topic string, key []byte) error {
	c.mu.Lock()
	c.topics[topic] = &relayTopic{Topic: topic, Key: key}
	c.mu.Unlock()

	params, err := json.Marshal(RelaySubscribeParams{Topic: topic})
	if err != nil {
		return err
	}
	return c.writeFrame(ctx, relayFrame{
		ID:      time.Now().UnixNano(),
		JSONRPC: "2.0",
		Method:  "relay_subscribe",
		Params:  params,
	})
}

// Unsubscribe drops interest in a topic.
func (c *RelayClient) Unsubscribe(ctx context.Context, topic string) error {
	c.mu.Lock()
	delete(c.topics, topic)
	c.mu.Unlock()

	params, err := json.Marshal(RelaySubscribeParams{Topic: topic})
	if err != nil {
		return err
	}
	return c.writeFrame(ctx, relayFrame{
		ID:      time.Now().UnixNano(),
		JSONRPC: "2.0",
		Method:  "relay_unsubscribe",
		Params:  params,
	})
}

// Publish seals payload with the topic's key and hands it to the relay.
func (c *RelayClient) Publish(ctx context.Context, topic string, payload []byte) error {
	c.mu.RLock()
	t, ok := c.topics[topic]
	c.mu.RUnlock()
	if !ok {
		return fmt.Errorf("not subscribed to topic %s", topic)
	}

	envelope, err := SealEnvelope(t.Key, payload, envelopeTypeMessage)
	if err != nil {
		return err
	}
	params, err := json.Marshal(RelayPublishParams{Topic: topic, Message: envelope, TTL: 300})
	if err != nil {
		return err
	}
	return c.writeFrame(ctx, relayFrame{
		ID:      time.Now().UnixNano(),
		JSONRPC: "2.0",
		Method:  "relay_publish",
		Params:  params,
	})
}

// writeFrame serialises one frame onto the socket. A write mutex is required
// because gorilla/websocket permits only one concurrent writer.
func (c *RelayClient) writeFrame(ctx context.Context, frame relayFrame) error {
	raw, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	c.write.Lock()
	defer c.write.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, raw)
}

// readLoop dispatches incoming frames until the socket closes.
func (c *RelayClient) readLoop() {
	defer c.Close()
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			select {
			case <-c.closed:
			default:
				log.Printf("walletconnect relay read ended: %v", err)
			}
			return
		}

		var frame relayFrame
		if err := json.Unmarshal(raw, &frame); err != nil {
			continue
		}

		switch frame.Method {
		case "relay_irn", "relay_publish":
			c.deliver(frame.Params)
		case "relay_subscribe", "relay_unsubscribe":
			if frame.Error != nil {
				log.Printf("walletconnect relay %s failed: %s", frame.Method, frame.Error.Message)
			}
		}
	}
}

// deliver opens a published envelope and passes the plaintext to the handler.
func (c *RelayClient) deliver(params json.RawMessage) {
	var p struct {
		Topic   string `json:"topic"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.Topic == "" {
		return
	}

	c.mu.RLock()
	t, ok := c.topics[p.Topic]
	c.mu.RUnlock()
	if !ok {
		// Not our topic, or it was unsubscribed between the publish and now.
		return
	}

	plaintext, msgType, err := OpenEnvelope(t.Key, p.Message)
	if err != nil {
		// A relay that tampers with an envelope produces this; drop it rather
		// than acting on unauthenticated bytes.
		log.Printf("walletconnect envelope rejected on topic %s: %v", p.Topic, err)
		return
	}
	if msgType != envelopeTypeMessage {
		return
	}
	if c.handler != nil {
		c.handler(p.Topic, plaintext)
	}
}

// Close shuts the socket down exactly once.
func (c *RelayClient) Close() {
	c.once.Do(func() {
		close(c.closed)
		c.conn.Close()
	})
}

// Connected reports whether the socket is still open.
func (c *RelayClient) Connected() bool {
	select {
	case <-c.closed:
		return false
	default:
		return true
	}
}

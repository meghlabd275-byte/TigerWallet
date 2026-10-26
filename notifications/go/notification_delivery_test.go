package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestDeliverToDevicePostsRealPayload exercises the delivery code path against a
// real HTTP server and asserts the request the push service receives.
func TestDeliverToDevicePostsRealPayload(t *testing.T) {
	type captured struct {
		path string
		body map[string]interface{}
	}
	got := make(chan captured, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Errorf("push service received invalid JSON: %v", err)
		}
		got <- captured{path: r.URL.Path, body: payload}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"sent"}`))
	}))
	defer server.Close()

	svc := &NotificationService{
		config:     NotificationConfig{PushServiceURL: server.URL + "/"},
		pushClient: server.Client(),
	}

	device := DeviceToken{
		UserAddress: "0xabc",
		Token:       "device-token-1",
		Platform:    "android",
	}

	if err := svc.deliverToDevice(device, "Price alert", "BTC is up 5%"); err != nil {
		t.Fatalf("deliverToDevice: %v", err)
	}

	select {
	case c := <-got:
		if c.path != "/push" {
			t.Fatalf("expected path /push, got %q", c.path)
		}
		for key, want := range map[string]string{
			"user_id":  "0xabc",
			"token":    "device-token-1",
			"platform": "android",
			"title":    "Price alert",
			"body":     "BTC is up 5%",
		} {
			if c.body[key] != want {
				t.Errorf("payload[%q] = %v, want %q", key, c.body[key], want)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("push service was never called")
	}
}

// TestDeliverToDeviceSurfacesProviderError proves an upstream failure is
// reported instead of being swallowed.
func TestDeliverToDeviceSurfacesProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid token"}`))
	}))
	defer server.Close()

	svc := &NotificationService{
		config:     NotificationConfig{PushServiceURL: server.URL},
		pushClient: server.Client(),
	}

	err := svc.deliverToDevice(DeviceToken{Token: "bad"}, "t", "m")
	if err == nil {
		t.Fatal("expected an error for a 401 from the push service")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("error should carry the upstream status, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("error should carry the upstream body, got %v", err)
	}
}

// TestDeliverToDeviceFailsClosedWithoutURL ensures an unconfigured push
// service cannot report a successful delivery.
func TestDeliverToDeviceFailsClosedWithoutURL(t *testing.T) {
	svc := &NotificationService{
		config:     NotificationConfig{PushServiceURL: ""},
		pushClient: &http.Client{Timeout: time.Second},
	}

	err := svc.deliverToDevice(DeviceToken{Token: "x"}, "t", "m")
	if err == nil {
		t.Fatal("expected an error when the push service URL is empty")
	}
}

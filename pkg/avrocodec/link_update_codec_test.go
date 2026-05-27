package avrocodec

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

func TestLinkUpdateCodecSerializeDeserialize(t *testing.T) {
	const expectedSchemaID = 17

	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/subjects/link-updates-value/versions" {
			t.Fatalf("unexpected schema registry path: %s", r.URL.Path)
		}

		var request registerSchemaRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode schema registry request: %v", err)
		}

		if request.Schema == "" {
			t.Fatal("expected non-empty avro schema")
		}

		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(registerSchemaResponse{
			ID: expectedSchemaID,
		}); err != nil {
			t.Fatalf("encode schema registry response: %v", err)
		}
	}))
	defer registry.Close()

	codec, err := NewLinkUpdateCodec(
		context.Background(),
		registry.URL,
		"link-updates-value",
	)
	if err != nil {
		t.Fatalf("create link update codec: %v", err)
	}

	expected := api.LinkUpdate{
		ID:          10,
		URL:         "https://github.com/golang/go",
		TgChatIDs:   []int64{1, 2},
		Type:        "github_issue",
		Title:       "Test issue",
		Username:    "tester",
		CreatedAt:   time.Date(2026, 5, 13, 18, 0, 0, 0, time.UTC),
		Preview:     "Preview",
		Description: "Description",
	}

	payload, err := codec.Serialize(expected)
	if err != nil {
		t.Fatalf("serialize link update: %v", err)
	}

	if len(payload) < confluentHeaderSize {
		t.Fatalf("expected payload length >= %d, got %d", confluentHeaderSize, len(payload))
	}

	if payload[0] != confluentMagicByte {
		t.Fatalf("expected magic byte %d, got %d", confluentMagicByte, payload[0])
	}

	actualSchemaID := int(binary.BigEndian.Uint32(payload[1:confluentHeaderSize]))
	if actualSchemaID != expectedSchemaID {
		t.Fatalf("expected schema id %d, got %d", expectedSchemaID, actualSchemaID)
	}

	actual, err := codec.Deserialize(payload)
	if err != nil {
		t.Fatalf("deserialize link update: %v", err)
	}

	if actual.ID != expected.ID {
		t.Fatalf("expected id %d, got %d", expected.ID, actual.ID)
	}

	if actual.URL != expected.URL {
		t.Fatalf("expected url %q, got %q", expected.URL, actual.URL)
	}

	if actual.Description != expected.Description {
		t.Fatalf("expected description %q, got %q", expected.Description, actual.Description)
	}

	if !actual.CreatedAt.Equal(expected.CreatedAt) {
		t.Fatalf("expected createdAt %s, got %s", expected.CreatedAt, actual.CreatedAt)
	}

	if len(actual.TgChatIDs) != len(expected.TgChatIDs) {
		t.Fatalf("expected tgChatIds %v, got %v", expected.TgChatIDs, actual.TgChatIDs)
	}

	for i := range expected.TgChatIDs {
		if actual.TgChatIDs[i] != expected.TgChatIDs[i] {
			t.Fatalf("expected tgChatIds %v, got %v", expected.TgChatIDs, actual.TgChatIDs)
		}
	}
}

package avrocodec

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/linkedin/goavro/v2"

	"github.com/slickip/link-tracker/pkg/api"
)

const (
	confluentMagicByte  byte = 0
	confluentHeaderSize int  = 5

	LinkUpdateEventSchema = `{
	  "type": "record",
	  "name": "LinkUpdateEvent",
	  "namespace": "ru.tbank.linktracker.notification",
	  "fields": [
	    {"name": "id", "type": "long", "doc": "ID ссылки"},
	    {"name": "url", "type": "string", "doc": "Ссылка"},
	    {"name": "tgChatIds", "type": {"type": "array", "items": "long"}, "doc": "ID чатов Telegram"},
	    {"name": "type", "type": "string", "doc": "Тип обновления"},
	    {"name": "title", "type": "string", "doc": "Заголовок обновления"},
	    {"name": "username", "type": "string", "doc": "Автор обновления"},
	    {"name": "createdAt", "type": "string", "doc": "Время создания обновления в RFC3339Nano"},
	    {"name": "preview", "type": "string", "doc": "Краткое описание"},
	    {"name": "description", "type": "string", "doc": "Полное описание уведомления"},
		{"name": "priority", "type": ["null", "string"], "default": null, "doc": "Приоритет обновления"}
	  ]
	}`
)

type LinkUpdateCodec struct {
	codec    *goavro.Codec
	schemaID int
}

func NewLinkUpdateCodec(
	ctx context.Context,
	schemaRegistryURL string,
	subject string,
) (*LinkUpdateCodec, error) {
	codec, err := goavro.NewCodec(LinkUpdateEventSchema)
	if err != nil {
		return nil, err
	}

	registryClient := NewSchemaRegistryClient(schemaRegistryURL)

	schemaID, err := registryClient.Register(ctx, subject, LinkUpdateEventSchema)
	if err != nil {
		return nil, err
	}

	return &LinkUpdateCodec{
		codec:    codec,
		schemaID: schemaID,
	}, nil
}

func (c *LinkUpdateCodec) Serialize(update api.LinkUpdate) ([]byte, error) {
	native := map[string]any{
		"id":          update.ID,
		"url":         update.URL,
		"tgChatIds":   int64SliceToNative(update.TgChatIDs),
		"type":        update.Type,
		"title":       update.Title,
		"username":    update.Username,
		"createdAt":   update.CreatedAt.UTC().Format(time.RFC3339Nano),
		"preview":     update.Preview,
		"description": update.Description,
		"priority":    nullableString(update.Priority),
	}

	avroPayload, err := c.codec.BinaryFromNative(nil, native)
	if err != nil {
		return nil, err
	}

	result := make([]byte, confluentHeaderSize+len(avroPayload))
	result[0] = confluentMagicByte
	binary.BigEndian.PutUint32(result[1:confluentHeaderSize], uint32(c.schemaID))
	copy(result[confluentHeaderSize:], avroPayload)

	return result, nil
}

func (c *LinkUpdateCodec) Deserialize(payload []byte) (api.LinkUpdate, error) {
	if len(payload) < confluentHeaderSize {
		return api.LinkUpdate{}, fmt.Errorf("avro payload is too short")
	}

	if payload[0] != confluentMagicByte {
		return api.LinkUpdate{}, fmt.Errorf("invalid avro magic byte: %d", payload[0])
	}

	native, _, err := c.codec.NativeFromBinary(payload[confluentHeaderSize:])
	if err != nil {
		return api.LinkUpdate{}, err
	}

	record, ok := native.(map[string]any)
	if !ok {
		return api.LinkUpdate{}, fmt.Errorf("avro payload is not a record")
	}

	createdAtRaw, err := getStringField(record, "createdAt")
	if err != nil {
		return api.LinkUpdate{}, err
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return api.LinkUpdate{}, err
	}

	id, err := getInt64Field(record, "id")
	if err != nil {
		return api.LinkUpdate{}, err
	}

	urlValue, err := getStringField(record, "url")
	if err != nil {
		return api.LinkUpdate{}, err
	}

	tgChatIDs, err := getInt64SliceField(record, "tgChatIds")
	if err != nil {
		return api.LinkUpdate{}, err
	}

	typeValue, err := getStringField(record, "type")
	if err != nil {
		return api.LinkUpdate{}, err
	}

	title, err := getStringField(record, "title")
	if err != nil {
		return api.LinkUpdate{}, err
	}

	username, err := getStringField(record, "username")
	if err != nil {
		return api.LinkUpdate{}, err
	}

	preview, err := getStringField(record, "preview")
	if err != nil {
		return api.LinkUpdate{}, err
	}

	description, err := getStringField(record, "description")
	if err != nil {
		return api.LinkUpdate{}, err
	}

	priority, _ := getOptionalStringField(record, "priority")

	return api.LinkUpdate{
		ID:          id,
		URL:         urlValue,
		TgChatIDs:   tgChatIDs,
		Type:        typeValue,
		Title:       title,
		Username:    username,
		CreatedAt:   createdAt,
		Preview:     preview,
		Description: description,
		Priority:    priority,
	}, nil
}

func int64SliceToNative(values []int64) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}

	return result
}

func getStringField(record map[string]any, field string) (string, error) {
	value, ok := record[field]
	if !ok {
		return "", fmt.Errorf("missing avro field %s", field)
	}

	result, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("avro field %s is not string", field)
	}

	return result, nil
}

func getInt64Field(record map[string]any, field string) (int64, error) {
	value, ok := record[field]
	if !ok {
		return 0, fmt.Errorf("missing avro field %s", field)
	}

	result, ok := value.(int64)
	if !ok {
		return 0, fmt.Errorf("avro field %s is not long", field)
	}

	return result, nil
}

func getInt64SliceField(record map[string]any, field string) ([]int64, error) {
	value, ok := record[field]
	if !ok {
		return nil, fmt.Errorf("missing avro field %s", field)
	}

	rawValues, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("avro field %s is not array", field)
	}

	result := make([]int64, 0, len(rawValues))

	for _, rawValue := range rawValues {
		item, ok := rawValue.(int64)
		if !ok {
			return nil, fmt.Errorf("avro field %s contains non-long item", field)
		}

		result = append(result, item)
	}

	return result, nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}

	return map[string]any{"string": value}
}

func getOptionalStringField(record map[string]any, field string) (string, error) {
	value, ok := record[field]
	if !ok || value == nil {
		return "", nil
	}

	switch v := value.(type) {
	case string:
		return v, nil
	case map[string]any:
		if s, ok := v["string"].(string); ok {
			return s, nil
		}
	}

	return "", fmt.Errorf("avro field %s is not optional string", field)
}

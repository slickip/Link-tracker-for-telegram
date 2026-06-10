package avrocodec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const schemaRegistryContentType = "application/vnd.schemaregistry.v1+json"

type SchemaRegistryClient struct {
	baseURL string
	client  *http.Client
}

type registerSchemaRequest struct {
	Schema     string `json:"schema"`
	SchemaType string `json:"schemaType,omitempty"`
}

type registerSchemaResponse struct {
	ID int `json:"id"`
}

func NewSchemaRegistryClient(baseURL string) *SchemaRegistryClient {
	return &SchemaRegistryClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *SchemaRegistryClient) Register(
	ctx context.Context,
	subject string,
	schema string,
) (int, error) {
	if c.baseURL == "" {
		return 0, fmt.Errorf("schema registry url is empty")
	}

	if subject == "" {
		return 0, fmt.Errorf("schema subject is empty")
	}

	body, err := json.Marshal(registerSchemaRequest{
		Schema:     schema,
		SchemaType: "AVRO",
	})
	if err != nil {
		return 0, err
	}

	endpoint := fmt.Sprintf(
		"%s/subjects/%s/versions",
		c.baseURL,
		url.PathEscape(subject),
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewBuffer(body),
	)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", schemaRegistryContentType)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf(
			"schema registry returned status %d: %s",
			resp.StatusCode,
			string(responseBody),
		)
	}

	var result registerSchemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.ID, nil
}
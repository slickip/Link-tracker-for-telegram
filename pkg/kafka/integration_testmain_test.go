//go:build integration
// +build integration

package kafka

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

const testKafkaImage = "confluentinc/confluent-local:7.5.0"

var sharedKafkaBootstrap string

func TestMain(m *testing.M) {
	ctx := context.Background()

	kafkaContainer, err := tckafka.Run(
		ctx,
		testKafkaImage,
		tckafka.WithClusterID("link-tracker-integration"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kafka testcontainer: %v\n", err)
		os.Exit(1)
	}

	brokers, err := kafkaContainer.Brokers(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kafka brokers: %v\n", err)
		_ = kafkaContainer.Terminate(ctx)
		os.Exit(1)
	}

	sharedKafkaBootstrap = strings.Join(brokers, ",")

	code := m.Run()

	if err := kafkaContainer.Terminate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "kafka terminate: %v\n", err)
	}

	os.Exit(code)
}

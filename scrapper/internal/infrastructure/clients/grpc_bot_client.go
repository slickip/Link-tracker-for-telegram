package clients

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	botpb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

type GRPCBotClient struct {
	client botpb.BotServiceClient
	conn   *grpc.ClientConn
}

func NewGRPCBotClient(addr string) (*GRPCBotClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &GRPCBotClient{
		client: botpb.NewBotServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *GRPCBotClient) SendUpdate(ctx context.Context, update domain.LinkUpdate) error {
	_, err := c.client.SendUpdate(ctx, &botpb.LinkUpdateRequest{
		Url:         update.URL,
		Description: update.Description,
		TgChatIds:   update.TgChatIDs,
	})
	return err
}

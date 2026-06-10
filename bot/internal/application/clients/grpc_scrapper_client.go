package clients

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/slickip/link-tracker/bot/internal/domain"
	scrapperpb "github.com/slickip/link-tracker/pkg/api/scrapper"
)

type GRPCScrapperClient struct {
	client scrapperpb.ScrapperServiceClient
	conn   *grpc.ClientConn
}

func NewGRPCScrapperClient(addr string) (*GRPCScrapperClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &GRPCScrapperClient{
		client: scrapperpb.NewScrapperServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *GRPCScrapperClient) RegisterChat(ctx context.Context, chatID int64) error {
	_, err := c.client.RegisterChat(ctx, &scrapperpb.ChatIdRequest{
		ChatId: chatID,
	})
	return err
}

func (c *GRPCScrapperClient) DeleteChat(ctx context.Context, chatID int64) error {
	_, err := c.client.DeleteChat(ctx, &scrapperpb.ChatIdRequest{
		ChatId: chatID,
	})
	return err
}

func (c *GRPCScrapperClient) AddLink(ctx context.Context, chatID int64, url string, tags []string) error {
	_, err := c.client.AddLink(ctx, &scrapperpb.AddLinkRequest{
		ChatId: chatID,
		Url:    url,
		Tags:   tags,
	})
	return err
}

func (c *GRPCScrapperClient) RemoveLink(ctx context.Context, chatID int64, url string) error {
	_, err := c.client.RemoveLink(ctx, &scrapperpb.RemoveLinkRequest{
		ChatId: chatID,
		Url:    url,
	})
	return err
}

func (c *GRPCScrapperClient) ListLinks(ctx context.Context, chatID int64) ([]domain.Link, error) {
	resp, err := c.client.ListLinks(ctx, &scrapperpb.ListLinksRequest{
		ChatId: chatID,
	})
	if err != nil {
		return nil, err
	}

	links := make([]domain.Link, 0, len(resp.Links))

	for _, l := range resp.Links {
		links = append(links, domain.Link{
			URL:  l.Url,
			Tags: l.Tags,
		})
	}

	return links, nil
}

func (c *GRPCScrapperClient) RemoveLinksByTag(ctx context.Context, chatID int64, tag string) (int64, error) {
	resp, err := c.client.RemoveLinksByTag(ctx, &scrapperpb.RemoveLinksByTagRequest{
		ChatId: chatID,
		Tag:    tag,
	})
	if err != nil {
		return 0, err
	}

	return resp.RemovedCount, nil
}

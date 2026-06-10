package grpcserver

import (
	"context"

	botpb "github.com/slickip/link-tracker/pkg/api/bot"
	"github.com/slickip/link-tracker/pkg/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MessageSender interface {
	SendMessage(chatID int64, text string) error
}

type BotGRPCServer struct {
	bot MessageSender
	log *logger.Slog

	botpb.UnimplementedBotServiceServer
}

func NewBotGRPCServer(bot MessageSender, log *logger.Slog) *BotGRPCServer {
	return &BotGRPCServer{
		bot: bot,
		log: log,
	}
}

func (s *BotGRPCServer) SendUpdate(
	_ context.Context,
	req *botpb.LinkUpdateRequest,
) (*botpb.Empty, error) {
	if req == nil || req.GetUrl() == "" || len(req.GetTgChatIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid update request")
	}

	for _, chatID := range req.GetTgChatIds() {
		if err := s.bot.SendMessage(
			chatID,
			"Обнаружено обновление по ссылке: "+req.GetUrl(),
		); err != nil {
			if s.log != nil {
				s.log.Warn("failed to send telegram message", "chat_id", chatID, "error", err)
			}
		}
	}

	return &botpb.Empty{}, nil
}

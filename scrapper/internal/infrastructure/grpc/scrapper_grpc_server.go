package grpcserver

import (
	"context"
	"errors"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
	scrapperpb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ScrapperGRPCServer struct {
	chatService *services.ChatService
	linkService *services.LinkService
	log         *logger.Slog

	scrapperpb.UnimplementedScrapperServiceServer
}

func NewScrapperGRPCServer(
	chatService *services.ChatService,
	linkService *services.LinkService,
	log *logger.Slog,
) *ScrapperGRPCServer {
	return &ScrapperGRPCServer{
		chatService: chatService,
		linkService: linkService,
		log:         log,
	}
}

func (s *ScrapperGRPCServer) RegisterChat(
	_ context.Context,
	req *scrapperpb.ChatIdRequest,
) (*scrapperpb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}

	if err := s.chatService.RegisterChat(req.GetChatId()); err != nil {
		return nil, statusFromErr(err)
	}

	return &scrapperpb.Empty{}, nil
}

func (s *ScrapperGRPCServer) DeleteChat(
	_ context.Context,
	req *scrapperpb.ChatIdRequest,
) (*scrapperpb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}

	if err := s.chatService.DeleteChat(req.GetChatId()); err != nil {
		return nil, statusFromErr(err)
	}

	return &scrapperpb.Empty{}, nil
}

func (s *ScrapperGRPCServer) AddLink(
	_ context.Context,
	req *scrapperpb.AddLinkRequest,
) (*scrapperpb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}

	link := domain.Link{
		URL:  req.GetUrl(),
		Tags: req.GetTags(),
	}

	if err := s.linkService.AddLink(req.GetChatId(), link); err != nil {
		return nil, statusFromErr(err)
	}

	return &scrapperpb.Empty{}, nil
}

func (s *ScrapperGRPCServer) RemoveLink(
	_ context.Context,
	req *scrapperpb.RemoveLinkRequest,
) (*scrapperpb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}

	if err := s.linkService.RemoveLink(req.GetChatId(), req.GetUrl()); err != nil {
		return nil, statusFromErr(err)
	}

	return &scrapperpb.Empty{}, nil
}

func (s *ScrapperGRPCServer) ListLinks(
	_ context.Context,
	req *scrapperpb.ListLinksRequest,
) (*scrapperpb.ListLinksResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}

	links, err := s.linkService.ListLinks(req.GetChatId())
	if err != nil {
		return nil, statusFromErr(err)
	}

	pbLinks := make([]*scrapperpb.Link, 0, len(links))
	for _, l := range links {
		pbLinks = append(pbLinks, &scrapperpb.Link{
			Url:  l.URL,
			Tags: l.Tags,
		})
	}

	return &scrapperpb.ListLinksResponse{Links: pbLinks}, nil
}

func statusFromErr(err error) error {
	switch {
	case errors.Is(err, pkg.ErrChatNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, pkg.ErrLinkAlreadyTracked):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, pkg.ErrLinkNotTracked):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, pkg.ErrInvalidURL):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, pkg.ErrInvalidAPIResponse):
		return status.Error(codes.Internal, err.Error())
	default:
        return status.Error(codes.Internal, "internal server error")
    }
	}
	
	if err != nil && err.Error() == "chat not found" {
		return status.Error(codes.NotFound, err.Error())
	}
	if err == nil {
		return status.Error(codes.Internal, "nil error")
	}
	return status.Error(codes.Internal, err.Error())
}

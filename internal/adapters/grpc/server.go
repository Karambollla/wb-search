package grpc

import (
	"context"

	searchv1 "github.com/Karambollla/wb-search/api/proto/searchv1"
	"github.com/Karambollla/wb-search/internal/core"
)

type Server struct {
	searchv1.UnimplementedTopServiceServer
	searchv1.UnimplementedStopListServiceServer
	service core.TopService
}

func NewServer(service core.TopService) *Server {
	return &Server{service: service}
}

func (s *Server) GetTop(ctx context.Context, req *searchv1.GetTopRequest) (*searchv1.GetTopResponse, error) {
	items, err := s.service.GetTop(ctx, int(req.GetLimit()))
	if err != nil {
		return nil, err
	}

	resp := &searchv1.GetTopResponse{Items: make([]*searchv1.TopItem, 0, len(items))}
	for _, item := range items {
		resp.Items = append(resp.Items, &searchv1.TopItem{
			Query: item.Query,
			Count: item.Count,
		})
	}
	return resp, nil
}

func (s *Server) ListStopWords(ctx context.Context, _ *searchv1.ListStopWordsRequest) (*searchv1.ListStopWordsResponse, error) {
	items, err := s.service.ListStopWords(ctx)
	if err != nil {
		return nil, err
	}
	return &searchv1.ListStopWordsResponse{Items: items}, nil
}

func (s *Server) AddStopWord(ctx context.Context, req *searchv1.StopWordRequest) (*searchv1.StopWordResponse, error) {
	if err := s.service.AddStopWord(ctx, req.GetTerm()); err != nil {
		return nil, err
	}
	return &searchv1.StopWordResponse{Term: core.NormalizeQuery(req.GetTerm())}, nil
}

func (s *Server) DeleteStopWord(ctx context.Context, req *searchv1.StopWordRequest) (*searchv1.StopWordResponse, error) {
	if err := s.service.DeleteStopWord(ctx, req.GetTerm()); err != nil {
		return nil, err
	}
	return &searchv1.StopWordResponse{Term: core.NormalizeQuery(req.GetTerm())}, nil
}

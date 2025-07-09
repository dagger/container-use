package mcpserver

import (
	"context"

	"dagger.io/dagger"
	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/repository"
	"github.com/mark3labs/mcp-go/mcp"
)

func dagFromContext(ctx context.Context) *dagger.Client {
	dag, ok := ctx.Value(daggerClientKey{}).(*dagger.Client)
	if !ok {
		panic("dagger client not found in context")
	}
	return dag
}

type Request interface {
	isRequest()
}

type BaseRequest struct {
	Explanation string `json:"explanation"`
}

func (BaseRequest) isRequest() {}

type BaseRepositoryRequest struct {
	BaseRequest

	EnvironmentSource string `json:"environment_source"`
}

type BaseEnvironmentRequest struct {
	BaseRepositoryRequest

	EnvironmentID string `json:"environment_id"`
}

type Response any

type ToolResponse[T Response] struct {
	Message string
	Data    T
}

func openRepositoryFromRequest(ctx context.Context, request BaseRepositoryRequest) (*repository.Repository, error) {
	repo, err := repository.Open(ctx, request.EnvironmentSource)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func openEnvironmentFromRequest(ctx context.Context, request BaseEnvironmentRequest) (*repository.Repository, *environment.Environment, error) {
	repo, err := openRepositoryFromRequest(ctx, request.BaseRepositoryRequest)
	if err != nil {
		return nil, nil, err
	}
	env, err := repo.Get(ctx, dagFromContext(ctx), request.EnvironmentID)
	if err != nil {
		return nil, nil, err
	}
	return repo, env, nil
}

func parseBaseRequest(request mcp.CallToolRequest) (BaseRequest, error) {
	explanation, err := request.RequireString("explanation")
	if err != nil {
		return BaseRequest{}, err
	}
	return BaseRequest{
		Explanation: explanation,
	}, nil
}

func parseBaseRepositoryRequest(request mcp.CallToolRequest) (BaseRepositoryRequest, error) {
	base, err := parseBaseRequest(request)
	if err != nil {
		return BaseRepositoryRequest{}, err
	}
	environmentSource, err := request.RequireString("environment_source")
	if err != nil {
		return BaseRepositoryRequest{}, err
	}
	return BaseRepositoryRequest{
		BaseRequest:       base,
		EnvironmentSource: environmentSource,
	}, nil
}

func parseBaseEnvironmentRequest(request mcp.CallToolRequest) (BaseEnvironmentRequest, error) {
	base, err := parseBaseRepositoryRequest(request)
	if err != nil {
		return BaseEnvironmentRequest{}, err
	}
	environmentID, err := request.RequireString("environment_id")
	if err != nil {
		return BaseEnvironmentRequest{}, err
	}
	return BaseEnvironmentRequest{
		BaseRepositoryRequest: base,
		EnvironmentID:         environmentID,
	}, nil
}

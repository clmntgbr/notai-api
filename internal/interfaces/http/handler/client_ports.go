package handler

import (
	"context"

	clientcmd "go-api/internal/application/command/client"
	queryclient "go-api/internal/application/query/client"
	domainclient "go-api/internal/domain/client"

	"github.com/google/uuid"
)

type clientCreateHandler interface {
	Handle(ctx context.Context, cmd clientcmd.CreateClientCommand) (*domainclient.Client, error)
}

type clientUpdateHandler interface {
	Handle(ctx context.Context, cmd clientcmd.UpdateClientCommand) error
}

type clientDeleteHandler interface {
	Handle(ctx context.Context, id uuid.UUID) error
}

type clientRemoveMemberHandler interface {
	Handle(ctx context.Context, cmd clientcmd.RemoveClientMemberCommand) error
}

type clientGetByIDHandler interface {
	Handle(ctx context.Context, q queryclient.GetClientByIDQuery) (*domainclient.ClientView, error)
}

type clientListByUserHandler interface {
	Handle(ctx context.Context, q queryclient.ListClientsByUserQuery) ([]domainclient.ClientView, int64, error)
}

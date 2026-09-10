package client

import (
	"context"
	"encoding/json"
	"log"

	"go-api/internal/application/messaging"
	domainclient "go-api/internal/domain/client"
)

type ClientCreatedHandler struct{}

func NewClientCreatedHandler() *ClientCreatedHandler { return &ClientCreatedHandler{} }

func (h *ClientCreatedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainclient.ClientCreated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	log.Printf(
		"event handled %s eventId=%s clientId=%s name=%s",
		domainclient.EventTypeClientCreated,
		evt.ID,
		evt.ClientID,
		evt.Name,
	)
	return nil
}

type ClientUpdatedHandler struct{}

func NewClientUpdatedHandler() *ClientUpdatedHandler { return &ClientUpdatedHandler{} }

func (h *ClientUpdatedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainclient.ClientUpdated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	log.Printf(
		"event handled %s eventId=%s clientId=%s name=%s",
		domainclient.EventTypeClientUpdated,
		evt.ID,
		evt.ClientID,
		evt.Name,
	)
	return nil
}

type ClientDeletedHandler struct{}

func NewClientDeletedHandler() *ClientDeletedHandler { return &ClientDeletedHandler{} }

func (h *ClientDeletedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainclient.ClientDeleted
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	log.Printf(
		"event handled %s eventId=%s clientId=%s",
		domainclient.EventTypeClientDeleted,
		evt.ID,
		evt.ClientID,
	)
	return nil
}

type ClientMemberAddedHandler struct{}

func NewClientMemberAddedHandler() *ClientMemberAddedHandler { return &ClientMemberAddedHandler{} }

func (h *ClientMemberAddedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainclient.ClientMemberAdded
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	log.Printf(
		"event handled %s eventId=%s clientId=%s userId=%s",
		domainclient.EventTypeClientMemberAdded,
		evt.ID,
		evt.ClientID,
		evt.UserID,
	)
	return nil
}

type ClientMemberRemovedHandler struct{}

func NewClientMemberRemovedHandler() *ClientMemberRemovedHandler {
	return &ClientMemberRemovedHandler{}
}

func (h *ClientMemberRemovedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainclient.ClientMemberRemoved
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	log.Printf(
		"event handled %s eventId=%s clientId=%s userId=%s",
		domainclient.EventTypeClientMemberRemoved,
		evt.ID,
		evt.ClientID,
		evt.UserID,
	)
	return nil
}

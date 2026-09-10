package user

import (
	"context"
	"encoding/json"
	"log"

	"go-api/internal/application/messaging"
	domainuser "go-api/internal/domain/user"
)

type UserCurrentClientChangedHandler struct{}

func NewUserCurrentClientChangedHandler() *UserCurrentClientChangedHandler {
	return &UserCurrentClientChangedHandler{}
}

func (h *UserCurrentClientChangedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainuser.UserCurrentClientChanged
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	log.Printf(
		"event handled %s eventId=%s userId=%s clientId=%s",
		domainuser.EventTypeUserCurrentClientChanged,
		evt.ID,
		evt.UserID,
		evt.ClientID,
	)
	return nil
}

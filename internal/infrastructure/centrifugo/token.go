package centrifugo

import (
	"fmt"
	"time"

	"go-api/internal/infrastructure/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ConnectionInfo struct {
	Token    string            `json:"token"`
	Channel  string            `json:"channel"`
	Channels map[string]string `json:"channels"`
	WSURL    string            `json:"wsUrl"`
}

func NewConnectionInfo(env *config.Config, userID uuid.UUID) (ConnectionInfo, error) {
	channels := map[string]string{
		InterestAccount:  UserChannel(userID),
		InterestMedia:    UserMediaChannel(userID),
		InterestContent:  UserContentChannel(userID),
		InterestActivity: UserActivityChannel(userID),
	}
	token, err := generateConnectionToken(env.CentrifugoTokenSecret, userID, UserSubscribeChannels(userID))
	if err != nil {
		return ConnectionInfo{}, err
	}

	return ConnectionInfo{
		Token:    token,
		Channel:  channels[InterestAccount],
		Channels: channels,
		WSURL:    env.CentrifugoPublicWSURL,
	}, nil
}

func generateConnectionToken(secret string, userID uuid.UUID, channels []string) (string, error) {
	claims := jwt.MapClaims{
		"sub":      userID.String(),
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"channels": channels,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign centrifugo token: %w", err)
	}
	return signed, nil
}

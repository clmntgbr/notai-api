package centrifugo

import "github.com/google/uuid"

const (
	InterestAccount  = "account"
	InterestMedia    = "media"
	InterestContent  = "content"
	InterestActivity = "activity"
)

func UserChannel(userID uuid.UUID) string {
	id := userID.String()
	// Trailing #userId makes the channel user-limited in Centrifugo.
	return "users:" + id + "#" + id
}

func UserMediaChannel(userID uuid.UUID) string {
	id := userID.String()
	return "users:" + id + ":media#" + id
}

func UserContentChannel(userID uuid.UUID) string {
	id := userID.String()
	return "users:" + id + ":content#" + id
}

func UserActivityChannel(userID uuid.UUID) string {
	id := userID.String()
	return "users:" + id + ":activity#" + id
}

func UserInterestChannel(userID uuid.UUID, interest string) (string, error) {
	switch interest {
	case "", InterestAccount:
		return UserChannel(userID), nil
	case InterestMedia:
		return UserMediaChannel(userID), nil
	case InterestContent:
		return UserContentChannel(userID), nil
	case InterestActivity:
		return UserActivityChannel(userID), nil
	default:
		return "", ErrUnknownInterest
	}
}

func UserSubscribeChannels(userID uuid.UUID) []string {
	return []string{
		UserChannel(userID),
		UserMediaChannel(userID),
		UserContentChannel(userID),
		UserActivityChannel(userID),
	}
}

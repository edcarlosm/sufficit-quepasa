package models

import (
	"strconv"
	"strings"
	"time"
)

// returning []QPMessageV1
// bot.GetMessages(searchTime)
func GetMessagesFromBotV2(source QPBot, timestamp string) (messages []QpMessageV2, err error) {

	server, err := GetServerFromBot(source)
	if err != nil {
		return
	}

	searchTimestamp, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		if len(timestamp) > 0 {
			return
		} else {
			searchTimestamp = 0
		}
	}

	searchTime := time.Unix(searchTimestamp, 0)
	messages = GetMessagesFromServerV2(server, searchTime)
	return
}

func ToQPBotV2(source *QpServer) (destination *QPBotV2) {
	destination = &QPBotV2{
		ID:              source.WId,
		Verified:        source.Verified,
		Token:           source.Token,
		UserID:          source.User,
		Devel:           source.Devel,
		HandleGroups:    source.HandleGroups,
		HandleBroadcast: source.HandleBroadcast,
		UpdatedAt:       source.Timestamp.String(),
		Version:         "multi",
	}

	if !strings.Contains(destination.ID, "@") {
		destination.ID = destination.ID + "@c.us"
	}
	return
}

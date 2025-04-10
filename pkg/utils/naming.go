package utils

import "fmt"

func GetSessionActorName(sessionId string) string {
	sessionActorName := fmt.Sprintf("%s-session", sessionId)
	return sessionActorName
}

func GetDefaultSSEConnectionName(sessionId string) string {
	return fmt.Sprintf("%s-channels-default", sessionId)
}

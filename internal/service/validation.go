package service

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func parseUUIDField(field, raw string) (uuid.UUID, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return uuid.Nil, fmt.Errorf("%s is required", field)
	}
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, fmt.Errorf("%s must be a valid UUID", field)
	}
	return id, nil
}

func normalizeRequiredName(field, raw string, maxLength int) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	if len(value) > maxLength {
		return "", fmt.Errorf("%s must be at most %d characters", field, maxLength)
	}
	if strings.ContainsAny(value, "\r\n\t") {
		return "", fmt.Errorf("%s must not contain control characters", field)
	}
	return value, nil
}
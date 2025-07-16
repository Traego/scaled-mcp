package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/traego/scaled-mcp/pkg/utils"
	"strings"
)

const DELIMITER = "_"

// SessionIDManager handles creating and verifying secure session IDs
type SignedStatelessSessionManager struct {
	secret []byte
}

func (sm *SignedStatelessSessionManager) CanSaveSession() bool {
	return false
}

func (sm *SignedStatelessSessionManager) SaveSession(sessionId string, sessionData interface{}) error {
	return errors.New("stateless_session_manager does not support saving sessions")
}

func (sm *SignedStatelessSessionManager) LoadSession(sessionId string, checkUniqueId *string) (interface{}, error) {
	return nil, ErrSessionStateNotFound
}

// NewSessionIDManager creates a new session ID manager with the provided secret
func NewSignedStatelessSessionManager(secret string) SessionManager {
	return &SignedStatelessSessionManager{
		secret: []byte(secret),
	}
}

// GenerateSessionID creates a signed session ID with format:
// [random_id].[timestamp].[signature]
// The signature is an HMAC-SHA256 of the random_id and timestamp
func (sm *SignedStatelessSessionManager) GenerateSessionId(uniqueCallerId *string) (string, error) {
	length := 16
	// Generate random ID
	randomId, err := utils.GenerateSecureID(length)
	if err != nil {
		return "", fmt.Errorf("failed to generate random ID: %w", err)
	}

	baseStr := getBaseSessionStr(randomId, uniqueCallerId)

	// Generate signature
	h := hmac.New(sha256.New, sm.secret)
	h.Write([]byte(baseStr))
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	// Combine all parts
	fullStr := fmt.Sprintf("%s%s%s", baseStr, DELIMITER, signature)
	return fullStr, nil
}

func getBaseSessionStr(randomId string, uniqueId *string) string {
	var uid string
	if uniqueId == nil {
		uid = ""
	} else {
		uid = *uniqueId
	}
	return fmt.Sprintf("%s%s%s", randomId, DELIMITER, uid)
}

// VerifySessionId checks if a session ID is valid and not expired
func (sm *SignedStatelessSessionManager) VerifySessionId(sessionId string, checkUniqueId *string) error {
	parts := strings.SplitN(sessionId, DELIMITER, 3)
	if len(parts) < 2 {
		return NewErrSessionInvalid("unexpected shape of session id")
	}

	var randomId string
	var providedSig string
	var uniqueId *string
	randomId = parts[0]
	if parts[1] != "" {
		uniqueId = &parts[1]
	}
	providedSig = parts[2]

	if checkUniqueId != nil {
		if uniqueId == nil || *uniqueId != *checkUniqueId {
			return NewErrSessionInvalid("unique caller ID mismatch")
		}
	}

	// Verify signature
	baseStr := getBaseSessionStr(randomId, uniqueId)
	h := hmac.New(sha256.New, sm.secret)
	h.Write([]byte(baseStr))
	expectedSig := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	if expectedSig != providedSig {
		return NewErrSessionInvalid("invalid signature")
	}

	return nil
}

package token

import (
	"fmt"
	"time"

	"aidanwoods.dev/go-paseto"
)

// PasetoMaker is a PASETO token maker
type PasetoMaker struct {
	symmetricKey paseto.V4SymmetricKey
}

// NewPasetoMaker creates a new PasetoMaker
func NewPasetoMaker(symmetricKey string) (Maker, error) {
	// V4 local tokens require a 32-byte key (same as chacha20poly1305.KeySize)
	if len(symmetricKey) != 32 {
		return nil, fmt.Errorf("invalid key size: must be exactly 32 characters")
	}

	key, err := paseto.V4SymmetricKeyFromBytes([]byte(symmetricKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create symmetric key: %w", err)
	}

	return &PasetoMaker{symmetricKey: key}, nil
}

// CreateToken creates a new token for a specific username and duration
func (maker *PasetoMaker) CreateToken(username string, duration time.Duration) (string, error) {
	payload, err := NewPayload(username, duration)
	if err != nil {
		return "", err
	}

	token := paseto.NewToken()
	token.SetExpiration(payload.ExpiredAt)

	if err := token.Set("payload", payload); err != nil {
		return "", fmt.Errorf("failed to set payload: %w", err)
	}

	return token.V4Encrypt(maker.symmetricKey, nil), nil
}

// VerifyToken checks if the token is valid or not
func (maker *PasetoMaker) VerifyToken(tokenStr string) (*Payload, error) {
	parser := paseto.NewParser()
	parser.AddRule(paseto.NotExpired())

	parsed, err := parser.ParseV4Local(maker.symmetricKey, tokenStr, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	payload := &Payload{}
	if err := parsed.Get("payload", payload); err != nil {
		return nil, ErrInvalidToken
	}

	if err := payload.Valid(); err != nil {
		return nil, err
	}

	return payload, nil
}

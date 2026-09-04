// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	// ErrUnsupportedProvider indicates that no KMS client is registered for the specified URI scheme.
	ErrUnsupportedProvider = errors.New("kms: unsupported key provider")

	// ErrInvalidURI indicates an invalid KMS URI syntax.
	ErrInvalidURI = errors.New("kms: invalid key URI")

	// ErrDecryptionFailed indicates the KMS service could not decrypt the ciphertext.
	ErrDecryptionFailed = errors.New("kms: decryption failed")
)

// Client defines the common interface for Key Management Service envelope encryption providers
// (e.g., AWS KMS, HashiCorp Vault Transit, cloud HSMs, or in-memory mock KMS).
type Client interface {
	// Encrypt encrypts a plaintext DEK or secret using the specified key ID or ARN.
	Encrypt(ctx context.Context, keyID string, plaintext []byte) ([]byte, error)

	// Decrypt decrypts a KMS ciphertext blob to recover the plaintext DEK or secret.
	Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)
}

// KeyURI represents a parsed KMS key identifier.
type KeyURI struct {
	Provider string // "aws", "vault", "mock", etc.
	KeyPath  string // Full key ARN, key name, or transit path
	Raw      string // Original raw URI string
}

// ParseURI parses a key identifier or URI into provider and key path.
// Supported formats:
//   - "arn:aws:kms:..." -> Provider: "aws", KeyPath: full ARN
//   - "aws://<key-id-or-arn>" -> Provider: "aws", KeyPath: <key-id-or-arn>
//   - "vault://<transit-key-path>" -> Provider: "vault", KeyPath: <transit-key-path>
//   - "mock://<key-id>" -> Provider: "mock", KeyPath: <key-id>
func ParseURI(rawURI string) (*KeyURI, error) {
	trimmed := strings.TrimSpace(rawURI)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: URI is empty", ErrInvalidURI)
	}

	if strings.HasPrefix(trimmed, "arn:aws:kms:") {
		return &KeyURI{
			Provider: "aws",
			KeyPath:  trimmed,
			Raw:      trimmed,
		}, nil
	}

	if idx := strings.Index(trimmed, "://"); idx != -1 {
		scheme := strings.ToLower(trimmed[:idx])
		path := trimmed[idx+3:]
		switch scheme {
		case "aws", "aws-kms":
			return &KeyURI{Provider: "aws", KeyPath: path, Raw: trimmed}, nil
		case "vault":
			return &KeyURI{Provider: "vault", KeyPath: strings.TrimPrefix(path, "transit/keys/"), Raw: trimmed}, nil
		case "mock":
			return &KeyURI{Provider: "mock", KeyPath: path, Raw: trimmed}, nil
		default:
			return &KeyURI{Provider: scheme, KeyPath: path, Raw: trimmed}, nil
		}
	}

	// Default fallback: if no scheme provided and not an ARN, return error
	return nil, fmt.Errorf("%w: unable to determine provider from '%s'", ErrInvalidURI, rawURI)
}

// Router dispatches KMS operations to registered provider clients based on URI schemes.
type Router struct {
	mu      sync.RWMutex
	clients map[string]Client
}

// NewRouter creates a thread-safe KMS router.
func NewRouter() *Router {
	return &Router{
		clients: make(map[string]Client),
	}
}

// Register registers a KMS client for the specified provider scheme (e.g. "aws", "vault", "mock").
func (r *Router) Register(provider string, client Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[strings.ToLower(provider)] = client
}

// ClientFor resolves the appropriate Client and KeyPath for a given key URI.
func (r *Router) ClientFor(rawURI string) (Client, string, error) {
	parsed, err := ParseURI(rawURI)
	if err != nil {
		return nil, "", err
	}

	r.mu.RLock()
	client, ok := r.clients[parsed.Provider]
	r.mu.RUnlock()

	if !ok {
		return nil, "", fmt.Errorf("%w: '%s'", ErrUnsupportedProvider, parsed.Provider)
	}

	return client, parsed.KeyPath, nil
}

// Encrypt resolves the client from rawURI and executes encryption.
func (r *Router) Encrypt(ctx context.Context, rawURI string, plaintext []byte) ([]byte, error) {
	client, keyPath, err := r.ClientFor(rawURI)
	if err != nil {
		return nil, err
	}
	return client.Encrypt(ctx, keyPath, plaintext)
}

// Decrypt resolves the client from rawURI and executes decryption.
func (r *Router) Decrypt(ctx context.Context, rawURI string, ciphertext []byte) ([]byte, error) {
	client, keyPath, err := r.ClientFor(rawURI)
	if err != nil {
		return nil, err
	}
	return client.Decrypt(ctx, keyPath, ciphertext)
}

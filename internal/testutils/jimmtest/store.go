// Copyright 2024 Canonical.
package jimmtest

import (
	"context"
	"sync"
	"time"

	"github.com/juju/names/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"

	"github.com/canonical/jimm/v3/internal/errors"
)

type controllerCredentials struct {
	username string
	password string
}

// InMemoryCredentialStore implements CredentialStore but only implements
// JWKS methods in order to use it as an in memory credential store replacing
// vault for tests.
type InMemoryCredentialStore struct {
	mu                        sync.RWMutex
	jwks                      jwk.Set
	privateKey                []byte
	expiry                    time.Time
	oauthKey                  []byte
	oauthSessionStoreSecret   []byte
	controllerCredentials     map[string]controllerCredentials
	cloudCredentialAttributes map[string]map[string]string
	controllerProxyConfigs    map[string]controllerProxyConfig
}

// NewInMemoryCredentialStore returns a new instance of `InMemoryCredentialStore`
// with some secrets/keys being populated.
func NewInMemoryCredentialStore() *InMemoryCredentialStore {
	return &InMemoryCredentialStore{
		oauthKey:                []byte(JWTTestSecret),
		oauthSessionStoreSecret: []byte(SessionStoreSecret),
	}
}

// Get retrieves the stored attributes of a cloud credential.
func (s *InMemoryCredentialStore) Get(ctx context.Context, credTag names.CloudCredentialTag) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	attrs, ok := s.cloudCredentialAttributes[credTag.String()]
	if !ok {
		return nil, errors.E(errors.CodeNotFound)
	}
	attrsCopy := make(map[string]string, len(attrs))
	for k, v := range attrs {
		attrsCopy[k] = v
	}
	return attrsCopy, nil
}

// Put stores the attributes of a cloud credential.
func (s *InMemoryCredentialStore) Put(ctx context.Context, credTag names.CloudCredentialTag, attrs map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cloudCredentialAttributes == nil {
		s.cloudCredentialAttributes = make(map[string]map[string]string)
	}

	attrsCopy := make(map[string]string, len(attrs))
	for k, v := range attrs {
		attrsCopy[k] = v
	}
	s.cloudCredentialAttributes[credTag.String()] = attrsCopy
	return nil
}

// GetControllerCredentials retrieves the credentials for the given controller from a vault
// service.
func (s *InMemoryCredentialStore) GetControllerCredentials(ctx context.Context, controllerName string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cc, ok := s.controllerCredentials[controllerName]
	if !ok {
		return "", "", errors.E(errors.CodeNotFound)
	}
	return cc.username, cc.password, nil
}

// PutControllerCredentials stores the controller credentials in a vault
// service.
func (s *InMemoryCredentialStore) PutControllerCredentials(ctx context.Context, controllerName string, username string, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.controllerCredentials == nil {
		s.controllerCredentials = map[string]controllerCredentials{
			controllerName: {
				username: username,
				password: password,
			},
		}
	} else {
		s.controllerCredentials[controllerName] = controllerCredentials{
			username: username,
			password: password,
		}
	}
	return nil
}

// CleanupJWKS removes all secrets associated with the JWKS process.
func (s *InMemoryCredentialStore) CleanupJWKS(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jwks = nil
	s.privateKey = nil
	s.expiry = time.Time{}

	return nil
}

// GetJWKS returns the current key set stored within the credential store.
func (s *InMemoryCredentialStore) GetJWKS(ctx context.Context) (jwk.Set, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.jwks == nil {
		return nil, errors.E(errors.CodeNotFound)
	}
	jwks := s.jwks
	return jwks, nil
}

// GetJWKSPrivateKey returns the current private key for the active JWKS.
func (s *InMemoryCredentialStore) GetJWKSPrivateKey(ctx context.Context) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.privateKey) == 0 {
		return nil, errors.E(errors.CodeNotFound)
	}

	pk := make([]byte, len(s.privateKey))
	copy(pk, s.privateKey)

	return pk, nil
}

// GetJWKSExpiry returns the expiry of the active JWKS.
func (s *InMemoryCredentialStore) GetJWKSExpiry(ctx context.Context) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.expiry.IsZero() {
		return time.Time{}, errors.E(errors.CodeNotFound)
	}

	return s.expiry, nil
}

// PutJWKS puts a generated RS256[4096 bit] JWKS without x5c or x5t into the credential store.
func (s *InMemoryCredentialStore) PutJWKS(ctx context.Context, jwks jwk.Set) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jwks = jwks

	return nil
}

// PutJWKSPrivateKey persists the private key associated with the current JWKS within the store.
func (s *InMemoryCredentialStore) PutJWKSPrivateKey(ctx context.Context, pem []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.privateKey = make([]byte, len(pem))
	copy(s.privateKey, pem)

	return nil
}

// PutJWKSExpiry sets the expiry time for the current JWKS within the store.
func (s *InMemoryCredentialStore) PutJWKSExpiry(ctx context.Context, expiry time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.expiry = expiry

	return nil
}

// controllerProxyConfig holds proxy configuration for a controller.
type controllerProxyConfig struct {
	proxyType string
	config    map[string]interface{}
}

// Add a field to store proxy configurations.
func (s *InMemoryCredentialStore) ensureControllerProxyConfig() {
	if s.controllerProxyConfigs == nil {
		s.controllerProxyConfigs = make(map[string]controllerProxyConfig)
	}
}

// GetControllerProxy retrieves the proxy configuration for the specified controller name.
func (s *InMemoryCredentialStore) GetControllerProxy(ctx context.Context, controllerName string) (string, map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.controllerProxyConfigs == nil {
		return "", nil, errors.E(errors.CodeNotFound)
	}

	cfg, ok := s.controllerProxyConfigs[controllerName]
	if !ok {
		return "", nil, errors.E(errors.CodeNotFound)
	}

	// Deep copy config map
	configCopy := make(map[string]interface{}, len(cfg.config))
	for k, v := range cfg.config {
		configCopy[k] = v
	}

	return cfg.proxyType, configCopy, nil
}

// PutControllerProxy stores the proxy configuration for the specified controller name.
func (s *InMemoryCredentialStore) PutControllerProxy(ctx context.Context, controllerName string, proxyType string, config map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureControllerProxyConfig()

	configCopy := make(map[string]interface{}, len(config))
	for k, v := range config {
		configCopy[k] = v
	}

	s.controllerProxyConfigs[controllerName] = controllerProxyConfig{
		proxyType: proxyType,
		config:    configCopy,
	}
	return nil
}

// DeleteControllerProxy removes the proxy configuration for the specified controller name.
func (s *InMemoryCredentialStore) DeleteControllerProxy(ctx context.Context, controllerName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.controllerProxyConfigs == nil {
		return errors.E(errors.CodeNotFound)
	}

	if _, ok := s.controllerProxyConfigs[controllerName]; !ok {
		return errors.E(errors.CodeNotFound)
	}

	delete(s.controllerProxyConfigs, controllerName)
	return nil
}

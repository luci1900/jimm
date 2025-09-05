// Copyright 2025 Canonical.

package logger

import (
	"context"
	"strings"

	"github.com/juju/zaputil/zapctx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type eventLevel string

const (
	info     eventLevel = "INFO"
	warning  eventLevel = "WARN"
	critical eventLevel = "CRITICAL"
)

type securityEvent struct {
	Event       string
	Description string
	Severity    eventLevel
}

func (s securityEvent) toZapFields() []zapcore.Field {
	return []zapcore.Field{
		zap.String("event", s.Event),
		zap.String("description", s.Description),
		zap.String("severity", string(s.Severity)),
	}
}

func logSecurityEvent(ctx context.Context, event securityEvent) {
	switch event.Severity {
	case info:
		zapctx.Info(ctx, event.Event, event.toZapFields()...)
	case warning:
		zapctx.Warn(ctx, event.Event, event.toZapFields()...)
	case critical:
		zapctx.Error(ctx, event.Event, event.toZapFields()...)
	}
}

// LogFailedLogin logs a failed login attempt for the given identityId.
func LogFailedLogin(ctx context.Context, identityId string) {
	logSecurityEvent(ctx, securityEvent{
		Event:       "authn_login_fail:" + identityId,
		Description: "login failed",
		Severity:    warning,
	})
}

// LogSuccessfulLogin logs a successful login attempt for the given identityId.
func LogSuccessfulLogin(ctx context.Context, identityId string) {
	logSecurityEvent(ctx, securityEvent{
		Event:       "authn_login_success:" + identityId,
		Description: "login succeeded",
		Severity:    info,
	})
}

// LogUnauthorizedAccess logs an unauthorized access attempt for the given identityId.
func LogUnauthorizedAccess(ctx context.Context, identityId string, description string) {
	logSecurityEvent(ctx, securityEvent{
		Event:       "authz_fail:" + identityId,
		Description: description,
		Severity:    critical,
	})
}

// LogGrantJimmAdmins logs the granting of JIMM admin role to the given identityIds.
func LogGrantJimmAdmins(ctx context.Context, identityIds []string) {
	identities := strings.Join(identityIds, ", ")
	logSecurityEvent(ctx, securityEvent{
		Event:       "authz_admin:" + identities,
		Description: "JIMM admin role was granted.",
		Severity:    warning,
	})
}

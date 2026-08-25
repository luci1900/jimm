// Copyright 2025 Canonical.

package jujuauth

import (
	"context"

	"github.com/juju/names/v5"
	"github.com/juju/version/v2"

	"github.com/canonical/jimm/v3/internal/openfga"
)

// minJujuVersionForScopedToken is the first Juju controller version that
// correctly honours model-level access in a JWT during permission checks.
// Controllers older than this version incorrectly reject tokens that carry
// model-admin access without a controller-superuser claim, so a superuser
// fallback token must be minted for them instead.
//
// TODO(de-proxy): Once the minimum supported Juju controller version
// exceeds this boundary, remove NewSuperuserLoginToken and
// makeSuperuserToken entirely.
var minJujuVersionForScopedToken = version.MustParse("3.6.24")

// Factory holds the necessary components for producing
// Juju authenticator objects. Currently a login token generator
// and an SSH token generator are available.
type Factory struct {
	db            GeneratorDatabase
	jwtService    JWTService
	accessChecker GeneratorAccessChecker
}

// NewFactory returns a new factory object.
func NewFactory(db GeneratorDatabase, jwtService JWTService, accessChecker GeneratorAccessChecker) *Factory {
	return &Factory{
		db:            db,
		jwtService:    jwtService,
		accessChecker: accessChecker,
	}
}

// NewLoginGenerator returns a new token generator for Juju RPC login requests.
// The LoginTokenGenerator is stateful and should be re-used for the lifetime
// of a single connection, and recreated for each new connection.
func (f *Factory) NewLoginGenerator() LoginTokenGenerator {
	return newLoginTokenGenerator(f.db, f.accessChecker, f.jwtService)
}

// NewLoginToken returns a Juju login token for the given user, model, and controller.
//
// This is a convenience method that wraps the creation of a LoginTokenGenerator and
// the generation of a login token in one step. This is useful for scenarios where we
// don't have a long lived connection that may need multiple tokens.
func (f *Factory) NewLoginToken(ctx context.Context, modelTag names.ModelTag, controllerTag names.ControllerTag, user *openfga.User) ([]byte, error) {
	generator := f.NewLoginGenerator()
	generator.SetTags(modelTag, controllerTag)
	return generator.MakeLoginToken(ctx, user)
}

// NewSuperuserLoginToken creates a login token for the provided user with controller superuser and model admin permissions.
//
// NB: Avoid using this method and prefer NewLoginTokenForController to mint a token with the
// user's real permissions when the controller version supports it.
//
// This is only used as a fallback for avoiding a bug in Juju 3.6.23 and below where Juju does
// not correctly check the model admin permission for a JWT.
func (f *Factory) NewSuperuserLoginToken(ctx context.Context, modelTag names.ModelTag, controllerTag names.ControllerTag, user *openfga.User) ([]byte, error) {
	generator := f.NewLoginGenerator()
	generator.SetTags(modelTag, controllerTag)
	return generator.makeSuperuserToken(ctx, user)
}

// NewLoginTokenForController mints a login token for the given user on the
// specified controller. If the controller's agent version supports caller-scoped
// tokens (>= minJujuVersionForScopedToken) a properly-scoped token is minted;
// otherwise a superuser token is returned as a fallback for older Juju versions
// that do not correctly honour model-level JWT claims.
//
// The model tag is required for both the caller-scoped and superuser fallback
// paths.
func (f *Factory) NewLoginTokenForController(ctx context.Context, modelTag names.ModelTag, controllerTag names.ControllerTag, user *openfga.User, controllerAgentVersion string) ([]byte, error) {
	ctrlVersion, err := version.Parse(controllerAgentVersion)
	if err != nil || ctrlVersion.Compare(minJujuVersionForScopedToken) < 0 {
		// Unknown version or too old: use the superuser fallback.
		return f.NewSuperuserLoginToken(ctx, modelTag, controllerTag, user)
	}
	return f.NewLoginToken(ctx, modelTag, controllerTag, user)
}

// NewSSHGenerator returns a new token generator for Juju SSH connections.
// The SSHToken generator is not stateful and can be re-used across
// multiple connections.
func (f *Factory) NewSSHGenerator() SSHTokenGenerator {
	return newSSHTokenGenerator(f.jwtService)
}

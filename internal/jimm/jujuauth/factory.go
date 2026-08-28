// Copyright 2025 Canonical.

package jujuauth

import (
	"context"

	"github.com/juju/names/v5"
	"github.com/juju/version/v2"

	"github.com/canonical/jimm/v3/internal/dbmodel"
	"github.com/canonical/jimm/v3/internal/jimmjwx"
	"github.com/canonical/jimm/v3/internal/openfga"
)

// minJujuVersionForScopedToken is the first Juju controller version that
// correctly honours model-level access in a JWT during permission checks.
// Controllers older than this version incorrectly reject tokens that carry
// model-admin access without a controller-superuser claim, so a superuser
// fallback token must be minted for them instead.
//
// TODO(de-proxy): Once the minimum supported Juju controller version
// exceeds this boundary, remove the superuser fallback in
// NewCallerLoginToken and makeSuperuserToken entirely.
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

// NewScopedLoginToken mints a caller-scoped login token for the given user
// on the specified controller. If the controller is already persisted, its
// CloudRegions are fetched from the database so cloud-access claims can be
// resolved; otherwise the passed-in controller is used as-is (which may
// not yet have CloudRegions populated, so cloud claims may be omitted).
//
// targets may contain any mix of model and application-offer tags whose
// access levels should be embedded in the token; pass no targets when the
// operation is not tied to a specific resource (e.g. cloud or
// controller-level calls).
func (f *Factory) NewScopedLoginToken(ctx context.Context, targets []names.Tag, ctl *dbmodel.Controller, user *openfga.User) ([]byte, error) {
	// Fetch the controller with CloudRegions populated so buildAccessMap can
	// enumerate clouds and resolve cloud-access claims. Fall back to the
	// passed-in controller if it is not yet persisted (e.g. during AddController,
	// which dials the controller before storing it).
	ctlWithClouds := dbmodel.Controller{}
	ctlWithClouds.SetTag(ctl.ResourceTag())
	if err := f.db.GetController(ctx, &ctlWithClouds); err != nil {
		ctlWithClouds = *ctl
	}
	accessMap, err := buildAccessMap(ctx, user, targets, ctl.ResourceTag(), ctlWithClouds, f.accessChecker)
	if err != nil {
		return nil, err
	}
	return f.jwtService.NewJWT(ctx, jimmjwx.JWTParams{
		Controller: ctl.ResourceTag().Id(),
		User:       user.Tag().String(),
		Access:     accessMap,
	})
}

// NewCallerLoginToken returns a login token suitable for a real user's call
// to the specified controller. Older Juju controllers do not correctly
// honour model-level JWT claims, so unknown and older versions use the
// compatibility superuser token.
func (f *Factory) NewCallerLoginToken(ctx context.Context, targets []names.Tag, ctl *dbmodel.Controller, user *openfga.User) ([]byte, error) {
	ctrlVersion, err := version.Parse(ctl.AgentVersion)
	if err != nil || ctrlVersion.Compare(minJujuVersionForScopedToken) < 0 {
		generator := f.NewLoginGenerator()
		generator.SetTags(names.ModelTag{}, ctl.ResourceTag())
		return generator.makeSuperuserToken(ctx, targets, user)
	}
	return f.NewScopedLoginToken(ctx, targets, ctl, user)
}

// NewSSHGenerator returns a new token generator for Juju SSH connections.
// The SSHToken generator is not stateful and can be re-used across
// multiple connections.
func (f *Factory) NewSSHGenerator() SSHTokenGenerator {
	return newSSHTokenGenerator(f.jwtService)
}

// Copyright 2025 Canonical.

// Package jujuauth generates JWT tokens to
// authenticate and authorize messages to Juju controllers.
// This package is more specialised than a generic
// JWT token generator as it crafts Juju specific
// permissions that are added as claims to the JWT
// and therefore exists in JIMM's business logic layer.
package jujuauth

import (
	"context"
	"fmt"
	"sync"

	jujuparams "github.com/juju/juju/rpc/params"
	"github.com/juju/names/v5"

	"github.com/canonical/jimm/v3/internal/dbmodel"
	"github.com/canonical/jimm/v3/internal/errors"
	"github.com/canonical/jimm/v3/internal/jimmjwx"
	"github.com/canonical/jimm/v3/internal/openfga"
	ofganames "github.com/canonical/jimm/v3/internal/openfga/names"
)

// GeneratorDatabase specifies the database interface used by the
// JWT generator.
type GeneratorDatabase interface {
	GetController(ctx context.Context, controller *dbmodel.Controller) error
}

// GeneratorAccessChecker specifies the access checker used by the JWT
// generator to obtain user's access rights to various entities.
type GeneratorAccessChecker interface {
	GetUserAccessBatch(ctx context.Context, user *openfga.User, resources []names.Tag) (map[string]openfga.Relation, error)
	CheckPermission(ctx context.Context, user *openfga.User, accessMap map[string]string, permissions map[string]any) (map[string]string, error)
}

// JWTService specifies the service JWT generator uses to generate JWTs.
type JWTService interface {
	NewJWT(context.Context, jimmjwx.JWTParams) ([]byte, error)
}

// LoginTokenGenerator provides the necessary state and
// methods to authorize a user and generate JWT tokens
// appropriate when logging in for RPC based communication..
type LoginTokenGenerator struct {
	database      GeneratorDatabase
	accessChecker GeneratorAccessChecker
	jwtService    JWTService

	mu             sync.Mutex
	accessMapCache map[string]string
	mt             names.ModelTag
	ct             names.ControllerTag
	user           *openfga.User
	callCount      int
}

// newLoginTokenGenerator returns a new LoginTokenGenerator.
func newLoginTokenGenerator(database GeneratorDatabase, accessChecker GeneratorAccessChecker, jwtService JWTService) LoginTokenGenerator {
	return LoginTokenGenerator{
		database:      database,
		accessChecker: accessChecker,
		jwtService:    jwtService,
	}
}

// SetTags implements TokenGenerator.
// It sets the model and controller we are operating on
// which will be used to set the JWT audience and claims.
func (auth *LoginTokenGenerator) SetTags(mt names.ModelTag, ct names.ControllerTag) {
	auth.mt = mt
	auth.ct = ct
}

// SetTags implements TokenGenerator.
func (auth *LoginTokenGenerator) GetUser() names.UserTag {
	if auth.user != nil {
		return auth.user.ResourceTag()
	}
	return names.UserTag{}
}

// makeSuperuserToken makes a token declaring the user is a controller
// superuser and model admin for the supplied model resource tags, without
// actually checking if that's the case.
func (auth *LoginTokenGenerator) makeSuperuserToken(ctx context.Context, resourceTags []names.Tag, user *openfga.User) ([]byte, error) {
	if user == nil {
		return nil, errors.New("user not specified")
	}

	accessMap := make(map[string]string)
	accessMap[auth.ct.String()] = "superuser"
	for _, target := range resourceTags {
		if model, ok := target.(names.ModelTag); ok && model.Id() != "" {
			accessMap[model.String()] = "admin"
		}
	}

	return auth.jwtService.NewJWT(ctx, jimmjwx.JWTParams{
		Controller: auth.ct.Id(),
		User:       user.Tag().String(),
		Access:     accessMap,
	})
}

// accessString maps a relation to the Juju access-level string used in JWT
// access claims, based on the resource kind.
func accessString(kind string, relation openfga.Relation) string {
	switch kind {
	case names.ModelTagKind:
		switch relation {
		case ofganames.AdministratorRelation:
			return "admin"
		case ofganames.WriterRelation:
			return "write"
		case ofganames.ReaderRelation:
			return "read"
		}
	case names.ApplicationOfferTagKind:
		switch relation {
		case ofganames.AdministratorRelation:
			return string(jujuparams.OfferAdminAccess)
		case ofganames.ConsumerRelation:
			return string(jujuparams.OfferConsumeAccess)
		case ofganames.ReaderRelation:
			return string(jujuparams.OfferReadAccess)
		}
	case names.ControllerTagKind:
		if relation == ofganames.AdministratorRelation {
			return "superuser"
		}
		return "login"
	case names.CloudTagKind:
		switch relation {
		case ofganames.AdministratorRelation:
			return "admin"
		case ofganames.CanAddModelRelation:
			return "add-model"
		}
	}
	return ""
}

// buildAccessMap resolves the caller's OpenFGA permissions for the given
// controller and the given resource tags, and returns a JWT access-claim
// map.
//
// resourceTags may contain any mix of names.ModelTag and
// names.ApplicationOfferTag values; each is resolved to the caller's real
// access level. All permission checks (resource tags, the controller and
// every cloud known to the controller) are issued as a single batched
// OpenFGA request.
//
// The returned map always contains the controller access entry, plus one
// entry per cloud known to the controller.
func buildAccessMap(
	ctx context.Context,
	user *openfga.User,
	resourceTags []names.Tag,
	ct names.ControllerTag,
	ctl dbmodel.Controller,
	accessChecker GeneratorAccessChecker,
) (map[string]string, error) {
	clouds := make(map[names.CloudTag]bool)
	for _, cloudRegion := range ctl.CloudRegions {
		clouds[cloudRegion.CloudRegion.Cloud.ResourceTag()] = true
	}

	// Collect every resource whose access level is needed, then resolve
	// them all in one batched OpenFGA request.
	resources := make([]names.Tag, 0, len(resourceTags)+1+len(clouds))
	resources = append(resources, resourceTags...)
	resources = append(resources, ct)
	for cloudTag := range clouds {
		resources = append(resources, cloudTag)
	}
	access, err := accessChecker.GetUserAccessBatch(ctx, user, resources)
	if err != nil {
		return nil, err
	}

	accessMap := make(map[string]string, len(access))
	hasRealAccess := false
	for _, target := range resourceTags {
		if access[target.String()] != ofganames.NoRelation {
			hasRealAccess = true
		}
	}

	// Juju rejects the entire login if any access claim in the JWT is
	// present but empty. So when the user holds real access to at least
	// one tag, empty claims are dropped to avoid invalidating the whole
	// token. When all claims are empty, one is kept so Juju itself
	// denies the login.
	for _, target := range resourceTags {
		level := accessString(target.Kind(), access[target.String()])
		if level == "" && hasRealAccess {
			continue
		}
		accessMap[target.String()] = level
	}

	accessMap[ct.String()] = accessString(ct.Kind(), access[ct.String()])
	for cloudTag := range clouds {
		accessMap[cloudTag.String()] = accessString(cloudTag.Kind(), access[cloudTag.String()])
	}

	return accessMap, nil
}

// MakeLoginToken authorizes the user and returns a JWT containing claims about user's access
// to the controller, model and all clouds that the controller knows about.
func (auth *LoginTokenGenerator) MakeLoginToken(ctx context.Context, user *openfga.User) ([]byte, error) {

	auth.mu.Lock()
	defer auth.mu.Unlock()

	if user == nil {
		return nil, errors.New("user not specified")
	}
	auth.user = user

	if auth.ct.Id() == "" {
		return nil, errors.New("controller not set")
	}

	var ctl dbmodel.Controller
	ctl.SetTag(auth.ct)
	if err := auth.database.GetController(ctx, &ctl); err != nil {
		return nil, fmt.Errorf("failed to fetch controller: %w", err)
	}

	var resourceTags []names.Tag
	if auth.mt.Id() != "" {
		resourceTags = []names.Tag{auth.mt}
	}

	// Recreate the accessMapCache to prevent leaking permissions across multiple login requests.
	var err error
	auth.accessMapCache, err = buildAccessMap(ctx, auth.user, resourceTags, auth.ct, ctl, auth.accessChecker)
	if err != nil {
		return nil, err
	}

	return auth.jwtService.NewJWT(ctx, jimmjwx.JWTParams{
		Controller: auth.ct.Id(),
		User:       auth.user.Tag().String(),
		Access:     auth.accessMapCache,
	})
}

// MakeToken assumes MakeLoginToken has already been called and checks the permissions
// specified in the permissionMap. If the logged in user has all those permissions
// a JWT will be returned with assertions confirming all those permissions.
func (auth *LoginTokenGenerator) MakeToken(ctx context.Context, permissionMap map[string]any) ([]byte, error) {

	auth.mu.Lock()
	defer auth.mu.Unlock()

	if auth.callCount >= 10 {
		return nil, errors.New("Permission check limit exceeded")
	}
	auth.callCount++
	if auth.user == nil {
		return nil, errors.New("User authorization missing.")
	}
	if permissionMap != nil {
		var err error
		auth.accessMapCache, err = auth.accessChecker.CheckPermission(ctx, auth.user, auth.accessMapCache, permissionMap)
		if err != nil {
			return nil, err
		}
	}
	jwt, err := auth.jwtService.NewJWT(ctx, jimmjwx.JWTParams{
		Controller: auth.ct.Id(),
		User:       auth.user.Tag().String(),
		Access:     auth.accessMapCache,
	})
	if err != nil {
		return nil, err
	}
	return jwt, nil
}

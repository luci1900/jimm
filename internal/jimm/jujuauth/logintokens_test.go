// Copyright 2025 Canonical.

package jujuauth_test

import (
	"context"
	"maps"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/google/uuid"
	"github.com/juju/names/v5"

	"github.com/canonical/jimm/v3/internal/dbmodel"
	"github.com/canonical/jimm/v3/internal/errors"
	"github.com/canonical/jimm/v3/internal/jimm/jujuauth"
	"github.com/canonical/jimm/v3/internal/jimmjwx"
	"github.com/canonical/jimm/v3/internal/openfga"
	ofganames "github.com/canonical/jimm/v3/internal/openfga/names"
)

// testDatabase is a database implementation intended for testing the token generator.
type testDatabase struct {
	ctl dbmodel.Controller
	err error
}

// GetController implements the GetController method of the JWTGeneratorDatabase interface.
func (tdb *testDatabase) GetController(ctx context.Context, controller *dbmodel.Controller) error {
	if tdb.err != nil {
		return tdb.err
	}
	*controller = tdb.ctl
	return nil
}

// testAccessChecker is an access checker implementation intended for testing the
// token generator. The access map is keyed by resource tag string and holds
// OpenFGA relation names (e.g. "administrator", "writer", "can_addmodel").
type testAccessChecker struct {
	access      map[string]openfga.Relation
	accessErr   error
	permissions map[string]string
	permErr     error
}

// GetUserAccessBatch implements the GetUserAccessBatch method of the
// JWTGeneratorAccessChecker interface. It returns the preconfigured
// relation for every requested resource.
func (tac *testAccessChecker) GetUserAccessBatch(ctx context.Context, user *openfga.User, resources []names.Tag) (map[string]openfga.Relation, error) {
	if tac.accessErr != nil {
		return nil, tac.accessErr
	}
	access := make(map[string]openfga.Relation, len(resources))
	for _, r := range resources {
		access[r.String()] = tac.access[r.String()]
	}
	return access, nil
}

// CheckPermission implements the CheckPermission methods of the JWTGeneratorAccessChecker interface.
func (tac *testAccessChecker) CheckPermission(ctx context.Context, user *openfga.User, accessMap map[string]string, permissions map[string]any) (map[string]string, error) {
	if tac.permErr != nil {
		return nil, tac.permErr
	}
	access := make(map[string]string)
	maps.Copy(access, accessMap)
	maps.Copy(access, tac.permissions)
	return access, nil
}

// testJWTService is a jwt service implementation intended for testing the token generator.
type testJWTService struct {
	newJWTError error

	params jimmjwx.JWTParams
}

// NewJWT implements the NewJWT methods of the JWTService interface.
func (t *testJWTService) NewJWT(ctx context.Context, params jimmjwx.JWTParams) ([]byte, error) {
	if t.newJWTError != nil {
		return nil, t.newJWTError
	}
	t.params = params
	return []byte("test jwt"), nil
}

func TestJWTGeneratorMakeLoginToken(t *testing.T) {
	c := qt.New(t)

	// Note: error-path cases for the model/controller access checks use an
	// empty testDatabase so the DB fetch inside MakeLoginToken succeeds before
	// buildAccessMap runs. The returned controller has no CloudRegions, which
	// keeps cloud lookups out of the path and makes these cases robust.
	ct := names.NewControllerTag(uuid.New().String())
	mt := names.NewModelTag(uuid.New().String())

	tests := []struct {
		about             string
		username          string
		database          *testDatabase
		accessChecker     *testAccessChecker
		jwtService        *testJWTService
		expectedError     string
		expectedJWTParams jimmjwx.JWTParams
	}{{
		about:    "initial login, all is well",
		username: "eve@canonical.com",
		database: &testDatabase{
			ctl: dbmodel.Controller{
				CloudRegions: []dbmodel.CloudRegionControllerPriority{{
					CloudRegion: dbmodel.CloudRegion{
						Cloud: dbmodel.Cloud{
							Name: "test-cloud",
						},
					},
				}},
			},
		},
		accessChecker: &testAccessChecker{
			access: map[string]openfga.Relation{
				mt.String():                              ofganames.AdministratorRelation,
				ct.String():                              ofganames.AdministratorRelation,
				names.NewCloudTag("test-cloud").String(): ofganames.CanAddModelRelation,
			},
		},
		jwtService: &testJWTService{},
		expectedJWTParams: jimmjwx.JWTParams{
			Controller: ct.Id(),
			User:       names.NewUserTag("eve@canonical.com").String(),
			Access: map[string]string{
				ct.String():                              "superuser",
				mt.String():                              "admin",
				names.NewCloudTag("test-cloud").String(): "add-model",
			},
		},
	}, {
		about:    "access check fails",
		username: "eve@canonical.com",
		database: &testDatabase{},
		accessChecker: &testAccessChecker{
			accessErr: errors.New("a test error"),
		},
		jwtService:    &testJWTService{},
		expectedError: "a test error",
	}, {
		about:    "get controller from db fails",
		username: "eve@canonical.com",
		database: &testDatabase{
			err: errors.New("a test error"),
		},
		accessChecker: &testAccessChecker{
			access: map[string]openfga.Relation{
				mt.String(): ofganames.AdministratorRelation,
				ct.String(): ofganames.AdministratorRelation,
			},
		},
		expectedError: "failed to fetch controller: a test error",
	}, {
		about:    "jwt service errors out",
		username: "eve@canonical.com",
		database: &testDatabase{
			ctl: dbmodel.Controller{
				CloudRegions: []dbmodel.CloudRegionControllerPriority{{
					CloudRegion: dbmodel.CloudRegion{
						Cloud: dbmodel.Cloud{
							Name: "test-cloud",
						},
					},
				}},
			},
		},
		accessChecker: &testAccessChecker{
			access: map[string]openfga.Relation{
				mt.String():                              ofganames.AdministratorRelation,
				ct.String():                              ofganames.AdministratorRelation,
				names.NewCloudTag("test-cloud").String(): ofganames.CanAddModelRelation,
			},
		},
		jwtService: &testJWTService{
			newJWTError: errors.New("a test error"),
		},
		expectedError: "a test error",
	}}

	for _, test := range tests {
		authFactory := jujuauth.NewFactory(test.database, test.jwtService, test.accessChecker)
		generator := authFactory.NewLoginGenerator()
		generator.SetTags(mt, ct)

		i, err := dbmodel.NewIdentity(test.username)
		c.Assert(err, qt.IsNil)
		_, err = generator.MakeLoginToken(context.Background(), &openfga.User{
			Identity: i,
		})
		if test.expectedError != "" {
			c.Assert(err, qt.ErrorMatches, test.expectedError)
		} else {
			c.Assert(err, qt.IsNil)
			c.Assert(test.jwtService.params, qt.DeepEquals, test.expectedJWTParams)
		}
	}
}

func TestJWTGeneratorMakeToken(t *testing.T) {
	c := qt.New(t)

	ct := names.NewControllerTag(uuid.New().String())
	mt := names.NewModelTag(uuid.New().String())

	tests := []struct {
		about                 string
		checkPermissions      map[string]string
		checkPermissionsError error
		jwtService            *testJWTService
		expectedError         string
		permissions           map[string]any
		expectedJWTParams     jimmjwx.JWTParams
	}{{
		about:      "all is well",
		jwtService: &testJWTService{},
		expectedJWTParams: jimmjwx.JWTParams{
			Controller: ct.Id(),
			User:       names.NewUserTag("eve@canonical.com").String(),
			Access: map[string]string{
				ct.String():                              "superuser",
				mt.String():                              "admin",
				names.NewCloudTag("test-cloud").String(): "add-model",
			},
		},
	}, {
		about:      "check permission fails",
		jwtService: &testJWTService{},
		permissions: map[string]any{
			"entity1": "access_level1",
		},
		checkPermissionsError: errors.New("a test error"),
		expectedError:         "a test error",
	}, {
		about:      "additional permissions need checking",
		jwtService: &testJWTService{},
		permissions: map[string]any{
			"entity1": "access_level1",
		},
		checkPermissions: map[string]string{
			"entity1": "access_level1",
		},
		expectedJWTParams: jimmjwx.JWTParams{
			Controller: ct.Id(),
			User:       names.NewUserTag("eve@canonical.com").String(),
			Access: map[string]string{
				ct.String():                              "superuser",
				mt.String():                              "admin",
				names.NewCloudTag("test-cloud").String(): "add-model",
				"entity1":                                "access_level1",
			},
		},
	}}

	for _, test := range tests {
		authFactory := jujuauth.NewFactory(
			&testDatabase{
				ctl: dbmodel.Controller{
					CloudRegions: []dbmodel.CloudRegionControllerPriority{{
						CloudRegion: dbmodel.CloudRegion{
							Cloud: dbmodel.Cloud{
								Name: "test-cloud",
							},
						},
					}},
				},
			},
			test.jwtService,
			&testAccessChecker{
				access: map[string]openfga.Relation{
					mt.String():                              ofganames.AdministratorRelation,
					ct.String():                              ofganames.AdministratorRelation,
					names.NewCloudTag("test-cloud").String(): ofganames.CanAddModelRelation,
				},
				permissions: test.checkPermissions,
				permErr:     test.checkPermissionsError,
			},
		)
		generator := authFactory.NewLoginGenerator()
		generator.SetTags(mt, ct)

		i, err := dbmodel.NewIdentity("eve@canonical.com")
		c.Assert(err, qt.IsNil)
		_, err = generator.MakeLoginToken(context.Background(), &openfga.User{
			Identity: i,
		})
		c.Assert(err, qt.IsNil)

		_, err = generator.MakeToken(context.Background(), test.permissions)
		if test.expectedError != "" {
			c.Assert(err, qt.ErrorMatches, test.expectedError)
		} else {
			c.Assert(err, qt.IsNil)
			c.Assert(test.jwtService.params, qt.DeepEquals, test.expectedJWTParams)
		}
	}
}

// TestBuildAccessMapEmptyClaims verifies the empty-claim handling: Juju
// rejects a token containing a present-but-empty access claim, so claims
// for resources the user cannot access are dropped when the user holds
// real access elsewhere, and kept (so Juju denies the login) when all
// resource-tag claims are empty.
func TestBuildAccessMapEmptyClaims(t *testing.T) {
	c := qt.New(t)

	ct := names.NewControllerTag(uuid.New().String())
	cloudTag := names.NewCloudTag("test-cloud")
	ctl := dbmodel.Controller{
		CloudRegions: []dbmodel.CloudRegionControllerPriority{{
			CloudRegion: dbmodel.CloudRegion{
				Cloud: dbmodel.Cloud{Name: "test-cloud"},
			},
		}},
	}

	i, err := dbmodel.NewIdentity("eve@canonical.com")
	c.Assert(err, qt.IsNil)
	user := &openfga.User{Identity: i}

	c.Run("empty claims dropped when user has real access", func(c *qt.C) {
		modelWithAccess := names.NewModelTag(uuid.New().String())
		modelNoAccess := names.NewModelTag(uuid.New().String())
		offerNoAccess := names.NewApplicationOfferTag(uuid.New().String())

		checker := &testAccessChecker{
			access: map[string]openfga.Relation{
				modelWithAccess.String(): ofganames.WriterRelation,
				// modelNoAccess and offerNoAccess intentionally absent:
				// the user has no access, so they resolve to NoRelation.
				ct.String():       ofganames.AdministratorRelation,
				cloudTag.String(): ofganames.CanAddModelRelation,
			},
		}

		accessMap, err := jujuauth.BuildAccessMapForTest(
			context.Background(), user,
			[]names.Tag{modelWithAccess, modelNoAccess, offerNoAccess},
			ct, ctl, checker,
		)
		c.Assert(err, qt.IsNil)
		c.Check(accessMap, qt.DeepEquals, map[string]string{
			modelWithAccess.String(): "write",
			// Empty claims dropped: the token must not carry them.
			ct.String():       "superuser",
			cloudTag.String(): "add-model",
		})
	})

	c.Run("empty claim kept when user has no access at all", func(c *qt.C) {
		modelNoAccess := names.NewModelTag(uuid.New().String())

		checker := &testAccessChecker{
			access: map[string]openfga.Relation{
				ct.String():       ofganames.NoRelation,
				cloudTag.String(): ofganames.NoRelation,
			},
		}

		accessMap, err := jujuauth.BuildAccessMapForTest(
			context.Background(), user,
			[]names.Tag{modelNoAccess},
			ct, ctl, checker,
		)
		c.Assert(err, qt.IsNil)
		c.Check(accessMap, qt.DeepEquals, map[string]string{
			// The empty claim is kept so Juju itself denies the login.
			modelNoAccess.String(): "",
			ct.String():            "login",
			cloudTag.String():      "",
		})
	})

	c.Run("mixed kinds resolve to their access strings", func(c *qt.C) {
		model := names.NewModelTag(uuid.New().String())
		offer := names.NewApplicationOfferTag(uuid.New().String())

		checker := &testAccessChecker{
			access: map[string]openfga.Relation{
				model.String():    ofganames.AdministratorRelation,
				offer.String():    ofganames.ConsumerRelation,
				ct.String():       ofganames.AdministratorRelation,
				cloudTag.String(): ofganames.AdministratorRelation,
			},
		}

		accessMap, err := jujuauth.BuildAccessMapForTest(
			context.Background(), user,
			[]names.Tag{model, offer},
			ct, ctl, checker,
		)
		c.Assert(err, qt.IsNil)
		c.Check(accessMap, qt.DeepEquals, map[string]string{
			model.String():    "admin",
			offer.String():    "consume",
			ct.String():       "superuser",
			cloudTag.String(): "admin",
		})
	})
}

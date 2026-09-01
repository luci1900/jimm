// Copyright 2026 Canonical.

package testing

import (
	"context"
	"encoding/json"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/google/uuid"
	"github.com/juju/names/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"

	"github.com/canonical/jimm/v3/internal/dbmodel"
	"github.com/canonical/jimm/v3/internal/openfga"
	ofganames "github.com/canonical/jimm/v3/internal/openfga/names"
	"github.com/canonical/jimm/v3/internal/testutils/jimmtest"
)

// TestCallerTokenMultipleResourceTags verifies that buildAccessMap, which
// resolves all resource-tag, controller and cloud claims in a single
// batched OpenFGA request, mints a token carrying the caller's real
// access across a heterogeneous set of resources: a model the user
// owns (admin), a model they only have read access to, a model they
// have no access to (claim dropped), and an application offer they can
// consume.
func TestCallerTokenMultipleResourceTags(t *testing.T) {
	c := qt.New(t)
	s := jimmtest.SetupJimmWithControllers(c)
	ctx := context.Background()

	// bob owns model1 (admin via ownership), has read access to model2,
	// and no access to model3.
	model1 := s.CreateModelForBob(c)
	model2 := s.CreateModelForCharlieWithBobReadAccess(c)
	model3 := s.CreateModelForCharlie(c)

	// Grant bob consume access on an offer tag (no DB row needed for the
	// OpenFGA check).
	offerTag := names.NewApplicationOfferTag(uuid.NewString())
	err := s.OFGAClient.AddRelation(ctx, openfga.Tuple{
		Object:   ofganames.ConvertTag(names.NewUserTag("bob@canonical.com")),
		Relation: ofganames.ConsumerRelation,
		Target:   ofganames.ConvertTag(offerTag),
	})
	c.Assert(err, qt.Equals, nil)

	bobIdentity, err := dbmodel.NewIdentity("bob@canonical.com")
	c.Assert(err, qt.Equals, nil)
	err = s.JIMM.Database.GetIdentity(ctx, bobIdentity)
	c.Assert(err, qt.Equals, nil)
	bob := openfga.NewUser(bobIdentity, s.OFGAClient)

	ctl := &model1.Controller
	resourceTags := []names.Tag{
		model1.ResourceTag(),
		model2.ResourceTag(),
		model3.ResourceTag(),
		offerTag,
	}
	token, err := s.JIMM.JujuAuthFactory.NewScopedLoginToken(ctx, resourceTags, ctl, bob)
	c.Assert(err, qt.Equals, nil)

	parsed, err := jwt.Parse(token, jwt.WithVerify(false), jwt.WithValidate(false))
	c.Assert(err, qt.Equals, nil)
	c.Assert(parsed.Subject(), qt.Equals, bobIdentity.ResourceTag().String())

	access := decodeCallerTokenAccess(c, parsed)

	// model1: bob is the owner, so admin.
	c.Check(access[model1.ResourceTag().String()], qt.Equals, "admin")
	// model2: bob was granted read.
	c.Check(access[model2.ResourceTag().String()], qt.Equals, "read")
	// model3: bob has no access; the empty claim must be dropped since
	// he holds real access on other resources.
	_, ok := access[model3.ResourceTag().String()]
	c.Check(ok, qt.IsFalse, qt.Commentf("empty claim for inaccessible model must be dropped"))
	// offer: bob was granted consume.
	c.Check(access[offerTag.String()], qt.Equals, "consume")
	// controller: bob is a non-admin, so login.
	c.Check(access[ctl.ResourceTag().String()], qt.Equals, "login")
	// cloud: bob can add models on the test cloud.
	c.Check(access[names.NewCloudTag(jimmtest.TestE2ECloudName).String()], qt.Equals, "add-model")

	// No superuser claim anywhere for the non-admin user.
	for tag, level := range access {
		if level == "superuser" {
			c.Fatalf("non-admin user received superuser access on %s", tag)
		}
	}
}

// decodeScopedTokenAccess extracts the "access" claim from a parsed JWT
// as a map[string]string.
func decodeCallerTokenAccess(c *qt.C, parsed jwt.Token) map[string]string {
	accessRaw, ok := parsed.Get("access")
	c.Assert(ok, qt.IsTrue, qt.Commentf("JWT missing access claim"))
	accessBytes, err := json.Marshal(accessRaw)
	c.Assert(err, qt.IsNil)
	var access map[string]string
	err = json.Unmarshal(accessBytes, &access)
	c.Assert(err, qt.IsNil)
	return access
}

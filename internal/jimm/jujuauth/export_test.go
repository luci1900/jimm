// Copyright 2026 Canonical.

package jujuauth

import (
	"context"

	"github.com/juju/names/v5"

	"github.com/canonical/jimm/v3/internal/dbmodel"
	"github.com/canonical/jimm/v3/internal/openfga"
)

// BuildAccessMapForTest exposes buildAccessMap for testing.
func BuildAccessMapForTest(
	ctx context.Context,
	user *openfga.User,
	resourceTags []names.Tag,
	ct names.ControllerTag,
	ctl dbmodel.Controller,
	accessChecker GeneratorAccessChecker,
) (map[string]string, error) {
	return buildAccessMap(ctx, user, resourceTags, ct, ctl, accessChecker)
}

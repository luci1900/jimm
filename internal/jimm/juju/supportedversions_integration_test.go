// Copyright 2026 Canonical.

package juju_test

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/canonical/jimm/v3/internal/jimm/juju"
)

// TestSupportedVersions_Integration calls the real GitHub API and verifies
// that the response contains well-formed entries. Run with -short to skip.
func TestSupportedVersions_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	c := qt.New(t)

	j := &juju.JujuManager{} // GitHubClient nil → uses real github.com API

	resp, err := j.SupportedVersions(context.Background(), nil)
	c.Assert(err, qt.IsNil)
	c.Assert(len(resp.Versions) > 0, qt.IsTrue, qt.Commentf("expected at least one supported version from GitHub"))
	for _, v := range resp.Versions {
		c.Assert(v.Version, qt.Not(qt.Equals), "", qt.Commentf("version string must not be empty"))
		c.Assert(v.Date, qt.Not(qt.Equals), "", qt.Commentf("date must not be empty"))
		c.Assert(v.LinkToRelease, qt.Not(qt.Equals), "", qt.Commentf("link to release must not be empty"))
	}
}

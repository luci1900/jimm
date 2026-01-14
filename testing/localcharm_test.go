// Copyright 2026 Canonical.

package testing

import (
	"fmt"
	"strings"

	"github.com/juju/charm/v12"
	"github.com/juju/charm/v12/resource"
	"github.com/juju/juju/api"
	"github.com/juju/juju/api/client/charms"
	"github.com/juju/juju/api/client/resources"
	"github.com/juju/juju/testcharms"
	"github.com/juju/version/v2"
	gc "gopkg.in/check.v1"

	"github.com/canonical/jimm/v3/internal/testutils/jimmtest"
)

// localCharmSuite tests end-to-end deployment of a local charm.
type localCharmSuite struct {
	jimmtest.WebsocketE2ESuite
}

var _ = gc.Suite(&localCharmSuite{})

func (s *localCharmSuite) TestLocalCharmDeploy(c *gc.C) {
	conn := s.Open(c, &api.Info{
		ModelTag:  s.Model.ResourceTag(),
		SkipLogin: false,
	}, s.AdminUser.Name, nil)

	client, err := charms.NewLocalCharmClient(conn)
	c.Assert(err, gc.IsNil)
	charmArchive := testcharms.Repo.CharmArchive(c.MkDir(), "dummy")
	curl := charm.MustParseURL(
		fmt.Sprintf("local:quantal/%s-%d", charmArchive.Meta().Name, charmArchive.Revision()),
	)
	vers := version.MustParse("2.6.6")
	url, err := client.AddLocalCharm(curl, charmArchive, false, vers)
	c.Assert(err, gc.IsNil)
	c.Assert(url.String(), gc.Equals, curl.String())
}

func (s *localCharmSuite) TestResourceEndpoint(c *gc.C) {
	// setup: app and pending resource
	conn := s.Open(c, &api.Info{
		ModelTag:  s.Model.ResourceTag(),
		SkipLogin: false,
	}, s.AdminUser.Name, nil)

	charmClient, err := charms.NewLocalCharmClient(conn)
	c.Assert(err, gc.IsNil)

	charmArchive := testcharms.Repo.CharmArchive(c.MkDir(), "dummy")
	curl := charm.MustParseURL(
		fmt.Sprintf("local:quantal/%s-%d", charmArchive.Meta().Name, charmArchive.Revision()),
	)
	url, err := charmClient.AddLocalCharm(curl, charmArchive, false, version.MustParse("2.6.6"))
	c.Assert(err, gc.IsNil)

	appName := "test-app"

	uploadClient, err := resources.NewClient(conn)
	c.Assert(err, gc.IsNil)

	pendingIDs, err := uploadClient.AddPendingResources(resources.AddPendingResourcesArgs{
		ApplicationID: appName,
		CharmID:       resources.CharmID{URL: url.String()},
		Resources: []resource.Resource{
			{
				Meta:   resource.Meta{Name: "test", Type: 1, Path: "file"},
				Origin: resource.OriginStore,
			},
		},
	})
	c.Assert(err, gc.IsNil)
	c.Assert(pendingIDs, gc.HasLen, 1)
	pendingId := pendingIDs[0]

	// act
	err = uploadClient.Upload(appName, "test", "file", pendingId, strings.NewReader("<data>"))
	c.Assert(err, gc.IsNil)
}

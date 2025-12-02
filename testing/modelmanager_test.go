// Copyright 2025 Canonical.

package testing

import (
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/juju/juju/api/base"
	"github.com/juju/juju/api/client/modelmanager"
	"github.com/juju/juju/core/life"
	"github.com/juju/juju/core/status"
	"github.com/juju/juju/state"
	gc "gopkg.in/check.v1"

	"github.com/canonical/jimm/v3/internal/testutils/jimmtest"
)

type modelE2eManagerSuite struct {
	jimmtest.WebsocketE2ESuite
}

var _ = gc.Suite(&modelE2eManagerSuite{})

func (s *modelE2eManagerSuite) TestListModelSummaries(c *gc.C) {
	conn := s.Open(c, nil, "bob")
	defer conn.Close()

	client := modelmanager.NewClient(conn)
	models, err := client.ListModelSummaries("bob@canonical.com", true)
	c.Assert(err, gc.Equals, nil)
	c.Assert(models, jimmtest.CmpEquals(
		cmpopts.IgnoreTypes(&time.Time{}),
		cmpopts.IgnoreFields(base.UserModelSummary{}, "DefaultSeries", "AgentVersion"),
		cmpopts.SortSlices(func(a, b base.UserModelSummary) bool {
			return a.Name < b.Name
		}),
	), []base.UserModelSummary{{
		Name:            "model-1",
		UUID:            s.Model.UUID.String,
		ControllerUUID:  jimmtest.ControllerUUID,
		ProviderType:    jimmtest.TestE2EProviderType,
		DefaultSeries:   "jammy",
		Cloud:           jimmtest.TestE2ECloudName,
		CloudRegion:     jimmtest.TestE2ECloudName,
		CloudCredential: jimmtest.TestE2ECloudName + "/bob@canonical.com/cred",
		Owner:           "bob@canonical.com",
		Life:            life.Value(state.Alive.String()),
		Status: base.Status{
			Status: status.Available,
			Data:   map[string]interface{}{},
		},
		Counts:          []base.EntityCount{},
		ModelUserAccess: "admin",
		Type:            "iaas",
		SLA: &base.SLASummary{
			Level: "",
			Owner: "bob@canonical.com",
		},
	}})
}

package db_test

import (
	"context"

	"github.com/canonical/jimm/v3/internal/dbmodel"
	"github.com/canonical/jimm/v3/internal/errors"
	"github.com/canonical/jimm/v3/internal/testutils/jimmtest"
	qt "github.com/frankban/quicktest"
	"github.com/juju/juju/caas/kubernetes/provider/proxy"
)

func (s *dbSuite) TestAddControllerProxy(c *qt.C) {
	err := s.Database.Migrate(context.Background())
	c.Assert(err, qt.IsNil)

	cloud := dbmodel.Cloud{
		Name: "test-cloud",
	}
	err = s.Database.AddCloud(context.Background(), &cloud)
	c.Assert(err, qt.IsNil)

	controller := &dbmodel.Controller{
		Name:      "test-controller",
		UUID:      "00000000-0000-0000-0000-0000-0000000000001",
		CloudName: "test-cloud",
	}
	err = s.Database.AddController(context.Background(), controller)
	c.Assert(err, qt.Equals, nil)

	controllerProxy := dbmodel.ControllerProxy{
		ControllerId: controller.ID,
		Type:         proxy.ProxierTypeKey,
		Config: map[string]interface{}{
			"api-host": "https://local",
		},
	}

	err = s.Database.AddControllerProxy(c.Context(), controllerProxy)
	c.Assert(err, qt.IsNil)

	storedProxy, err := s.Database.GetControllerProxy(c.Context(), controllerProxy.ControllerId)
	c.Assert(err, qt.IsNil)
	c.Assert(storedProxy, jimmtest.DBObjectEquals, &controllerProxy)
}

func (s *dbSuite) TestGetControllerProxy_NotFound(c *qt.C) {
	err := s.Database.Migrate(context.Background())
	c.Assert(err, qt.IsNil)

	_, err = s.Database.GetControllerProxy(c.Context(), 999)
	c.Assert(errors.ErrorCode(err), qt.Equals, errors.CodeNotFound)
}

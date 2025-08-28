// Copyright 2025 Canonical.

package db_test

import (
	"context"

	"github.com/canonical/jimm/v3/internal/dbmodel"
	"github.com/canonical/jimm/v3/internal/errors"
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

	proxyType := proxy.ProxierTypeKey
	config := map[string]interface{}{
		"api-host": "https://local",
	}

	err = s.Database.PutControllerProxy(c.Context(), controller.Name, proxyType, config)
	c.Assert(err, qt.IsNil)

	storedType, storedConfig, err := s.Database.GetControllerProxy(c.Context(), controller.Name)
	c.Assert(err, qt.IsNil)
	c.Assert(storedType, qt.Equals, proxyType)
	c.Assert(storedConfig, qt.DeepEquals, config)
}

func (s *dbSuite) TestGetControllerProxy_NotFound(c *qt.C) {
	err := s.Database.Migrate(context.Background())
	c.Assert(err, qt.IsNil)

	_, _, err = s.Database.GetControllerProxy(c.Context(), "non-existent-controller")
	c.Assert(errors.ErrorCode(err), qt.Equals, errors.CodeNotFound)
}

// Copyright 2026 Canonical.

package jujuclient

import (
	"context"

	"github.com/juju/juju/api/controller/controller"
	jujucontroller "github.com/juju/juju/controller"
)

// ControllerConfig retrieves the controller configuration.
func (c Connection) ControllerConfig(ctx context.Context) (jujucontroller.Config, error) {
	return controller.NewClient(&c).ControllerConfig(ctx)
}

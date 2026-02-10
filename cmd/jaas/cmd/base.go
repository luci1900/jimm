package cmd

import (
	"fmt"

	"github.com/canonical/jimm/v3/pkg/api"
	"github.com/juju/juju/cmd/modelcmd"
)

type JAASCommand interface {
	modelcmd.ControllerCommand

	SetJIMMAPI(JIMMAPI)
	JIMMAPI() (JIMMAPI, error)
}

type JAASCommandBase struct {
	modelcmd.ControllerCommandBase
	jimmAPI JIMMAPI
}

func (c *JAASCommandBase) SetJIMMAPI(api JIMMAPI) {
	c.jimmAPI = api
}

func (c *JAASCommandBase) JIMMAPI() (JIMMAPI, error) {
	if c.jimmAPI != nil {
		return c.jimmAPI, nil
	}

	currentController, err := c.ClientStore().CurrentController()
	if err != nil {
		return nil, fmt.Errorf("could not determine controller: %w", err)
	}

	apiCaller, err := c.NewAPIRootWithDialOpts(c.ClientStore(), currentController, "", nil)
	if err != nil {
		return nil, err
	}

	return api.NewClient(apiCaller), nil
}

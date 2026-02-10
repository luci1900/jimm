package cmd

import (
	"fmt"

	jujuapi "github.com/juju/juju/api"
	"github.com/juju/juju/cmd/modelcmd"

	"github.com/canonical/jimm/v3/pkg/api"
)

type JAASCommand interface {
	modelcmd.ControllerCommand

	SetJIMMAPI(JIMMAPI)
	JIMMAPI() (JIMMAPI, error)
}

type JAASCommandBase struct {
	modelcmd.ControllerCommandBase
	jimmAPI JIMMAPI
	dialOpts *jujuapi.DialOpts
}

func (c *JAASCommandBase) SetJIMMAPI(api JIMMAPI) {
	c.jimmAPI = api
}

func (c *JAASCommandBase) SetDialOpts(dialOpts *jujuapi.DialOpts) {
	c.dialOpts = dialOpts
}

func (c *JAASCommandBase) JIMMAPI() (JIMMAPI, error) {
	return c.JIMMAPIWithController("")
}

func (c *JAASCommandBase) JIMMAPIWithController(controller string) (JIMMAPI, error) {
	if c.jimmAPI != nil {
		return c.jimmAPI, nil
	}

	currentController := controller
	if currentController == "" {
		var err error
		currentController, err = c.ClientStore().CurrentController()
		if err != nil {
			return nil, fmt.Errorf("could not determine controller: %w", err)
		}
	}

	apiCaller, err := c.NewAPIRootWithDialOpts(c.ClientStore(), currentController, "", c.dialOpts)
	if err != nil {
		return nil, err
	}

	return api.NewClient(apiCaller), nil
}

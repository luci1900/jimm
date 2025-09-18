// Copyright 2025 Canonical.

package cmd

import (
	"github.com/juju/cmd/v3"
	"github.com/juju/gnuflag"
	jujuapi "github.com/juju/juju/api"
	jujucmd "github.com/juju/juju/cmd"
	"github.com/juju/juju/cmd/modelcmd"
	"github.com/juju/juju/jujuclient"

	"github.com/canonical/jimm/v3/internal/errors"
	jimmAPI "github.com/canonical/jimm/v3/pkg/api"
	"github.com/canonical/jimm/v3/pkg/api/params"
)

const (
	destroyControllerCommandDoc = `
`
	destroyControllerCommandExample = `
`
)

func NewDestroyControllerCommand() cmd.Command {
	cmd := &destroyControllerCommand{
		store: jujuclient.NewFileClientStore(),
	}
	return modelcmd.WrapBase(cmd)
}

// destroyControllerCommand
type destroyControllerCommand struct {
	modelcmd.ControllerCommandBase

	store          jujuclient.ClientStore
	dialOpts       *jujuapi.DialOpts
	controllerName string
}

// Info implements cmd.Info interface
func (c *destroyControllerCommand) Info() *cmd.Info {
	return jujucmd.Info(&cmd.Info{
		Name:     "destroy-controller",
		Args:     "<controller-name>",
		Purpose:  "",
		Doc:      destroyControllerCommandDoc,
		Examples: destroyControllerCommandExample,
	})
}

// SetFlags implements cmd.SetFlags interface
func (c *destroyControllerCommand) SetFlags(f *gnuflag.FlagSet) {
	c.CommandBase.SetFlags(f)
}

// Init implements the cmd.Command interface
func (c *destroyControllerCommand) Init(args []string) error {
	if len(args) < 1 {
		return errors.E("missing controller name")
	}

	c.controllerName, args = args[0], args[1:]
	if len(args) > 0 {
		return errors.E("unknown arguments")
	}
	return nil
}

// Run implements cmd.Command.Run interface
func (c *destroyControllerCommand) Run(ctx *cmd.Context) error {
	// TODO
	_, err := c.store.ControllerByName(c.controllerName)
	if err != nil {
		return err
	}

	apiCaller, err := c.NewAPIRootWithDialOpts(c.store, c.controllerName, "", c.dialOpts)
	if err != nil {
		return err
	}

	client := jimmAPI.NewClient(apiCaller)
	defer client.Close()

	err = client.DestroyController(&params.DestroyControllerRequest{
		Name: c.controllerName,
	})
	if err != nil {
		return err
	}

	return nil
}

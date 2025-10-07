// Copyright 2025 Canonical.

package cmd

import (
	"context"

	"github.com/juju/cmd/v3"
	"github.com/juju/cmd/v3/cmdtesting"
	"github.com/juju/gnuflag"
	"github.com/juju/juju/jujuclient/jujuclienttesting"
	"go.uber.org/mock/gomock"
	gc "gopkg.in/check.v1"

	"github.com/canonical/jimm/v3/cmd/jaas/cmd/mocks"
	"github.com/canonical/jimm/v3/pkg/api/params"
)

type destroyControllerCmdSuite struct {
	client *mocks.MockJIMMAPI
	writer *mocks.MockWriter
	store  *mocks.MockClientStore
}

var _ = gc.Suite(&destroyControllerCmdSuite{})

func (s *destroyControllerCmdSuite) SetupMocks(c *gc.C) *gomock.Controller {
	ctrl := gomock.NewController(c)
	s.client = mocks.NewMockJIMMAPI(ctrl)
	s.writer = mocks.NewMockWriter(ctrl)
	s.store = mocks.NewMockClientStore(ctrl)

	return ctrl
}

func (s *destroyControllerCmdSuite) TestArgParsing(c *gc.C) {
	tests := []struct {
		args       []string
		checkFlags func(*gc.C, *destroyControllerCommand)
		errMatch   string
	}{
		{
			args: []string{"controller-name"},
			checkFlags: func(c *gc.C, command *destroyControllerCommand) {
				c.Check(command.controllerName, gc.Equals, "controller-name")
			},
		},
		{
			args: []string{"controller-name"},
			checkFlags: func(c *gc.C, command *destroyControllerCommand) {
				c.Check(command.controllerName, gc.Equals, "controller-name")
			},
		}, {
			args: []string{"controller-name", "--detach"},
			checkFlags: func(c *gc.C, command *destroyControllerCommand) {
				c.Check(command.controllerName, gc.Equals, "controller-name")
				c.Check(command.detach, gc.Equals, true)
			},
		}, {
			args:     []string{},
			errMatch: "missing controller name",
		},
	}
	for i, test := range tests {
		c.Log("Test ", i)
		command := &destroyControllerCommand{}
		command.SetClientStore(jujuclienttesting.MinimalStore())
		command.noPrompt = true
		err := cmdtesting.InitCommand(command, test.args)
		if test.errMatch == "" {
			c.Check(err, gc.IsNil)
			test.checkFlags(c, command)
		} else {
			c.Check(err, gc.ErrorMatches, test.errMatch)
		}
	}
}

func (s *destroyControllerCmdSuite) TestRunDetached(c *gc.C) {
	ctrl := s.SetupMocks(c)
	defer ctrl.Finish()

	s.client.EXPECT().StartDestroyControllerJob(gomock.Any()).DoAndReturn(func(bsp *params.DestroyControllerRequest) (*params.StartJobResponse, error) {
		expected := &params.DestroyControllerRequest{
			ControllerName: "controller-name",
		}
		c.Assert(bsp.ControllerName, gc.Equals, expected.ControllerName)

		return &params.StartJobResponse{
			JobID: "test-job-id",
		}, nil
	})
	s.client.EXPECT().Close().Return(nil)

	command := &destroyControllerCommand{
		store: s.store,
		destroyControllerAPIFunc: func() (JIMMAPI, error) {
			return s.client, nil
		},
	}

	f := gnuflag.NewFlagSet("test", gnuflag.ExitOnError)
	f.SetOutput(s.writer)
	command.SetFlags(f)
	command.controllerName = "controller-name"
	command.detach = true
	command.noPrompt = true

	ctx := &cmd.Context{
		Context: context.Background(),
		Stdout:  s.writer,
	}

	err := command.Run(ctx)
	c.Assert(err, gc.IsNil)
}

func (s *destroyControllerCmdSuite) TestWatchLogs(c *gc.C) {
	ctrl := s.SetupMocks(c)
	defer ctrl.Finish()

	s.client.EXPECT().StartDestroyControllerJob(gomock.Any()).Return(&params.StartJobResponse{
		JobID: "test-job-id",
	}, nil)
	s.client.EXPECT().Close().Return(nil)

	s.client.EXPECT().GetJobInfo(gomock.Any()).Return(params.GetJobInfoResponse{
		Status:    params.StatusSuccessful,
		Logs:      []string{"log-line", "log-line"},
		Watermark: 2,
	}, nil)

	s.writer.EXPECT().Write(gomock.Any()).DoAndReturn(func(b []byte) (int, error) {
		c.Check(string(b), gc.Equals, "log-line\n")
		return len(b), nil
	}).Times(2)

	s.writer.EXPECT().Write(gomock.Any()).DoAndReturn(func(b []byte) (int, error) {
		c.Check(string(b), gc.Equals, "Job completed successfully.\n")
		return len(b), nil
	})

	command := &destroyControllerCommand{
		store: s.store,
		destroyControllerAPIFunc: func() (JIMMAPI, error) {
			return s.client, nil
		},
	}
	f := gnuflag.NewFlagSet("test", gnuflag.ExitOnError)
	f.SetOutput(s.writer)
	command.SetFlags(f)
	command.noPrompt = true

	ctx := &cmd.Context{
		Context: context.Background(),
		Stdout:  s.writer,
	}

	err := command.Run(ctx)
	c.Assert(err, gc.IsNil)
}

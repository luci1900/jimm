// Copyright 2026 Canonical.

package cmd

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"go.uber.org/mock/gomock"

	apiparams "github.com/canonical/jimm/v3/pkg/api/params"
)

func TestListJobsRun_FlagsArePassedToAPICorrectly(t *testing.T) {
	c := qt.New(t)

	cmdMocks := setupCmdMocks(c)

	expected := &apiparams.ListJobsResponse{
		Jobs: []apiparams.ListJobInfo{},
	}

	cmdMocks.client.EXPECT().
		ListJobs(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req *apiparams.ListJobsRequest) (*apiparams.ListJobsResponse, error) {
			c.Check(req.Count, qt.Equals, 500)
			c.Check(req.Kinds, qt.DeepEquals, []string{"backup", "restore"})
			c.Check(req.Statuses, qt.DeepEquals, []apiparams.JobStatus{
				apiparams.StatusRunning,
				apiparams.StatusPending,
			})
			return expected, nil
		}).
		Times(1)

	cmdMocks.client.EXPECT().Close()

	command := &listjobsCommand{}
	command.setJIMMAPI(cmdMocks.client)

	initCommand(c, command, "--count=500", "--kind=backup", "--kind=restore", "--status=running", "--status=pending")

	err := command.Run(newTestContext(c))
	c.Assert(err, qt.IsNil)
}

func TestListJobsRun_CountValidation(t *testing.T) {
	c := qt.New(t)

	cmdMocks := setupCmdMocks(c)

	command := &listjobsCommand{}
	command.setJIMMAPI(cmdMocks.client)

	initCommand(c, command, "--count=10001")

	err := command.Run(newTestContext(c))
	c.Assert(err, qt.ErrorMatches, "count cannot exceed 10000, got 10001")
}

func TestListJobsRun_StatusValidation(t *testing.T) {
	c := qt.New(t)

	cmdMocks := setupCmdMocks(c)
	cmdMocks.client.EXPECT().Close()

	command := &listjobsCommand{}
	command.setJIMMAPI(cmdMocks.client)

	initCommand(c, command, "--status=invalid")

	err := command.Run(newTestContext(c))
	c.Assert(err, qt.ErrorMatches, `invalid status "invalid", must be one of: running, successful, pending, failed, unknown`)
}

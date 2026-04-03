// Copyright 2026 Canonical.

package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/juju/gnuflag"
	"github.com/juju/juju/api/jujuclient"
	jujucmd "github.com/juju/juju/cmd"
	"github.com/juju/juju/cmd/cmd"
	"github.com/juju/juju/cmd/modelcmd"

	"github.com/canonical/jimm/v3/pkg/api/params"
)

const (
	bootstrapStatusCommandDoc = `
Displays logs for a bootstrap or destroy-controller job.
`
	bootstrapStatusCommandExample = `
    juju bootstrap-status <id>
    juju destroy-status <id>
`
)

// sleepBetweenGetLogs is the duration to wait between successive calls to get logs for a job.
const sleepBetweenGetLogs = 1 * time.Second

// NewBootstrapStatusCommand returns a command to display logs for a job.
func NewBootstrapStatusCommand() cmd.Command {
	cmd := &bootstrapStatusCommand{}
	cmd.SetClientStore(jujuclient.NewFileClientStore())

	return modelcmd.WrapBase(cmd)
}

// bootstrapStatusCommand displays logs for a job.
type bootstrapStatusCommand struct {
	jaasCommandBase

	jobId string

	sleepBetweenGetLogs time.Duration
	follow              bool
}

// Info implements cmd.Info interface.
func (c *bootstrapStatusCommand) Info() *cmd.Info {
	return jujucmd.Info(&cmd.Info{
		Name:     "bootstrap-status",
		Aliases:  []string{"destroy-status"},
		Args:     "<job id>",
		Purpose:  "Displays logs for a bootstrap/destroy job",
		Doc:      bootstrapStatusCommandDoc,
		Examples: bootstrapStatusCommandExample,
	})
}

// SetFlags implements cmd.SetFlags interface.
func (c *bootstrapStatusCommand) SetFlags(f *gnuflag.FlagSet) {
	c.CommandBase.SetFlags(f)
	f.BoolVar(&c.follow, "f", false, "follow the logs")
}

// Init implements the cmd.Command interface.
func (c *bootstrapStatusCommand) Init(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing job id")
	}
	c.jobId, args = args[0], args[1:]
	if len(args) > 0 {
		return fmt.Errorf("unknown arguments")
	}

	c.sleepBetweenGetLogs = sleepBetweenGetLogs
	return nil
}

// Run implements cmd.Command.Run interface.
func (c *bootstrapStatusCommand) Run(ctxt *cmd.Context) error {
	client, err := c.getJIMMAPI(ctxt)
	if err != nil {
		return fmt.Errorf("failed to create JIMM client: %v", err)
	}
	defer client.Close()

	poller := bootstrapLogPoller{
		client:              client,
		jobId:               c.jobId,
		sleepBetweenGetLogs: c.sleepBetweenGetLogs,
		out:                 ctxt.Stdout,
		follow:              c.follow,
	}
	return poller.watchBootstrapLogs(ctxt)
}

type bootstrapLogPoller struct {
	client              JIMMAPI
	jobId               string
	sleepBetweenGetLogs time.Duration
	out                 io.Writer
	follow              bool
}

func (p bootstrapLogPoller) watchBootstrapLogs(ctxt *cmd.Context) error {
	watermark := 0

	for {
		response, err := p.client.BootstrapInfo(
			ctxt,
			&params.GetBootstrapInfoRequest{
				JobID:     p.jobId,
				Watermark: watermark,
			})
		if err != nil {
			return fmt.Errorf("failed to get info: %w", err)
		}
		for _, log := range response.Logs {
			_, err = p.out.Write([]byte(log + "\n"))
			if err != nil {
				return fmt.Errorf("failed to write log: %w", err)
			}
		}
		watermark = response.Watermark
		switch response.Status {
		case params.StatusRunning:
			// If the job is still running, we just continue to the next iteration.
		case params.StatusSuccessful:
			_, err = p.out.Write([]byte("Job completed successfully.\n"))
			if err != nil {
				return fmt.Errorf("failed to write success message: %w", err)
			}
			return nil
		case params.StatusFailed:
			return fmt.Errorf("job failed: %s", response.Error)
		case params.StatusPending:
			_, err := p.out.Write([]byte("Job is pending...\n"))
			if err != nil {
				return fmt.Errorf("failed to write pending message: %w", err)
			}
		default:
			return fmt.Errorf("unknown job status: %s", response.Status)
		}
		if !p.follow {
			return nil
		}
		time.Sleep(p.sleepBetweenGetLogs)
	}
}

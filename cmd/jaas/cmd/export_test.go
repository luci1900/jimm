// Copyright 2026 Canonical.

package cmd

import (
	"github.com/juju/cmd/v3"
	jujuapi "github.com/juju/juju/api"
	"github.com/juju/juju/cmd/modelcmd"
	"github.com/juju/juju/jujuclient"

	"github.com/canonical/jimm/v3/internal/testutils/cmdtest"
)

var (
	AccessMessage          = accessMessageFormat
	AccessResultAllowed    = accessResultAllowed
	AccessResultDenied     = accessResultDenied
	DefaultPageSize        = defaultPageSize
	FormatRelationsTabular = formatRelationsTabular
)

type AccessResult = accessResult

func NewRevokeAuditLogAccessCommandForTesting(store jujuclient.ClientStore, lp jujuapi.LoginProvider) cmd.Command {
	cmd := &revokeAuditLogAccessCommand{
		dialOpts: cmdtest.TestDialOpts(lp),
	}
	cmd.SetClientStore(store)

	return modelcmd.WrapBase(cmd)
}

# Dialing backing controllers

JIMM dials backing controllers along two axes: connection scope (model vs
controller) and identity (real user vs JIMM's service identity). Prefer
dialing as the real user wherever possible, since every AsService call is
a potential gap to close in Juju's permission model. See the `Dialer`
interface godoc for method semantics.

## Remaining dial-as-service uses

### Internal housekeeping

Not user-triggered in the first place.

- `watcher.go`: model-summary watcher
- `model_poller.go`: migration-mode polling
- `jimm.go` `forEachController`: fan-out across controllers
- `upgrade/upgrade.go`: automated model upgrade worker
- `controller.go` `AddController`: registering a new controller

### Migration machinery 

User-triggered, but JIMM is the acting party.

- `migrationtarget.go`: Abort, CheckMachines, Import, AdoptResources,
  LatestLogTime, Activate, StageImport
- `controller.go`: fetchModelInfo, post-migration success check,
  InitiateMigration
- `jujuapi/streamcontrollerproxy.go`: migration log-transfer stream

### JIMM-admin operations

Authorized by JIMM, but the admin holds no OpenFGA relations on the
backing controller's resources, so there are no claims to mint.

- `jimm.go` `FullModelStatus`
- `model.go` `UpgradeController`: upgrades the controller model, which
  is never related to users in OpenFGA
- `controller.go` `ControllerConfig`
- `model_status_parser.go`

### Workarounds 

JIMM enforces authorization, but the backing facade needs more than
the caller holds.

- `cloud.go`: add/manage/remove hosted clouds
- `model.go` `ChangeModelCredential`: forced credential update
  (`force=true` requires controller superuser)
- `modelbuilder.go`: forced credential update on model create.
  `GrantJIMMModelAdmin` (JUJU-8869)
- `applicationoffer.go` `queryControllersForOffers`: spans many models,
  a single caller-scoped JWT can't satisfy Juju's per-model check.
  Results re-authorized via OpenFGA


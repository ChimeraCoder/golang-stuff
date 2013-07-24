// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package machine

import (
	"launchpad.net/juju-core/state"
	"launchpad.net/juju-core/state/api/params"
	"launchpad.net/juju-core/state/apiserver/common"
	"launchpad.net/juju-core/state/watcher"
)

// MachinerAPI implements the API used by the machiner worker.
type MachinerAPI struct {
	*common.LifeGetter
	st        *state.State
	resources *common.Resources
	auth      common.Authorizer
}

// NewMachinerAPI creates a new instance of the Machiner API.
func NewMachinerAPI(st *state.State, resources *common.Resources, authorizer common.Authorizer) (*MachinerAPI, error) {
	if !authorizer.AuthMachineAgent() {
		return nil, common.ErrPerm
	}
	getCanRead := func() (common.AuthFunc, error) {
		return func(tag string) bool {
			// TODO(go1.1): method expression
			return authorizer.AuthOwner(tag)
		}, nil
	}
	return &MachinerAPI{
		LifeGetter: common.NewLifeGetter(st, getCanRead),
		st:         st,
		resources:  resources,
		auth:       authorizer,
	}, nil
}

// SetStatus sets the status of each given machine.
func (m *MachinerAPI) SetStatus(args params.MachinesSetStatus) (params.ErrorResults, error) {
	result := params.ErrorResults{
		Errors: make([]*params.Error, len(args.Machines)),
	}
	if len(args.Machines) == 0 {
		return result, nil
	}
	for i, arg := range args.Machines {
		err := common.ErrPerm
		if m.auth.AuthOwner(arg.Tag) {
			var machine *state.Machine
			machine, err = m.st.Machine(state.MachineIdFromTag(arg.Tag))
			if err == nil {
				err = machine.SetStatus(arg.Status, arg.Info)
			}
		}
		result.Errors[i] = common.ServerError(err)
	}
	return result, nil
}

// Watch starts an NotifyWatcher for each given machine.
func (m *MachinerAPI) Watch(args params.Entities) (params.NotifyWatchResults, error) {
	result := params.NotifyWatchResults{
		Results: make([]params.NotifyWatchResult, len(args.Entities)),
	}
	if len(args.Entities) == 0 {
		return result, nil
	}
	for i, entity := range args.Entities {
		err := common.ErrPerm
		if m.auth.AuthOwner(entity.Tag) {
			var machine *state.Machine
			machine, err = m.st.Machine(state.MachineIdFromTag(entity.Tag))
			if err == nil {
				watch := machine.Watch()
				// Consume the initial event. Technically, API
				// calls to Watch 'transmit' the initial event
				// in the Watch response. But NotifyWatchers
				// have no state to transmit.
				if _, ok := <-watch.Changes(); ok {
					result.Results[i].NotifyWatcherId = m.resources.Register(watch)
				} else {
					err = watcher.MustErr(watch)
				}
			}
		}
		result.Results[i].Error = common.ServerError(err)
	}
	return result, nil
}

// EnsureDead changes the lifecycle of each given machine to Dead if
// it's Alive or Dying. It does nothing otherwise.
func (m *MachinerAPI) EnsureDead(args params.Entities) (params.ErrorResults, error) {
	result := params.ErrorResults{
		Errors: make([]*params.Error, len(args.Entities)),
	}
	if len(args.Entities) == 0 {
		return result, nil
	}
	for i, entity := range args.Entities {
		err := common.ErrPerm
		if m.auth.AuthOwner(entity.Tag) {
			var machine *state.Machine
			machine, err = m.st.Machine(state.MachineIdFromTag(entity.Tag))
			if err == nil {
				err = machine.EnsureDead()
			}
		}
		result.Errors[i] = common.ServerError(err)
	}
	return result, nil
}

// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package commands

import (
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/event"
)

type MinimalCommandHandler interface {
	Run(*Event)
}

type MinimalCommandHandlerFunc func(*Event)

func (mhf MinimalCommandHandlerFunc) Run(ce *Event) {
	mhf(ce)
}

type CommandState struct {
	Next   MinimalCommandHandler
	Action string
	Meta   any
	Cancel func()
}

type CommandHandler interface {
	MinimalCommandHandler
	GetName() string
}

type AliasedCommandHandler interface {
	CommandHandler
	GetAliases() []string
}

func NetworkAPIImplements[T bridgev2.NetworkAPI](val bridgev2.NetworkAPI) bool {
	_, ok := val.(T)
	return ok
}

func NetworkConnectorImplements[T bridgev2.NetworkConnector](val bridgev2.NetworkConnector) bool {
	_, ok := val.(T)
	return ok
}

type ImplementationChecker[T any] func(val T) bool

type FullHandler struct {
	Func func(*Event)

	Name    string
	Aliases []string
	Help    HelpMeta

	RequiresAdmin           bool
	RequiresPortal          bool
	RequiresLogin           bool
	RequiresEventLevel      event.Type
	RequiresLoginPermission bool

	NetworkAPI       ImplementationChecker[bridgev2.NetworkAPI]
	NetworkConnector ImplementationChecker[bridgev2.NetworkConnector]
}

func (fh *FullHandler) GetHelp() HelpMeta {
	fh.Help.Command = fh.Name
	return fh.Help
}

func (fh *FullHandler) GetName() string {
	return fh.Name
}

func (fh *FullHandler) GetAliases() []string {
	return fh.Aliases
}

func (fh *FullHandler) ImplementationsFulfilled(ce *Event) bool {
	// TODO add dedicated method to get an empty NetworkAPI instead of getting default login
	client := ce.User.GetDefaultLogin()
	return (fh.NetworkAPI == nil || client == nil || fh.NetworkAPI(client.Client)) &&
		(fh.NetworkConnector == nil || fh.NetworkConnector(ce.Bridge.Network))
}

func (fh *FullHandler) ShowInHelp(ce *Event) bool {
	return fh.ImplementationsFulfilled(ce) && (!fh.RequiresAdmin || ce.User.Permissions.Admin)
}

func (fh *FullHandler) userHasRoomPermission(ce *Event) bool {
	levels, err := ce.Bridge.Matrix.GetPowerLevels(ce.Ctx, ce.RoomID)
	if err != nil {
		ce.Log.Warn().Err(err).Msg("Failed to check room power levels")
		ce.Reply("Falha ao obter os níveis de poder da sala para verificar se você tem permissão para usar esse comando.")
		return false
	}
	return levels.GetUserLevel(ce.User.MXID) >= levels.GetEventLevel(fh.RequiresEventLevel)
}

func (fh *FullHandler) Run(ce *Event) {
	if fh.RequiresAdmin && !ce.User.Permissions.Admin {
		ce.Reply("Esse comando é restrito a administradores da ponte.")
	} else if fh.RequiresLoginPermission && !ce.User.Permissions.Login {
		ce.Reply("Você não tem permissão para fazer login nesta ponte.")
	} else if fh.RequiresEventLevel.Type != "" && !ce.User.Permissions.Admin && !fh.userHasRoomPermission(ce) {
		ce.Reply("Esse comando requer direitos de administrador da sala.")
	} else if fh.RequiresPortal && ce.Portal == nil {
		ce.Reply("Esse comando só pode ser executado em salas de portal.")
	} else if fh.RequiresLogin && ce.User.GetDefaultLogin() == nil {
		ce.Reply("Esse comando requer que você esteja logado.")
	} else {
		fh.Func(ce)
	}
}

// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package tray

import "github.com/alisaitteke/sideguard/internal/api"

// Controller is a backward-compatible alias for Session during the mtp migration.
type Controller = Session

// NewController is a backward-compatible alias for NewSession.
func NewController(client *api.Client) *Session {
	return NewSession(client)
}

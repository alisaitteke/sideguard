// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package shell

// Prepare is the top-level entry point for the detect engine: it normalizes the
// command, statically deobfuscates it (revealing hidden payloads), and parses
// the result into an IR. IR.Raw is preserved as the normalized ORIGINAL command
// (not the deobfuscated string) so callers can display what the user typed while
// still seeing decoded intent in Stages/Substitutions/NestedCommands.
//
// No stage executes anything — this is pure static analysis.
//
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-1.0-shell.md).
func Prepare(command string) (IR, DeobfuscateMeta, error) {
	normalized := Normalize(command)
	deobfuscated, meta := Deobfuscate(normalized)

	ir, err := Parse(deobfuscated)
	// Keep Raw as the normalized original so display/audit shows the real input.
	ir.Raw = normalized
	return ir, meta, err
}

// status_popover.h — C API for macOS NSStatusItem + NSPopover tray shell.
// See docs/plans/2026-07-01-1537-mac-tray-popover/ (mtp-phase-3.0-darwin-panel.md).

#ifndef VIBEGUARD_STATUS_POPOVER_H
#define VIBEGUARD_STATUS_POPOVER_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// on_ready is invoked on the main thread after NSStatusItem and NSPopover are set up.
typedef void (*darwin_ready_fn)(void);

// darwin_tray_prepare configures NSApplication (call before darwin_tray_run_loop).
void darwin_tray_prepare(darwin_ready_fn on_ready);

// darwin_tray_run_loop blocks until darwin_quit terminates NSApplication.
void darwin_tray_run_loop(void);

// darwin_set_icon updates the status-item image from PNG bytes (main-thread dispatch).
void darwin_set_icon(const unsigned char *data, size_t len);

// darwin_set_title sets the menu-bar title next to the icon (may be empty).
void darwin_set_title(const char *title);

// darwin_set_tooltip sets the status-item tooltip.
void darwin_set_tooltip(const char *tooltip);

// darwin_update_panel rebuilds popover content from a JSON snapshot (main-thread dispatch).
// Schema: daemon_status, pending_count, mode_index, mode_enabled, rows[{id,label}],
// overflow_hint, empty_message.
void darwin_update_panel(const char *json);

// darwin_show_popover opens the popover below the status item if hidden (no-op when shown).
void darwin_show_popover(void);

// darwin_quit terminates the NSApplication run loop.
void darwin_quit(void);

#ifdef __cplusplus
}
#endif

#endif /* VIBEGUARD_STATUS_POPOVER_H */

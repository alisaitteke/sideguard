package update

import "context"

// PlatformApplier stops and restarts platform services around a binary swap.
// OS-specific implementations live in apply_{darwin,linux,windows}.go.
// See docs/plans/2026-07-02-1102-github-update/ (vgu-phase-5.0-update-platform.md).
type PlatformApplier interface {
	Stop(ctx context.Context) error
	SwapBinary(ctx context.Context, stagingPath, targetPath string) error
	Start(ctx context.Context) error
}

// NoopPlatformApplier is a test stub: atomic swap only, no stop/start work.
type NoopPlatformApplier struct{}

// Stop is a no-op.
func (NoopPlatformApplier) Stop(context.Context) error { return nil }

// SwapBinary performs a direct atomic rename swap.
func (NoopPlatformApplier) SwapBinary(_ context.Context, stagingPath, targetPath string) error {
	return atomicSwapBinary(stagingPath, targetPath)
}

// Start is a no-op.
func (NoopPlatformApplier) Start(context.Context) error { return nil }

// NewPlatformApplier returns the PlatformApplier for the current GOOS (build-tag dispatch).
func NewPlatformApplier() PlatformApplier {
	return newPlatformApplier()
}

//go:build !darwin && !linux && !windows

package update

func newPlatformApplier() PlatformApplier {
	return NoopPlatformApplier{}
}

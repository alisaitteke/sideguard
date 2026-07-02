package update

import (
	"fmt"
	"os"
)

// atomicSwapBinary copies stagingPath to targetPath.new then renames over targetPath.
func atomicSwapBinary(stagingPath, targetPath string) error {
	newPath := targetPath + ".new"
	if err := copyFile(stagingPath, newPath); err != nil {
		return err
	}
	if err := os.Chmod(newPath, 0o755); err != nil {
		_ = os.Remove(newPath)
		return err
	}
	if err := os.Rename(newPath, targetPath); err != nil {
		_ = os.Remove(newPath)
		return fmt.Errorf("atomic replace: %w", err)
	}
	return nil
}

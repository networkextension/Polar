package dock

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func (s *Server) hydrateSiteSettings(settings *SiteSettings) *SiteSettings {
	if settings == nil {
		return nil
	}

	cloned := *settings
	cloned.SystemInfo = s.collectSystemInfo()
	return &cloned
}

func (s *Server) collectSystemInfo() *SystemInfo {
	partitionPath, capacity := s.runtimePartitionInfo()
	return &SystemInfo{
		GitTagVersion:     s.gitTagVersion(),
		OS:                runtime.GOOS,
		CPUArch:           runtime.GOARCH,
		PartitionPath:     partitionPath,
		PartitionCapacity: capacity,
	}
}

func (s *Server) gitTagVersion() string {
	candidates := []string{}
	if s != nil && strings.TrimSpace(s.workDir) != "" {
		candidates = append(candidates, s.workDir)
	}
	if exePath, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exePath))
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, dir := range candidates {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}

		tag, err := runGitTagCommand(dir, "describe", "--tags", "--abbrev=0")
		if err == nil && tag != "" {
			return tag
		}

		tag, err = runGitTagCommand(dir, "tag", "--sort=-v:refname")
		if err == nil && tag != "" {
			lines := strings.Split(tag, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					return line
				}
			}
		}
	}

	return "未知"
}

func runGitTagCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *Server) runtimePartitionInfo() (string, string) {
	target := "."
	if s != nil && strings.TrimSpace(s.workDir) != "" {
		target = s.workDir
	}
	if exePath, err := os.Executable(); err == nil {
		target = exePath
	}

	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err == nil && resolvedTarget != "" {
		target = resolvedTarget
	}

	infoTarget := target
	if stat, err := os.Stat(target); err == nil && !stat.IsDir() {
		infoTarget = filepath.Dir(target)
	}

	var fsStat syscall.Statfs_t
	if err := syscall.Statfs(infoTarget, &fsStat); err != nil {
		return infoTarget, "未知"
	}

	availableBytes := uint64(fsStat.Bavail) * uint64(fsStat.Bsize)
	return infoTarget, formatBytes(availableBytes)
}

func formatBytes(size uint64) string {
	if size == 0 {
		return "0 B"
	}

	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	value := float64(size)
	unit := units[0]
	for i := 0; i < len(units)-1 && value >= 1024; i++ {
		value /= 1024
		unit = units[i+1]
	}

	if unit == "B" {
		return fmt.Sprintf("%.0f %s", value, unit)
	}
	return fmt.Sprintf("%.2f %s", value, unit)
}

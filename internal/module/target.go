package module

import (
	"os"
	"strings"

	"github.com/Gui97p/wisp/internal/target"
)

func ConfigureTarget(name string) {
	target.SetActive(name)
}

func currentTarget() string {
	return target.Active()
}

func FileDirective(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "//"):
			continue
		case strings.HasPrefix(line, "#target"):
			return strings.TrimSpace(strings.TrimPrefix(line, "#target"))
		}
		break
	}
	return ""
}

package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

func (s *Server) handleGetGlobalEnv(w http.ResponseWriter, r *http.Request) {
	s.envOnce.Do(func() {
		info := make(map[string]any)
		info["os"] = runtime.GOOS
		info["arch"] = runtime.GOARCH
		info["num_cpu"] = runtime.NumCPU()
		info["probed_at"] = time.Now().UTC().Format(time.RFC3339)

		toolchains := make(map[string]string)
		tools := []string{"git", "gh", "go", "node", "pnpm", "python3", "sqlite3"}
		for _, tool := range tools {
			if path, err := exec.LookPath(tool); err == nil {
				toolchains[tool] = path
			} else {
				toolchains[tool] = "not_found"
			}
		}
		info["toolchains"] = toolchains

		hasher := sha256.New()
		_ = json.NewEncoder(hasher).Encode(info)
		info["fingerprint"] = hex.EncodeToString(hasher.Sum(nil))

		s.envInfo = info
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.envInfo)
}

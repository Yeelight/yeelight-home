package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func writeMCPServersJSON(path string, servers []setupMCPServer) error {
	return updateJSONFile(path, func(document map[string]any) error {
		entries, err := objectField(document, "mcpServers")
		if err != nil {
			return err
		}
		for _, server := range servers {
			entries[server.Name] = standardMCPEntry(server)
		}
		document["mcpServers"] = entries
		return nil
	})
}

func writeClaudeDesktopMCPJSON(path string, servers []setupMCPServer) error {
	return updateJSONFile(path, func(document map[string]any) error {
		entries, err := objectField(document, "mcpServers")
		if err != nil {
			return err
		}
		for _, server := range servers {
			if server.isLocal() {
				entries[server.Name] = standardMCPEntry(server)
			} else {
				entries[server.Name] = mcpRemoteEntry(server)
			}
		}
		document["mcpServers"] = entries
		return nil
	})
}

func writeClaudeCodeMCPJSON(path string, servers []setupMCPServer) error {
	return updateJSONFile(path, func(document map[string]any) error {
		entries, err := objectField(document, "mcpServers")
		if err != nil {
			return err
		}
		for _, server := range servers {
			entry := standardMCPEntry(server)
			if server.isLocal() {
				entry["type"] = "stdio"
			} else {
				entry["type"] = "http"
			}
			entries[server.Name] = entry
		}
		document["mcpServers"] = entries
		return nil
	})
}

func writeFactoryDroidMCPJSON(path string, servers []setupMCPServer) error {
	return updateJSONFile(path, func(document map[string]any) error {
		entries, err := objectField(document, "mcpServers")
		if err != nil {
			return err
		}
		for _, server := range servers {
			entry := standardMCPEntry(server)
			if server.isLocal() {
				entry["type"] = "stdio"
			} else {
				entry["type"] = "http"
			}
			entry["disabled"] = false
			entries[server.Name] = entry
		}
		document["mcpServers"] = entries
		return nil
	})
}

func writeVSCodeMCPJSON(path string, servers []setupMCPServer) error {
	return updateJSONFile(path, func(document map[string]any) error {
		entries, err := objectField(document, "servers")
		if err != nil {
			return err
		}
		for _, server := range servers {
			entry := standardMCPEntry(server)
			if server.isLocal() {
				entry["type"] = "stdio"
			} else {
				entry["type"] = "http"
			}
			entries[server.Name] = entry
		}
		document["servers"] = entries
		return nil
	})
}

func writeGeminiMCPJSON(path string, servers []setupMCPServer) error {
	return updateJSONFile(path, func(document map[string]any) error {
		entries, err := objectField(document, "mcpServers")
		if err != nil {
			return err
		}
		for _, server := range servers {
			entry := standardMCPEntry(server)
			if !server.isLocal() {
				delete(entry, "url")
				entry["httpUrl"] = server.URL
				entry["trust"] = false
			}
			entries[server.Name] = entry
		}
		document["mcpServers"] = entries
		return nil
	})
}

func writeOpenCodeMCPJSON(path string, servers []setupMCPServer) error {
	return updateJSONFile(path, func(document map[string]any) error {
		entries, err := objectField(document, "mcp")
		if err != nil {
			return err
		}
		for _, server := range servers {
			entry := map[string]any{"enabled": true}
			if server.isLocal() {
				entry["type"] = "local"
				entry["command"] = append([]string{server.Command}, server.Args...)
				if len(server.Env) > 0 {
					entry["environment"] = server.Env
				}
			} else {
				entry["type"] = "remote"
				entry["url"] = server.URL
				if len(server.Headers) > 0 {
					entry["headers"] = server.Headers
				}
			}
			entries[server.Name] = entry
		}
		document["mcp"] = entries
		return nil
	})
}

func writeZedMCPJSON(path string, servers []setupMCPServer) error {
	return updateJSONFile(path, func(document map[string]any) error {
		entries, err := objectField(document, "context_servers")
		if err != nil {
			return err
		}
		for _, server := range servers {
			entries[server.Name] = standardMCPEntry(server)
		}
		document["context_servers"] = entries
		return nil
	})
}

func writeAmpMCPJSON(path string, servers []setupMCPServer) error {
	return updateJSONFile(path, func(document map[string]any) error {
		entries, err := objectField(document, "amp.mcpServers")
		if err != nil {
			return err
		}
		for _, server := range servers {
			entries[server.Name] = standardMCPEntry(server)
		}
		document["amp.mcpServers"] = entries
		return nil
	})
}

func standardMCPEntry(server setupMCPServer) map[string]any {
	if server.isLocal() {
		entry := map[string]any{"command": server.Command, "args": append([]string(nil), server.Args...)}
		if len(server.Env) > 0 {
			entry["env"] = server.Env
		}
		return entry
	}
	entry := map[string]any{"url": server.URL}
	if len(server.Headers) > 0 {
		entry["headers"] = server.Headers
	}
	return entry
}

func mcpRemoteEntry(server setupMCPServer) map[string]any {
	args := []string{"-y", "mcp-remote", server.URL}
	environment := map[string]string{}
	for index, name := range sortedStringKeys(server.Headers) {
		envName := fmt.Sprintf("YEELIGHT_MCP_HEADER_%d", index+1)
		environment[envName] = server.Headers[name]
		args = append(args, "--header", name+":${"+envName+"}")
	}
	entry := map[string]any{"command": "npx", "args": args}
	if len(environment) > 0 {
		entry["env"] = environment
	}
	return entry
}

func updateJSONFile(path string, update func(map[string]any) error) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("MCP config path is empty")
	}
	document := map[string]any{}
	var original []byte
	originalExists := false
	if data, err := os.ReadFile(path); err == nil {
		original = data
		originalExists = true
		if len(bytes.TrimSpace(data)) > 0 {
			if err := json.Unmarshal(data, &document); err != nil {
				return fmt.Errorf("parse existing MCP config %s: %w", path, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := update(document); err != nil {
		return err
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".yeelight-mcp-*.json")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(append(data, '\n')); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := verifyMCPConfigUnchanged(path, original, originalExists); err != nil {
		return err
	}
	return replaceMCPConfigFile(tempPath, path)
}

func verifyMCPConfigUnchanged(path string, original []byte, originalExists bool) error {
	current, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if originalExists {
			return fmt.Errorf("MCP config changed during setup; retry without another config editor running")
		}
		return nil
	}
	if err != nil {
		return err
	}
	if !originalExists || !bytes.Equal(current, original) {
		return fmt.Errorf("MCP config changed during setup; retry without another config editor running")
	}
	return nil
}

func replaceMCPConfigFile(source, destination string) error {
	if runtime.GOOS != "windows" {
		return os.Rename(source, destination)
	}
	return replaceMCPConfigFileWindows(source, destination, os.Rename)
}

func replaceMCPConfigFileWindows(source, destination string, rename func(string, string) error) error {
	if _, err := os.Lstat(destination); os.IsNotExist(err) {
		return rename(source, destination)
	} else if err != nil {
		return err
	}
	backupFile, err := os.CreateTemp(filepath.Dir(destination), ".yeelight-mcp-backup-*.json")
	if err != nil {
		return err
	}
	backup := backupFile.Name()
	if err := backupFile.Close(); err != nil {
		return err
	}
	if err := os.Remove(backup); err != nil {
		return err
	}
	if err := rename(destination, backup); err != nil {
		return err
	}
	if err := rename(source, destination); err != nil {
		if restoreErr := rename(backup, destination); restoreErr != nil {
			return errors.Join(err, fmt.Errorf("restore previous MCP config from %s: %w", backup, restoreErr))
		}
		return err
	}
	return os.Remove(backup)
}

func objectField(document map[string]any, name string) (map[string]any, error) {
	value, exists := document[name]
	if !exists {
		return map[string]any{}, nil
	}
	existing, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("existing MCP config field %q must be a JSON object", name)
	}
	return existing, nil
}

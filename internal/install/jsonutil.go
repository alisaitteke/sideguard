// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package install

import (
	"encoding/json"
	"fmt"
)

func patchJSONObject(data []byte, mutate func(map[string]json.RawMessage) error) ([]byte, error) {
	doc := map[string]json.RawMessage{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, err
		}
	}
	if err := mutate(doc); err != nil {
		return nil, err
	}
	return marshalJSONPretty(doc)
}

func rawMCPServers(doc map[string]json.RawMessage) (map[string]mcpServerEntry, error) {
	raw, ok := doc["mcpServers"]
	if !ok || len(raw) == 0 {
		return map[string]mcpServerEntry{}, nil
	}
	servers := map[string]mcpServerEntry{}
	if err := json.Unmarshal(raw, &servers); err != nil {
		return nil, fmt.Errorf("parse mcpServers: %w", err)
	}
	return servers, nil
}

func setRawMCPServers(doc map[string]json.RawMessage, servers map[string]mcpServerEntry) error {
	if servers == nil {
		servers = map[string]mcpServerEntry{}
	}
	raw, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	doc["mcpServers"] = raw
	return nil
}

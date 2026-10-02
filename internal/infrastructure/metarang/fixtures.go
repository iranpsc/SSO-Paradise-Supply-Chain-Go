package metarang

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
)

type Registry interface {
	LookupRegistration(context.Context, string) (bool, string, error)
}

type Fixture struct {
	Registered bool   `json:"already_registered"`
	Code       string `json:"user_code"`
}

type fixtureRegistry map[string]Fixture

// Local fixtures emulate only the external membership lookup. Wallet signatures,
// nonces, ownership and all database writes still use the normal application.
func LoadRegistry(baseURL, fixturePath, environment string) (Registry, error) {
	if fixturePath == "" {
		return Client{BaseURL: baseURL}, nil
	}
	if environment != "development" {
		return nil, errors.New("local wallet fixtures require development mode")
	}
	data, err := os.ReadFile(fixturePath)
	if err != nil || len(data) > 256*1024 {
		return nil, errors.New("local wallet fixtures are unavailable")
	}
	var entries fixtureRegistry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, errors.New("invalid local wallet fixtures")
	}
	for address, entry := range entries {
		if len(address) != 42 || !strings.HasPrefix(address, "0x") || (entry.Registered && entry.Code == "") {
			return nil, errors.New("invalid local wallet fixture entry")
		}
	}
	return entries, nil
}

func (f fixtureRegistry) LookupRegistration(_ context.Context, address string) (bool, string, error) {
	entry := f[strings.ToLower(address)]
	return entry.Registered, entry.Code, nil
}

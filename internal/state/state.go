package state

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/YewFence/yew-key/internal/keyringstore"
	"github.com/YewFence/yew-key/internal/provider"
	"github.com/adrg/xdg"
)

const relativeStatePath = "yewk/state.json"

type State struct {
	Profiles map[string]ProfileState `json:"profiles,omitempty"`
}

type ProfileState struct {
	LastSuccess time.Time               `json:"last_success,omitempty"`
	LastError   string                  `json:"last_error,omitempty"`
	Cursor      provider.SyncCursor     `json:"cursor,omitempty"`
	Variables   []keyringstore.Variable `json:"variables,omitempty"`
}

func Path() (string, error) {
	return xdg.StateFile(relativeStatePath)
}

func Load() (State, string, error) {
	path, err := Path()
	if err != nil {
		return State{}, "", err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{Profiles: map[string]ProfileState{}}, path, nil
	}
	if err != nil {
		return State{}, path, err
	}
	if len(data) == 0 {
		return State{Profiles: map[string]ProfileState{}}, path, nil
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, path, err
	}
	if state.Profiles == nil {
		state.Profiles = map[string]ProfileState{}
	}
	return state, path, nil
}

func Save(state State) (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

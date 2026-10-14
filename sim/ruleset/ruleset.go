// Package ruleset is the seam between the shared simulation engine and a
// specific game version (vanilla Classic Era today, other Classic versions
// later). Everything that differs between versions should be reachable
// through a Ruleset so a new version can be added without forking the engine.
//
// Today a Ruleset owns which class/spec modules are registered. Version
// specific constants and data still live in sim/core and assets/database and
// are vanilla values; they move behind this interface once a second version
// needs different ones (see CLAUDE.md, "Ruleset layer").
package ruleset

import (
	"fmt"
	"sort"
	"sync"
)

// Default is the ruleset used when none is specified.
const Default = "vanilla"

type Ruleset struct {
	// ID is the short, stable identifier used on the command line, e.g. "vanilla".
	ID string
	// Name is the human readable game version name.
	Name string
	// MaxLevel is the player level cap for this version.
	MaxLevel int32
	// RegisterSpecs registers every class/spec module that this version supports.
	RegisterSpecs func()
}

var (
	mu         sync.Mutex
	rulesets   = map[string]*Ruleset{}
	activeID   string
	activeOnce sync.Once
)

// Register makes a ruleset available. Called from each ruleset package's init.
func Register(r *Ruleset) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := rulesets[r.ID]; ok {
		panic(fmt.Sprintf("ruleset %q registered twice", r.ID))
	}
	rulesets[r.ID] = r
}

// Get returns the ruleset with the given ID.
func Get(id string) (*Ruleset, error) {
	mu.Lock()
	defer mu.Unlock()
	r, ok := rulesets[id]
	if !ok {
		return nil, fmt.Errorf("unknown ruleset %q (available: %v)", id, idsLocked())
	}
	return r, nil
}

// IDs lists the registered ruleset IDs in sorted order.
func IDs() []string {
	mu.Lock()
	defer mu.Unlock()
	return idsLocked()
}

func idsLocked() []string {
	ids := make([]string, 0, len(rulesets))
	for id := range rulesets {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Activate registers the spec modules of the given ruleset. The engine's spec
// registry is global, so only one ruleset can be active per process; calling
// Activate again with the same ID is a no-op and with a different ID is an error.
func Activate(id string) error {
	r, err := Get(id)
	if err != nil {
		return err
	}
	mu.Lock()
	if activeID != "" && activeID != id {
		prev := activeID
		mu.Unlock()
		return fmt.Errorf("ruleset %q is already active; cannot switch to %q in the same process", prev, id)
	}
	activeID = id
	mu.Unlock()
	activeOnce.Do(r.RegisterSpecs)
	return nil
}

// Active returns the ID of the active ruleset, or "" if none has been activated.
func Active() string {
	mu.Lock()
	defer mu.Unlock()
	return activeID
}

package invoke

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

// Registry provides bidirectional lookup between TL schema names,
// type IDs, and Go reflect.Types.
type Registry struct {
	mu sync.RWMutex

	// idToName maps type ID → TL schema name (e.g. 0x545cd15a → "messages.sendMessage#545cd15a").
	idToName map[uint32]string
	// nameToID maps schema name → type ID (e.g. "messages.sendMessage" → 0x545cd15a).
	// The key is the bare name without the hash suffix.
	nameToID map[string]uint32
	// nameToType maps schema name → reflect.Type for the Go struct.
	nameToType map[string]reflect.Type
	// nameToConstructor maps schema name → constructor function returning bin.Object.
	nameToConstructor map[string]func() bin.Object
	// ctors caches the constructor map from tg.TypesConstructorMap() for fast lookup by type ID.
	ctors map[uint32]func() bin.Object
}

var (
	globalRegistry *Registry
	globalOnce     sync.Once
)

// GlobalRegistry returns a singleton Registry initialized from tg.TypesMap()
// and tg.TypesConstructorMap(). Safe for concurrent use.
func GlobalRegistry() *Registry {
	globalOnce.Do(func() {
		globalRegistry = newRegistry()
	})
	return globalRegistry
}

func newRegistry() *Registry {
	r := &Registry{
		idToName:          make(map[uint32]string),
		nameToID:          make(map[string]uint32),
		nameToType:        make(map[string]reflect.Type),
		nameToConstructor: make(map[string]func() bin.Object),
	}

	// Build idToName from TypesMap.
	for id, name := range tg.TypesMap() {
		r.idToName[id] = name
		// Strip the #hash suffix to get the bare schema name.
		bare := name
		if idx := strings.Index(name, "#"); idx >= 0 {
			bare = name[:idx]
		}
		r.nameToID[bare] = id
	}

	// Build nameToType and nameToConstructor from TypesConstructorMap.
	constructors := tg.TypesConstructorMap()
	r.ctors = constructors
	for id, ctor := range constructors {
		name, ok := r.idToName[id]
		if !ok {
			continue
		}
		bare := name
		if idx := strings.Index(name, "#"); idx >= 0 {
			bare = name[:idx]
		}

		obj := ctor()
		r.nameToConstructor[bare] = ctor
		r.nameToType[bare] = reflect.TypeOf(obj)
	}

	return r
}

// NameByID resolves a type ID to its TL schema name.
// Returns the bare name (without #hash) and the full name (with #hash).
// Returns ("", "") if not found.
func (r *Registry) NameByID(id uint32) (bare, full string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	full, ok := r.idToName[id]
	if !ok {
		return "", ""
	}
	bare = full
	if idx := strings.Index(full, "#"); idx >= 0 {
		bare = full[:idx]
	}
	return bare, full
}

// IDByName resolves a bare TL schema name to its type ID.
// Returns 0 and false if not found.
func (r *Registry) IDByName(name string) (uint32, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.nameToID[name]
	return id, ok
}

// TypeByName resolves a bare TL schema name to its Go reflect.Type.
// Returns nil if not found.
func (r *Registry) TypeByName(name string) reflect.Type {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.nameToType[name]
}

// ConstructorByName returns a constructor function for the given schema name.
// Returns nil if not found.
func (r *Registry) ConstructorByName(name string) func() bin.Object {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.nameToConstructor[name]
}

// NewRequest creates a new zero-value request struct for the given schema name.
// Returns an error if the name is not found or does not represent a request type.
func (r *Registry) NewRequest(name string) (bin.Object, error) {
	ctor := r.ConstructorByName(name)
	if ctor == nil {
		return nil, fmt.Errorf("unknown method: %s", name)
	}
	return ctor(), nil
}

// Methods returns all registered method names (bare schema names).
func (r *Registry) Methods() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.nameToConstructor))
	for name := range r.nameToConstructor {
		names = append(names, name)
	}
	return names
}

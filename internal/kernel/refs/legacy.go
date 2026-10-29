package refs

import "fmt"

// LegacySystem names a legacy source system.
type LegacySystem string

// Legacy systems (E0 §9B).
const (
	LegacyWP      LegacySystem = "wp"
	LegacyWoo     LegacySystem = "woo"
	LegacyGateway LegacySystem = "gateway"
	LegacyBiawin  LegacySystem = "biawin"
)

// Valid reports whether s is a declared legacy system.
func (s LegacySystem) Valid() bool {
	switch s {
	case LegacyWP, LegacyWoo, LegacyGateway, LegacyBiawin:
		return true
	}
	return false
}

// LegacyRef is an opaque reference to a legacy record. Its fields are
// unexported so it can only be created through NewLegacyRef, and the
// architecture lint permits that call only from internal/integration/legacy.
// The kernel itself holds no legacy table knowledge: Entity is caller data.
type LegacyRef struct {
	system LegacySystem
	entity string
	id     string
}

const maxEntityLen = 64

// NewLegacyRef validates and returns a LegacyRef. Restricted by architecture
// lint to the legacy integration package.
func NewLegacyRef(system LegacySystem, entity, id string) (LegacyRef, error) {
	if !system.Valid() {
		return LegacyRef{}, fmt.Errorf("%w: %q", ErrInvalidLegacySystem, string(system))
	}
	if !validEntity(entity) {
		return LegacyRef{}, fmt.Errorf("%w: %q", ErrInvalidLegacyEntity, entity)
	}
	if err := ValidateID(id); err != nil {
		return LegacyRef{}, err
	}
	return LegacyRef{system: system, entity: entity, id: id}, nil
}

func validEntity(s string) bool {
	if s == "" || len(s) > maxEntityLen || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

// System returns the legacy system.
func (r LegacyRef) System() LegacySystem { return r.system }

// Entity returns the legacy entity (table or object type) name.
func (r LegacyRef) Entity() string { return r.entity }

// ID returns the legacy native id.
func (r LegacyRef) ID() string { return r.id }

// IsZero reports whether r was never constructed.
func (r LegacyRef) IsZero() bool { return r.system == "" }

// Equal reports full equality of system, entity and id.
func (r LegacyRef) Equal(o LegacyRef) bool { return r == o }

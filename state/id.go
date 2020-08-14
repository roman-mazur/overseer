package state

import (
	"fmt"
	"strconv"
)

// ItemId represents a unique state item ID.
// It must be unique in a Set.
// If two items from different Sets share the same ID, it means they represent the same structural elemnt of the state.
type ItemId interface {
	String() string
}

// StringId is a primitive ItemId implementation that holds a unique string identifier.
type StringId string

func (sid StringId) String() string {
	return string(sid)
}

// valueId is an implementation of ItemId used by the reflect state Set builders.
// It represents a node of the state structure tree.
type valueId struct {
	// Structure tree references.
	parent *valueId
	children []*valueId

	// If the related item is inside some list (slice or array).
	listMember bool

	// ID value.
	part     string
	cachedId string
}

func (vi *valueId) next(part string) *valueId {
	res := &valueId{part: part, parent: vi}
	if vi != nil {
		vi.children = append(vi.children, res)
	}
	return res
}

func (vi *valueId) nextListId(index int) *valueId {
	res := vi.next(strconv.Itoa(index))
	res.listMember = true
	return res
}

func (vi *valueId) inject(id string) *valueId {
	if vi != nil && len(vi.cachedId) > 0 {
		panic(fmt.Errorf("state: inject with %s after cache value has been set on %s", id, vi.cachedId))
	}

	if vi != nil && vi.listMember {
		// Replace list index with the provided ID.
		vi.part = id
		return vi
	}

	inject := &valueId{part: id, parent: vi}
	if vi != nil {
		inject.children = vi.children
		vi.children = []*valueId{inject}
		for _, ch := range inject.children {
			ch.parent = inject
		}
	}
	return inject
}

func (vi *valueId) String() string {
	if vi == nil {
		return ""
	}

	if len(vi.cachedId) == 0 {
		parent := ""
		if vi.parent != nil {
			parent = vi.parent.String() + "/"
		}
		vi.cachedId = parent + vi.part
	}
	return vi.cachedId
}

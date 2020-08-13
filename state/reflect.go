package state

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type valueId struct {
	parts    []string
	cachedId string
}

func (vi *valueId) next(part string) *valueId {
	res := &valueId{parts: vi.parts}
	res.update(part)
	return res
}

func (vi *valueId) update(part string) {
	if len(vi.cachedId) > 0 {
		panic(fmt.Errorf("update after cache value has been set"))
	}
	vi.parts = append(vi.parts, part)
}

func (vi *valueId) String() string {
	if len(vi.cachedId) == 0 {
		vi.cachedId = strings.Join(vi.parts, "/")
	}
	return vi.cachedId
}

type valueStateItem struct {
	Actionable

	valueId *valueId
	value   reflect.Value
}

func (vsi valueStateItem) String() string {
	return fmt.Sprintf("id:%s value:[%s]", vsi.valueId, vsi.value.Type())
}

func (vsi valueStateItem) Id() string {
	return vsi.valueId.String()
}

func (vsi valueStateItem) IsSame(other Item) bool {
	if aVsi, ok := other.(valueStateItem); ok {
		return vsi.Id() == aVsi.Id() && reflect.DeepEqual(vsi.value.Interface(), aVsi.value.Interface())
	} else {
		return false
	}
}

type noAction struct{}

func (na noAction) Create(context.Context) error             { return nil }
func (na noAction) Remove(context.Context) error             { return nil }
func (na noAction) Update(context.Context, Actionable) error { return nil }

var (
	noop   Actionable = noAction{}
	rootId            = &valueId{parts: []string{""}}
)

func init() {
	// Make sure it's made immutable early.
	_ = rootId.String()
}

// BuildStateItems creates a state representation fom the input struct or slice.
func BuildStateItems(input interface{}) ([]Item, error) {
	v := reflect.ValueOf(input)
	res, err := buildStateItem(v, rootId, noop)
	if err != nil {
		return nil, err
	}
	if cRes, ok := res.(ComposedItem); ok {
		return cRes.Parts, nil
	}
	return nil, fmt.Errorf("unsupported type %s, value: %v", v.Kind(), v)
}

func buildStateItem(v reflect.Value, id *valueId, actionable Actionable) (Item, error) {
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		// Unwrap first.
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		panic("value is not unwrapped: " + v.String())

	case reflect.Slice, reflect.Array:
		parts := make([]Item, v.Len())
		for i := range parts {
			var err error
			parts[i], err = buildStateItem(v.Index(i), id.next(strconv.Itoa(i)), noop)
			if err != nil {
				return nil, err
			}
		}
		return ComposedItem{id, parts}, nil

	case reflect.Map:
		parts := make([]Item, v.Len())
		i := 0
		iter := v.MapRange()
		for iter.Next() {
			var err error
			parts[i], err = buildStateItem(iter.Value(), id.next(iter.Key().String()), noop)
			if err != nil {
				return nil, err
			}
			i++
		}
		return ComposedItem{id, parts}, nil

	case reflect.Struct:
		parts := make([]Item, 0, v.NumField())
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			// Skip unexported fields.
			if field.PkgPath != "" {
				continue
			}

			tag := field.Tag.Get("state")
			if tag == "-" {
				continue
			}
			if tag == "id" {
				id.update(fmt.Sprintf("%s", v.Field(i)))
				continue
			}

			if part, err := buildStateItem(v.Field(i), id.next(field.Name), noop); err != nil {
				return nil, err
			} else {
				parts = append(parts, part)
			}
		}
		return ComposedItem{id, parts}, nil

	default:
		return valueStateItem{
			Actionable: buildActionable(v),
			valueId:    id,
			value:      v,
		}, nil
	}
}

type actions struct {
	create, remove Action
	update         UpdateAction
}

func (a actions) Create(ctx context.Context) error                  { return a.create(ctx) }
func (a actions) Remove(ctx context.Context) error                  { return a.remove(ctx) }
func (a actions) Update(ctx context.Context, prev Actionable) error { return a.update(ctx, prev) }

func buildActionable(target reflect.Value) Actionable {
	iValue := target.Interface()
	if act, ok := iValue.(Actionable); ok {
		return act
	}
	return noop
}

//type fieldMetadata struct {
//	id bool
//	ignore bool
//}
//
//func parseMetadata(data string) *fieldMetadata {
//	json.Marshal()
//}

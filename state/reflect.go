package state

import (
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

func (vsi valueStateItem) IsSame(other StateItem) bool {
	if aVsi, ok := other.(valueStateItem); ok {
		return vsi.Id() == aVsi.Id() && reflect.DeepEqual(vsi.value.Interface(), aVsi.value.Interface())
	} else {
		return false
	}
}

var rootId = &valueId{parts: []string{""}}

func init() {
	// Make sure it's made immutable early.
	_ = rootId.String()
}

// BuildStateItems creates a state representation fom the input struct or slice.
func BuildStateItems(input interface{}) ([]StateItem, error) {
	v := reflect.ValueOf(input)
	res, err := buildStateItem(v, rootId)
	if err != nil {
		return nil, err
	}
	if cRes, ok := res.(ComposedStateItem); ok {
		return cRes.Parts, nil
	}
	return nil, fmt.Errorf("unsupported type %s, value: %v", v.Kind(), v)
}

func buildStateItem(v reflect.Value, id *valueId) (StateItem, error) {
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		// Unwrap first.
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		panic("value is not unwrapped: " + v.String())

	case reflect.Slice, reflect.Array:
		parts := make([]StateItem, v.Len())
		for i := range parts {
			var err error
			parts[i], err = buildStateItem(v.Index(i), id.next(strconv.Itoa(i)))
			if err != nil {
				return nil, err
			}
		}
		return ComposedStateItem{id, parts}, nil

	case reflect.Map:
		parts := make([]StateItem, v.Len())
		i := 0
		iter := v.MapRange()
		for iter.Next() {
			var err error
			parts[i], err = buildStateItem(iter.Value(), id.next(iter.Key().String()))
			if err != nil {
				return nil, err
			}
			i++
		}
		return ComposedStateItem{id, parts}, nil

	case reflect.Struct:
		parts := make([]StateItem, 0, v.NumField())
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
			// TODO: Use the actionable instance.
			_ = buildActionable(v, tag)

			if part, err := buildStateItem(v.Field(i), id.next(field.Name)); err != nil {
				return nil, err
			} else {
				parts = append(parts, part)
			}
		}
		return ComposedStateItem{id, parts}, nil

	default:
		return valueStateItem{
			valueId: id,
			value:   v,
		}, nil
	}
}

func buildActionable(target reflect.Value, funcName string) Actionable {
	// TODO: Implement.
	return nil
}

//type fieldMetadata struct {
//	id bool
//	ignore bool
//}
//
//func parseMetadata(data string) *fieldMetadata {
//	json.Marshal()
//}

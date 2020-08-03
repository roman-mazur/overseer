package state

import (
	"fmt"
	"reflect"
	"strconv"
)

type valueStateItem struct {
	Actionable

	valueId string
	value   reflect.Value
}

func (vsi valueStateItem) String() string {
	return fmt.Sprintf("%s [%s]", vsi.valueId, vsi.value.Type())
}

func (vsi valueStateItem) Id() string {
	return vsi.valueId
}

func (vsi valueStateItem) IsSame(other StateItem) bool {
	if aVsi, ok := other.(valueStateItem); ok {
		return vsi.Id() == aVsi.Id() && reflect.DeepEqual(vsi.value.Interface(), aVsi.value.Interface())
	} else {
		return false
	}
}

// BuildStateItems creates a state representation fom the input struct or slice.
func BuildStateItems(input interface{}) ([]StateItem, error) {
	v := reflect.ValueOf(input)
	res, err := buildStateItem(v, "")
	if err != nil {
		return nil, err
	}
	if cRes, ok := res.(ComposedStateItem); ok {
		return cRes.Parts, nil
	}
	return nil, fmt.Errorf("unsupported type %s, value: %v", v.Kind(), v)
}

func buildStateItem(v reflect.Value, id string) (StateItem, error) {
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
			parts[i], err = buildStateItem(v.Index(i), nextId(id, strconv.Itoa(i)))
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
			parts[i], err = buildStateItem(iter.Value(), nextId(id, iter.Key().String()))
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
			if part, err := buildStateItem(v.Field(i), nextId(id, field.Name)); err != nil {
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

func nextId(id string, name string) string {
	return fmt.Sprintf("%s/%s", id, name)
}

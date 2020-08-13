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
		panic(fmt.Errorf("update with %s after cache value has been set on %s", part, vi.cachedId))
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
	return fmt.Sprintf("{id:%s, value:%s}", vsi.valueId, vsi.value.Type())
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

func (na noAction) Create(context.Context) error              { return nil }
func (na noAction) Remove(context.Context) error              { return nil }
func (na noAction) Update(context.Context, interface{}) error { return nil }

var (
	noop Actionable = noAction{}
)

// BuildStateItems creates a state representation fom the input struct or slice.
func BuildStateItems(input interface{}) ([]Item, error) {
	v := reflect.ValueOf(input)
	res, err := buildStateItem(v, &valueId{parts: []string{""}})
	if err != nil {
		return nil, err
	}
	if cRes, ok := res.(ComposedItem); ok {
		if cRes.Actions != nil && cRes.Actions != noop {
			return []Item{cRes}, nil
		}
		return cRes.Parts, nil
	}
	return nil, fmt.Errorf("unsupported type %s, value: %v", v.Kind(), v)
}

func buildStateItem(v reflect.Value, id *valueId) (Item, error) {
	origValue := v
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
			parts[i], err = buildStateItem(v.Index(i), id.next(strconv.Itoa(i)))
			if err != nil {
				return nil, err
			}
		}
		return ComposedItem{id, parts, nil, nil}, nil

	case reflect.Map:
		parts := make([]Item, v.Len())
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
		return ComposedItem{id, parts, nil, nil}, nil

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

			if part, err := buildStateItem(v.Field(i), id.next(field.Name)); err != nil {
				return nil, err
			} else {
				parts = append(parts, part)
			}
		}
		act, err := structActionable(v, origValue)
		if err != nil {
			return nil, err
		}
		if act == noop {
			act = nil
		}
		return ComposedItem{id, parts, act, v.Interface()}, nil

	default:
		act, err := buildActionable(v)
		if err != nil {
			return nil, err
		}
		return valueStateItem{
			Actionable: act,
			valueId:    id,
			value:      v,
		}, nil
	}
}

func structActionable(v reflect.Value, origValue reflect.Value) (Actionable, error) {
	var (
		act Actionable
		err error
	)
	act, err = buildActionable(origValue)
	if err != nil {
		return nil, err
	}
	if act == noop {
		act, err = buildActionable(v)
		if err != nil {
			return nil, err
		}
	}
	return act, nil
}

type actions struct {
	create, remove Action
	update         UpdateAction
}

func callAction(a Action, ctx context.Context) error {
	if a != nil {
		return a(ctx)
	}
	return nil
}

func (a actions) Create(ctx context.Context) error { return callAction(a.create, ctx) }
func (a actions) Remove(ctx context.Context) error { return callAction(a.remove, ctx) }
func (a actions) Update(ctx context.Context, prev interface{}) error {
	if a.update == nil {
		return nil
	}
	return a.update(ctx, prev)
}

func (a actions) isNoop() bool {
	return a.create == nil && a.remove == nil && a.update == nil
}

var actionType = reflect.TypeOf(Action(nil))
var updateActionType = reflect.TypeOf(UpdateAction(nil))

func actionWithMethod(target reflect.Value, name string) (Action, error) {
	m := target.MethodByName(name)
	if m.Kind() == reflect.Invalid {
		return nil, nil
	}
	t := m.Type()
	if t.NumIn() != actionType.NumIn() || t.In(0) != actionType.In(0) {
		return nil, fmt.Errorf("bad action method signature %v: 1 parameter expected of type context.Context", m)
	}
	return func(ctx context.Context) error {
		res := m.Call([]reflect.Value{reflect.ValueOf(ctx)})
		if res[0].IsNil() {
			return nil
		}
		return res[0].Interface().(error)
	}, nil
}

func updateActionWithMethod(target reflect.Value, name string, fieldIndex []int) (UpdateAction, error) {
	m := target.MethodByName(name)
	if m.Kind() == reflect.Invalid {
		return nil, nil
	}
	t := m.Type()
	if t.NumIn() != updateActionType.NumIn() || t.In(0) != updateActionType.In(0) {
		return nil, fmt.Errorf("bad action method signature %s %#v on %s: 2 parameters expected, first must be context.Context",
			name, m, target.Type())
	}
	return func(ctx context.Context, prev interface{}) error {
		prevArg := reflect.ValueOf(prev)
		if len(fieldIndex) > 0 {
			structValue := prevArg
			if structValue.Kind() == reflect.Ptr {
				structValue = structValue.Elem()
			}
			if structValue.Kind() != reflect.Struct {
				panic("state: inconsistency in method calls, got " + structValue.Type().String() + "instead of a struct")
			}
			prevArg = structValue.FieldByIndex(fieldIndex)
		}
		res := m.Call([]reflect.Value{reflect.ValueOf(ctx), prevArg})
		if res[0].IsNil() {
			return nil
		}
		return res[0].Interface().(error)
	}, nil
}

func buildActionable(target reflect.Value) (Actionable, error) {
	iValue := target.Interface()
	if act, ok := iValue.(Actionable); ok {
		return act, nil
	}

	methodsPossible := target.Kind() == reflect.Struct
	structType := target.Type()
	if target.Kind() == reflect.Ptr && target.Elem().Kind() == reflect.Struct {
		structType = target.Elem().Type()
		methodsPossible = true
	}

	if !methodsPossible {
		return noop, nil
	}

	var (
		res actions
		err error
	)
	if res.create, err = actionWithMethod(target, "Create"); err != nil {
		return nil, err
	}
	if res.remove, err = actionWithMethod(target, "Remove"); err != nil {
		return nil, err
	}
	if res.update, err = updateActionWithMethod(target, "Update", nil); err != nil {
		return nil, err
	}

	if res.update == nil {
		updateActions := make([]UpdateAction, 0, structType.NumField())
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			if field.PkgPath != "" {
				continue
			}
			tagValue := field.Tag.Get("state")
			if tagValue != "" && tagValue != "id" {
				if update, err := updateActionWithMethod(target, tagValue, field.Index); err == nil {
					if update != nil {
						updateActions = append(updateActions, update)
					}
				} else {
					return nil, err
				}
			}
		}
		if len(updateActions) > 0 {
			res.update = func(ctx context.Context, prev interface{}) error {
				for _, act := range updateActions {
					if err := act(ctx, prev); err != nil {
						return err
					}
				}
				return nil
			}
		}
	}

	if res.isNoop() {
		return noop, nil
	}
	return res, err
}

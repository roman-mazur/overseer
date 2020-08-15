package state

import (
	"context"
	"fmt"
	"reflect"
)

type valueStateItem struct {
	Actionable

	valueId *valueId
	value   reflect.Value
	parent  *valueStateItem
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

// BuildStateItems creates a state representation fom the input struct or slice.
func BuildStateItems(input interface{}) ([]Item, error) {
	v := reflect.ValueOf(input)
	res, err := buildStateItem(v, &valueId{}, nil)
	if err != nil {
		return nil, err
	}
	if cRes, ok := res.(ComposedItem); ok {
		if cRes.actions != nil && cRes.actions != noop {
			return []Item{cRes}, nil
		}
		return cRes.Parts, nil
	}
	return nil, fmt.Errorf("unsupported type %s, value: %v", v.Kind(), v)
}

type fieldContext struct {
	field  *reflect.StructField
	target *reflect.Value
}

func buildStateItem(v reflect.Value, id *valueId, fctx *fieldContext) (Item, error) {
	origValue := v
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		// Unwrap first.
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		panic("state: value is not unwrapped: " + v.String())

	case reflect.Slice, reflect.Array:
		parts := make([]Item, v.Len())
		for i := range parts {
			var err error
			parts[i], err = buildStateItem(v.Index(i), id.nextListId(i), nil)
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
			parts[i], err = buildStateItem(iter.Value(), id.next(iter.Key().String()), nil)
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
				id = id.inject(fmt.Sprintf("%s", v.Field(i)))
				continue
			}

			if part, err := buildStateItem(v.Field(i), id.next(field.Name), &fieldContext{field: &field, target: &origValue}); err != nil {
				return nil, err
			} else {
				parts = append(parts, part)
			}
		}
		act, err := structActionable(v, origValue, fctx)
		if err != nil {
			return nil, err
		}
		if act == noop {
			act = nil
		}
		return ComposedItem{id, parts, act, v.Interface()}, nil

	default:
		act, err := buildActionable(v, fctx)
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

func structActionable(v reflect.Value, origValue reflect.Value, fctx *fieldContext) (Actionable, error) {
	var (
		act Actionable
		err error
	)
	act, err = buildActionable(origValue, fctx)
	if err != nil {
		return nil, err
	}
	if act == noop {
		act, err = buildActionable(v, fctx)
		if err != nil {
			return nil, err
		}
	}
	return act, nil
}

var actionFuncType = reflect.TypeOf(ActionFunc(nil))
var updateActionFuncType = reflect.TypeOf(updateActionFunc(nil))

func resolveMethod(target reflect.Value, name string, methodType reflect.Type) (reflect.Value, bool, error) {
	m := target.MethodByName(name)
	if m.Kind() == reflect.Invalid {
		return m, false, nil
	}
	t := m.Type()
	if t.NumIn() != methodType.NumIn() || t.In(0) != methodType.In(0) {
		return m, true, fmt.Errorf("bad action method signature %s, expected %s", t, methodType)
	}
	return m, true, nil
}

func actionWithMethod(target reflect.Value, name string) (ActionFunc, error) {
	m, present, err := resolveMethod(target, name, actionFuncType)
	if !present || err != nil {
		return nil, err
	}
	return func(ctx context.Context) error {
		res := m.Call([]reflect.Value{reflect.ValueOf(ctx)})
		if res[0].IsNil() {
			return nil
		}
		return res[0].Interface().(error)
	}, nil
}

func updateActionWithMethod(target reflect.Value, name string) (updateActionFunc, error) {
	m, present, err := resolveMethod(target, name, updateActionFuncType)
	if !present || err != nil {
		return nil, err
	}
	return func(ctx context.Context, prev interface{}) error {
		prevArg := reflect.ValueOf(prev)
		if vsi, ok := prev.(valueStateItem); ok {
			prevArg = vsi.value
		}
		res := m.Call([]reflect.Value{reflect.ValueOf(ctx), prevArg})
		if res[0].IsNil() {
			return nil
		}
		return res[0].Interface().(error)
	}, nil
}

func buildActionable(target reflect.Value, fctx *fieldContext) (Actionable, error) {
	if fctx != nil && fctx.target == nil {
		panic("state: field context target not defined")
	}

	var (
		res actions
		err error
	)

	iValue := target.Interface()
	if act, ok := iValue.(Actionable); ok {
		res.wrapped = act
	} else {
		cruMethodsPossible := target.Kind() == reflect.Struct
		if target.Kind() == reflect.Ptr && target.Elem().Kind() == reflect.Struct {
			cruMethodsPossible = true
		}

		if cruMethodsPossible {
			if res.create, err = actionWithMethod(target, "Create"); err != nil {
				return nil, err
			}
			if res.remove, err = actionWithMethod(target, "Remove"); err != nil {
				return nil, err
			}
			if res.update, err = updateActionWithMethod(target, "Update"); err != nil {
				return nil, err
			}
		}
	}

	parentUpdateMethod := ""
	if fctx != nil {
		parentUpdateMethod = fctx.field.Tag.Get("state")
		if parentUpdateMethod == "-" || parentUpdateMethod == "id" {
			parentUpdateMethod = ""
		}
	}

	if parentUpdateMethod != "" {
		res.parentUpdate, err = updateActionWithMethod(*fctx.target, parentUpdateMethod)
	}

	if res.isWrapperOnly() {
		return res.wrapped, nil
	}
	if res.isNoop() {
		return noop, nil
	}
	return res, err
}

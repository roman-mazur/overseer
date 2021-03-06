package state

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type recorder []string

func (r *recorder) record(msg string) {
	*r = append(*r, msg)
}

func (r recorder) String() string {
	return "\n" + strings.Join(r, "\n") + "\n"
}

type testStateItem struct {
	id  string
	arg string

	*recorder
}

func (tsi testStateItem) Id() string {
	return tsi.id
}

func (tsi testStateItem) String() string {
	return tsi.id + "/" + tsi.arg
}

func (tsi testStateItem) IsSame(another Item) bool {
	if atsi, ok := another.(testStateItem); ok {
		return tsi.id == atsi.id && tsi.arg == atsi.arg
	} else {
		return false
	}
}

func (tsi testStateItem) Create(ctx context.Context) error {
	tsi.record(fmt.Sprintf("create %s with %s", tsi.id, tsi.arg))
	return nil
}

func (tsi testStateItem) Remove(ctx context.Context) error {
	tsi.record(fmt.Sprintf("remove %s with %s", tsi.id, tsi.arg))
	return nil
}

func (tsi testStateItem) Update(ctx context.Context, from interface{}) error {
	tsi.record(fmt.Sprintf("update %s with %s from %s", tsi.id, tsi.arg, from))
	return nil
}

type testInput struct {
	id  string
	arg string
}

func stateItems(input []testInput, recorder *recorder) Set {
	res := make(Set, len(input))
	for i, in := range input {
		res[i] = testStateItem{id: in.id, arg: in.arg, recorder: recorder}
	}
	return res
}

func TestInferActions(t *testing.T) {
	type args struct {
		prev []testInput
		next []testInput
	}
	tests := []struct {
		name string
		args args
		want recorder
	}{
		{
			name: "create one",
			args: args{
				prev: nil,
				next: []testInput{{"1", "a"}},
			},
			want: recorder{"create 1 with a"},
		},
		{
			name: "delete one",
			args: args{
				prev: []testInput{{"1", "a"}},
				next: nil,
			},
			want: recorder{"remove 1 with a"},
		},
		{
			name: "create and delete",
			args: args{
				prev: []testInput{{"1", "a"}},
				next: []testInput{{"2", "b"}},
			},
			want: recorder{"remove 1 with a", "create 2 with b"},
		},
		{
			name: "noop",
			args: args{
				prev: []testInput{{"1", "a"}, {"2", "b"}},
				next: []testInput{{"2", "b"}, {"1", "a"}},
			},
			want: nil,
		},
		{
			name: "update",
			args: args{
				prev: []testInput{{"1", "a"}, {"2", "b"}},
				next: []testInput{{"2", "c"}, {"1", "a"}},
			},
			want: recorder{"update 2 with c from 2/b"},
		},
		{
			name: "order: remove, update, create",
			args: args{
				prev: []testInput{{"1", "a"}, {"2", "b"}},
				next: []testInput{{"2", "c"}, {"3", "a"}},
			},
			want: recorder{"remove 1 with a", "update 2 with c from 2/b", "create 3 with a"},
		},
	}

	for _, tt := range tests {
		var performedActions recorder

		prev := stateItems(tt.args.prev, &performedActions)
		next := stateItems(tt.args.next, &performedActions)
		t.Run(tt.name, func(t *testing.T) {
			err := InferActions(prev, next).Do(context.TODO())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(performedActions, tt.want) {
				t.Errorf("actions resulted in %v, want %v", performedActions, tt.want)
			}
		})
	}
}

func TestComposedStateItem_Create(t *testing.T) {
	var performedActions recorder
	csi := ComposedItem{IdValue: StringId("csi1"), Parts: stateItems([]testInput{
		{id: "1", arg: "a"}, {id: "2", arg: "b"},
	}, &performedActions)}
	if err := csi.Create(context.TODO()); err != nil {
		t.Fatal(err)
	}
	wanted := recorder{"create 1 with a", "create 2 with b"}
	if !reflect.DeepEqual(performedActions, wanted) {
		t.Errorf("actions resulted in %v, want %v", performedActions, wanted)
	}
}

func TestComposedStateItem_Remove(t *testing.T) {
	var performedActions recorder
	csi := ComposedItem{IdValue: StringId("csi1"), Parts: stateItems([]testInput{
		{id: "1", arg: "a"}, {id: "2", arg: "b"},
	}, &performedActions)}
	if err := csi.Remove(context.TODO()); err != nil {
		t.Fatal(err)
	}
	wanted := recorder{"remove 1 with a", "remove 2 with b"}
	if !reflect.DeepEqual(performedActions, wanted) {
		t.Errorf("actions resulted in %v, want %v", performedActions, wanted)
	}
}

func TestComposedStateItem_IsSame(t *testing.T) {
	csi1 := ComposedItem{IdValue: StringId("csi1"), Parts: stateItems([]testInput{
		{id: "1", arg: "a"}, {id: "2", arg: "b"},
	}, nil)}
	csi1Copy := ComposedItem{IdValue: StringId("csi1"), Parts: stateItems([]testInput{
		{id: "1", arg: "a"}, {id: "2", arg: "b"},
	}, nil)}
	csi1Copy2 := ComposedItem{IdValue: StringId("csi1"), Parts: stateItems([]testInput{
		{id: "2", arg: "b"}, {id: "1", arg: "a"},
	}, nil)}
	csi2 := ComposedItem{IdValue: StringId("csi2"), Parts: stateItems([]testInput{
		{id: "1", arg: "a"}, {id: "2", arg: "b"},
	}, nil)}
	csi1Changed := ComposedItem{IdValue: StringId("csi1"), Parts: stateItems([]testInput{
		{id: "2", arg: "a"}, {id: "1", arg: "b"},
	}, nil)}
	csi1Changed2 := ComposedItem{IdValue: StringId("csi1"), Parts: stateItems([]testInput{
		{id: "1", arg: "a"},
	}, nil)}

	if !csi1.IsSame(csi1Copy) {
		t.Errorf("%s should be the same as %s", csi1, csi1Copy)
	}
	if !csi1.IsSame(csi1Copy2) {
		t.Errorf("%s should be the same as %s", csi1, csi1Copy2)
	}
	if csi1.IsSame(csi2) {
		t.Errorf("%s should not be the same as %s", csi1, csi1Copy)
	}
	if csi1.IsSame(csi1Changed) {
		t.Errorf("%s should not be the same as %s", csi1, csi1Copy)
	}
	if csi1.IsSame(csi1Changed2) {
		t.Errorf("%s should not be the same as %s", csi1, csi1Copy)
	}
}

func TestComposedStateItem_Update(t *testing.T) {
	var performedActions recorder
	csi1 := ComposedItem{IdValue: StringId("csi1"), Parts: stateItems([]testInput{
		{id: "1", arg: "a"}, {id: "2", arg: "b"},
	}, &performedActions)}
	csi1Changed := ComposedItem{IdValue: StringId("csi1"), Parts: stateItems([]testInput{
		{id: "2", arg: "a"}, {id: "1", arg: "b"},
	}, &performedActions)}
	if err := csi1Changed.Update(context.TODO(), csi1); err != nil {
		t.Fatal(err)
	}
	wanted := recorder{"update 1 with b from 1/a", "update 2 with a from 2/b"}
	if !reflect.DeepEqual(performedActions, wanted) {
		t.Errorf("actions resulted in %v, want %v", performedActions, wanted)
	}

	performedActions = nil
	csi1Changed2 := ComposedItem{IdValue: StringId("csi1"), Parts: stateItems([]testInput{
		{id: "1", arg: "a"},
	}, &performedActions)}
	if err := csi1Changed2.Update(context.TODO(), csi1); err != nil {
		t.Fatal(err)
	}
	wanted = recorder{"remove 2 with b"}
	if !reflect.DeepEqual(performedActions, wanted) {
		t.Errorf("actions resulted in %v, want %v", performedActions, wanted)
	}
}

func TestStringStateItem_IsSame(t *testing.T) {
	ssi1 := &StringItem{IdValue: "test-1", Value: "v1"}
	ssi1Copy := &StringItem{IdValue: "test-1", Value: "v1"}
	ssi2 := &StringItem{IdValue: "test-2", Value: "v1"}
	ssi2Changed := &StringItem{IdValue: "test-2", Value: "v2"}

	if !ssi1.IsSame(ssi1Copy) {
		t.Errorf("%s should be the same as %s", ssi1, ssi1Copy)
	}
	if ssi1.IsSame(ssi2) {
		t.Errorf("%s should not be the same as %s", ssi1, ssi2)
	}
	if ssi1.IsSame(ssi2Changed) {
		t.Errorf("%s should not be the same as %s", ssi1, ssi2Changed)
	}
}

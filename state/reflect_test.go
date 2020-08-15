package state

import (
	"context"
	"reflect"
	"testing"
)

func makeExpectedItem(id string, value interface{}) Item {
	return valueStateItem{
		valueId: &valueId{cachedId: id},
		value:   reflect.ValueOf(value),
	}
}

func TestBuildStateItems(t *testing.T) {
	testStruct := struct {
		A string
		B int
		C bool
		e string
	}{"a", 42, true, "internal"}

	wrappingStruct := struct {
		Data interface{}
	}{
		Data: &testStruct,
	}

	structState := []Item{
		makeExpectedItem("/A", "a"),
		makeExpectedItem("/B", 42),
		makeExpectedItem("/C", true),
	}

	mapState := append(structState, makeExpectedItem("/d", 5.5))

	prefixedStructState := func(prefix string) []Item {
		res := make([]Item, len(structState))
		for i := range res {
			item := structState[i].(valueStateItem)
			res[i] = makeExpectedItem(prefix+item.valueId.String(), item.value.Interface())
		}
		return res
	}

	tests := []struct {
		name    string
		input   interface{}
		want    []Item
		wantErr bool
	}{
		{
			name:    "struct",
			input:   testStruct,
			want:    structState,
			wantErr: false,
		},
		{
			name:    "struct ptr",
			input:   &testStruct,
			want:    structState,
			wantErr: false,
		},
		{
			name: "map",
			input: map[string]interface{}{
				"A": "a", "B": 42, "C": true, "d": 5.5,
			},
			want:    mapState,
			wantErr: false,
		},
		{
			name:  "slice",
			input: []string{"a"},
			want: []Item{
				makeExpectedItem("/0", "a"),
			},
			wantErr: false,
		},
		{
			name:  "array",
			input: [2]string{"abc", "d"},
			want: []Item{
				makeExpectedItem("/0", "abc"),
				makeExpectedItem("/1", "d"),
			},
			wantErr: false,
		},
		{
			name:  "slice of structs",
			input: []interface{}{&testStruct, &testStruct},
			want: []Item{
				ComposedItem{StringId("/0"), prefixedStructState("/0"), nil, nil},
				ComposedItem{StringId("/1"), prefixedStructState("/1"), nil, nil},
			},
			wantErr: false,
		},
		{
			name:  "struct with struct",
			input: &wrappingStruct,
			want: []Item{
				ComposedItem{StringId("/Data"), prefixedStructState("/Data"), nil, nil},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildStateItems(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildStateItems() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				t.Errorf("BuildStateItems() got nil")
				return
			}
			gotComparator := ComposedItem{IdValue: StringId(""), Parts: got}
			wantComparator := ComposedItem{IdValue: StringId(""), Parts: tt.want}
			if !gotComparator.IsSame(wantComparator) {
				t.Errorf("BuildStateItems() got\n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}

func TestBuildStateItems_IdTag(t *testing.T) {
	type data struct {
		Value     int
		Id        string `state:"id"`
		BoolValue bool

		Ignore1 string `state:"-"`
		ignore2 string
	}

	items, err := BuildStateItems(&data{42, "id1", true, "", ""})
	if err != nil {
		t.Fatal(err)
	}

	if len(items) < 2 {
		t.Fatalf("Unexpected count of items: %d", len(items))
	} else if len(items) != 2 {
		t.Errorf("Unexpected count of items: %d", len(items))
	}

	if items[0].Id() != "/id1/Value" {
		t.Errorf("Unexpected IDs in the items: %s", items)
	}
	if items[1].Id() != "/id1/BoolValue" {
		t.Errorf("Unexpected IDs in the items: %s", items)
	}
}

func assureNoErrors(t *testing.T, act Actionable, updateArg interface{}) {
	if err := act.Create(context.TODO()); err != nil {
		t.Error("Unexpected error on create", err)
	}
	if err := act.Remove(context.TODO()); err != nil {
		t.Error("Unexpected error on remove", err)
	}
	if err := act.Update(context.TODO(), updateArg); err != nil {
		t.Error("Unexpected error on update", err)
	}
}

func mustBuildActionable(t *testing.T, v interface{}) Actionable {
	res, err := buildActionable(reflect.ValueOf(v), nil)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestBuildActionable_Noop(t *testing.T) {
	assureNoErrors(t, mustBuildActionable(t, "something"), "does not matter")
	assureNoErrors(t, mustBuildActionable(t, 42), "does not matter")
	assureNoErrors(t, mustBuildActionable(t, noop), "does not matter")
}

func TestBuildActionable_Actionable(t *testing.T) {
	var recording recorder
	tsi := testStateItem{id: "test", arg: "a", recorder: &recording}
	actionable := mustBuildActionable(t, tsi)
	assureNoErrors(t, actionable, tsi)
	want := recorder{"create test with a", "remove test with a", "update test with a from test/a"}
	if !reflect.DeepEqual(recording, want) {
		t.Errorf("Unexpected actions result: got %s, want %s", recording, want)
	}
	if actionable != tsi {
		t.Errorf("Actioinable implemntation was not used, got %s", actionable)
	}
}

type testStateStruct struct {
	Id              string `state:"id"`
	Value           string `state:"Reset"`
	AnotherValue    int
	ActionableValue testStateItem

	*recorder
}

func (t *testStateStruct) Create(context.Context) error {
	t.record("create testStateStruct with id " + t.Id)
	return nil
}

func (t *testStateStruct) Reset(ctx context.Context, prev string) error {
	t.record("change testStateStruct value from " + prev + " to " + t.Value)
	return nil
}

func (t *testStateStruct) Remove(context.Context) error {
	t.record("remove testStateStruct with id " + t.Id)
	return nil
}

func TestBuildActionable_Struct(t *testing.T) {
	var recording recorder
	value := makeTestStruct(&recording)
	valuePrev := makeTestStruct(&recording)
	valuePrev.Value = "v0"
	assureNoErrors(t, mustBuildActionable(t, value), valuePrev)
	want := recorder{
		"create testStateStruct with id aa",
		"remove testStateStruct with id aa",
	}
	if !reflect.DeepEqual(recording, want) {
		t.Errorf("Unexpected actions result: got %s, want %s", recording, want)
	}
}

func TestBuildStateItems_StructActions(t *testing.T) {
	var recording recorder
	value := makeTestStruct(&recording)
	items, err := BuildStateItems(value)
	if err != nil {
		t.Fatal(err)
	}

	// Create.
	if err := InferActions(nil, items).Do(context.TODO()); err != nil {
		t.Fatal(err)
	}
	// Remove.
	if err := InferActions(items, nil).Do(context.TODO()); err != nil {
		t.Fatal(err)
	}
	// Update.
	value2 := makeTestStruct(&recording)
	value2.Value = "v2"
	value2.ActionableValue.arg = "b"
	items2, err := BuildStateItems(value2)
	if err != nil {
		t.Fatal(err)
	}
	if err := InferActions(items, items2).Do(context.TODO()); err != nil {
		t.Fatal(err)
	}

	want := recorder{
		"create testStateStruct with id aa",
		"create bb with some arg",
		"remove bb with some arg",
		"remove testStateStruct with id aa",
		"change testStateStruct value from v1 to v2",
		"update bb with b from bb/some arg",
	}
	if !reflect.DeepEqual(recording, want) {
		t.Errorf("Unexpected actions result: got %s, want %s", recording, want)
	}
}

type Inner struct{ *recorder }

func (i Inner) Create(ctx context.Context) error {
	i.record("create inner")
	return nil
}

type outer struct{ Inner }

func TestBuildActionable_Embedded(t *testing.T) {
	var recording recorder
	items, err := BuildStateItems(&outer{Inner{&recording}})
	if err != nil {
		t.Fatal(err)
	}
	_ = InferActions(nil, items).Do(context.TODO())
	want := recorder{"create inner"}
	if !reflect.DeepEqual(recording, want) {
		t.Errorf("Unexpected result: want %s, got %s", want, recording)
	}
}

func makeTestStruct(recorder *recorder) *testStateStruct {
	return &testStateStruct{
		recorder:     recorder,
		Id:           "aa",
		Value:        "v1",
		AnotherValue: 11,
		ActionableValue: testStateItem{
			id:       "bb",
			arg:      "some arg",
			recorder: recorder,
		},
	}
}

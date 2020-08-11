package state

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func makeValueStateItem(id string, value interface{}) valueStateItem {
	return valueStateItem{
		valueId: &valueId{parts: strings.Split(id, "/")},
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

	structState := []StateItem{
		makeValueStateItem("/A", "a"),
		makeValueStateItem("/B", 42),
		makeValueStateItem("/C", true),
	}

	mapState := append(structState, makeValueStateItem("/d", 5.5))

	indexedStructState := func(k int) []StateItem {
		res := make([]StateItem, len(structState))
		for i := range res {
			item := structState[i].(valueStateItem)
			res[i] = makeValueStateItem(fmt.Sprintf("/%d%s", k, item.valueId), item.value.Interface())
		}
		return res
	}

	tests := []struct {
		name    string
		input   interface{}
		want    []StateItem
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
			want: []StateItem{
				makeValueStateItem("/0", "a"),
			},
			wantErr: false,
		},
		{
			name:  "array",
			input: [2]string{"abc", "d"},
			want: []StateItem{
				makeValueStateItem("/0", "abc"),
				makeValueStateItem("/1", "d"),
			},
			wantErr: false,
		},
		{
			name:  "slice of structs",
			input: []interface{}{&testStruct, &testStruct},
			want: []StateItem{
				ComposedStateItem{StringId("/0"), indexedStructState(0)},
				ComposedStateItem{StringId("/1"), indexedStructState(1)},
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
			if !(ComposedStateItem{IdValue: StringId(""), Parts: got}).IsSame(ComposedStateItem{IdValue: StringId(""), Parts: tt.want}) {
				t.Errorf("BuildStateItems() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildStateItems_IdTag(t *testing.T) {
	type data struct {
		Id string `state:"id"`

		Value     int
		BoolValue bool

		Ignore1 string `state:"-"`
		ignore2 string
	}

	items, err := BuildStateItems(&data{"id1", 42, true, "", ""})
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

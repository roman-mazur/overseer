package state

import (
	"fmt"
	"reflect"
	"testing"
)

func TestBuildStateItems(t *testing.T) {
	testStruct := struct {
		A string
		B int
		C bool
		e string
	}{"a", 42, true, "internal"}

	structState := []StateItem{
		valueStateItem{valueId: "/A", value: reflect.ValueOf("a")},
		valueStateItem{valueId: "/B", value: reflect.ValueOf(42)},
		valueStateItem{valueId: "/C", value: reflect.ValueOf(true)},
	}

	mapState := append(structState, valueStateItem{
		valueId: "/d", value: reflect.ValueOf(5.5),
	})

	indexedStructState := func(k int) []StateItem {
		res := make([]StateItem, len(structState))
		for i := range res {
			item := structState[i].(valueStateItem)
			item.valueId = fmt.Sprintf("/%d%s", k, item.valueId)
			res[i] = item
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
				valueStateItem{valueId: "/0", value: reflect.ValueOf("a")},
			},
			wantErr: false,
		},
		{
			name:  "array",
			input: [2]string{"abc", "d"},
			want: []StateItem{
				valueStateItem{valueId: "/0", value: reflect.ValueOf("abc")},
				valueStateItem{valueId: "/1", value: reflect.ValueOf("d")},
			},
			wantErr: false,
		},
		{
			name:  "slice of structs",
			input: []interface{}{&testStruct, &testStruct},
			want: []StateItem{
				ComposedStateItem{"/0", indexedStructState(0)},
				ComposedStateItem{"/1", indexedStructState(1)},
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
			if !(ComposedStateItem{Parts: got}).IsSame(ComposedStateItem{Parts: tt.want}) {
				t.Errorf("BuildStateItems() got = %v, want %v", got, tt.want)
			}
		})
	}
}

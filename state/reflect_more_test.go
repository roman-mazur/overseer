package state_test

import (
	"rmazur.io/overseer/state"
	"testing"
)

func TestBuildStateItems_Rooms(t *testing.T) {
	blueSpace := Space{ColorBlue, 1}
	redSpace := Space{ColorRed, 1}

	testState, _ := state.BuildStateItems(struct{Rooms []*Room}{Rooms: []*Room{
		{Name: "bedroom 1", Space: blueSpace},
		{Name: "bedroom 2", Space: redSpace},
	}})
	t.Log(testState)

	if len(testState) != 1 {
		t.Fatal("Bad result state", testState)
	}
	cstate, ok := testState[0].(state.ComposedItem)
	if !ok {
		t.Fatal("Bad result state", testState)
	}
	if len(cstate.Parts) != 2 {
		t.Fatal("Bad result state", cstate.Parts)
	}
	room1State, ok := cstate.Parts[0].(state.ComposedItem)
	if !ok {
		t.Fatal("Bad result state", cstate.Parts[0])
	}
	if room1State.Id() != "/Rooms/bedroom 1" {
		t.Fatal("Bad result room state", room1State)
	}
	if len(room1State.Parts) != 1 {
		t.Fatal("Bad result room parts state", room1State.Parts)
	}

	if room1State.Parts[0].Id() != "/Rooms/bedroom 1/Space" {
		t.Fatal("Unexpected room part ID", room1State.Parts[0].Id())
	}
}

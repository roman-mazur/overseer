package state_test

import (
	"context"
	"fmt"

	"rmazur.io/overseer/state"
)

type Desk struct {
	Number int
	Color  string
}

func (d *Desk) Id() string {
	return fmt.Sprintf("desk-%d", d.Number)
}

func (d *Desk) IsSame(other state.Item) bool {
	if otherDesk, ok := other.(*Desk); ok {
		return otherDesk.Number == d.Number && otherDesk.Color == d.Color
	} else {
		return false
	}
}

func (d *Desk) String() string {
	return fmt.Sprintf("%s/%s", d.Id(), d.Color)
}

func (d *Desk) Create(ctx context.Context) error {
	fmt.Println("create", d)
	return nil
}

func (d *Desk) Remove(ctx context.Context) error {
	fmt.Println("remove", d)
	return nil
}

func (d *Desk) Update(ctx context.Context, from state.Actionable) error {
	fmt.Println("update from", from, "to", d)
	return nil
}

type HouseRoom struct {
	Number int
	Name   string
	Desks  []*Desk
}

func (r *HouseRoom) String() string {
	return fmt.Sprintf("room-%d/%s", r.Number, r.Name)
}

func (r *HouseRoom) Remove(ctx context.Context) error {
	fmt.Println("remove", r)
	return nil
}

func (r *HouseRoom) Create(ctx context.Context) error {
	fmt.Println("create", r)
	return nil
}

func (r *HouseRoom) Update(ctx context.Context, from state.Actionable) error {
	fmt.Println("update from", from, "to", r)
	return nil
}

func (r *HouseRoom) AsState() state.Item {
	parts := make([]state.Item, len(r.Desks)+1)
	for i := range r.Desks {
		parts[i] = r.Desks[i]
	}
	id := fmt.Sprintf("room-%d", r.Number)
	parts[len(parts)-1] = &state.StringItem{IdValue: fmt.Sprintf("%s-name", id), Value: r.Name, Actionable: r}
	return state.ComposedItem{
		IdValue: state.StringId(id),
		Parts:   parts,
	}
}

func ExampleInferActions() {
	officeRoom := &HouseRoom{
		Number: 1,
		Name:   "Office",
		Desks: []*Desk{
			{Number: 42, Color: "brown"},
		},
	}
	livingRoom := &HouseRoom{
		Number: 2,
		Name:   "Living",
		Desks: []*Desk{
			{Number: 42, Color: "blue"},
		},
	}

	officeRoomRecolored := &HouseRoom{
		Number: 1,
		Name:   "Office",
		Desks: []*Desk{
			{Number: 42, Color: "green"},
		},
	}

	ctx := context.Background()

	fmt.Println(">> color change")
	actions := state.InferActions(
		[]state.Item{officeRoom.AsState(), livingRoom.AsState()},
		[]state.Item{officeRoomRecolored.AsState(), livingRoom.AsState()},
	)
	for _, act := range actions {
		_ = act(ctx)
	}

	bedroom := &HouseRoom{
		Number: 3,
		Name:   "Bedroom",
	}

	fmt.Println(">> structure change")
	actions = state.InferActions(
		[]state.Item{officeRoom.AsState(), livingRoom.AsState()},
		[]state.Item{livingRoom.AsState(), bedroom.AsState()},
	)
	for _, act := range actions {
		_ = act(ctx)
	}

	// Output:
	// >> color change
	// update from desk-42/brown to desk-42/green
	// >> structure change
	// remove desk-42/brown
	// remove room-1/Office
	// create room-3/Bedroom
}

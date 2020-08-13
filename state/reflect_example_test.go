package state_test

import (
	"context"
	"fmt"

	"rmazur.io/overseer/state"
)

type Color int

const (
	ColorWhite Color = iota
	ColorRed
	ColorBlue
)

type Space struct {
	Color Color
	Area  float32
}

func (space Space) Create(ctx context.Context) error {
	fmt.Printf("Creating space with color %d and size %f\n", space.Color, space.Area)
	return nil
}

func (space Space) Repaint(ctx context.Context, prev Color) error {
	fmt.Println("Repainting from", prev, "to", space.Color)
	return nil
}

func (space Space) Resize(ctx context.Context, prev float32) error {
	fmt.Println("Resizing from", prev, "to", space.Area)
	return nil
}

func (space Space) Remove(ctx context.Context) error {
	fmt.Printf("Removing space with color %d and size %f\n", space.Color, space.Area)
	return nil
}

type MobileHouse struct {
	Space
	Bedrooms []*Room
	Address  string

	Id         string `state:"id"`
	HasTenants bool   `state:"-"`
}

func (h *MobileHouse) Create(ctx context.Context) error {
	fmt.Printf("Creating house at %s", h.Address)
	return h.Space.Create(ctx)
}

func (h *MobileHouse) Move(ctx context.Context, prevAddress string) error {
	fmt.Printf("Moving %s from %s to %s\n", h.Id, prevAddress, h.Address)
	return nil
}

func (h *MobileHouse) Removing(ctx context.Context) error {
	fmt.Printf("Removing house at %s", h.Address)
	return h.Space.Remove(ctx)
}

type Room struct {
	Space
	Name string `state:"id"`
}

func (r *Room) Create(ctx context.Context) error {
	fmt.Printf("Creating room %s", r.Name)
	return r.Space.Create(ctx)
}

func ExampleBuildStateItems() {
	blueSpace := Space{ColorBlue, 1}
	redSpace := Space{ColorRed, 1}
	whiteSpace := Space{ColorWhite, 2}

	var state0 []state.Item

	state1, err := state.BuildStateItems(&MobileHouse{
		Space:   blueSpace,
		Id:      "house A",
		Address: "5 Cherry lane",
		Bedrooms: []*Room{
			{Name: "bedroom 0", Space: blueSpace},
			{Name: "bedroom 1", Space: whiteSpace},
		},
		HasTenants: false,
	})
	if err != nil {
		panic(err)
	}

	state2, err := state.BuildStateItems(&MobileHouse{
		Space:   redSpace,
		Id:      "house A",
		Address: "5 Bazhana ave.",
		Bedrooms: []*Room{
			{Name: "bedroom 1", Space: blueSpace},
			{Name: "bedroom 2", Space: redSpace},
		},
		HasTenants: true,
	})
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	fmt.Println(">> From state0 to state1")
	actionsCreate := state.InferActions(state0, state1)
	for _, act := range actionsCreate {
		_ = act(ctx)
	}

	fmt.Println(">> From state1 to state0")
	actionsUpdate := state.InferActions(state1, state2)
	for _, act := range actionsUpdate {
		_ = act(ctx)
	}

	// Output:
	// >> From state0 to state1
	// Creating house at 5 Cherry lane
	// Creating space with color and size
	// Creating room bedroom 0
	// Creating space with color and size
	// Creating room bedroom 1
	// Creating space with color and size
	// >> From state1 to state0
	// Repainting from blue to red
	// Moving house A from 5 Cherry lane to 5 Bazhana ave.
	// Removing bedroom 0
	// Repainting from white to blue
	// Resizing from 2 to 1
	// Creating bedroom 1
}

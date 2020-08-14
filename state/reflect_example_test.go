package state_test

import (
	"context"
	"fmt"

	"rmazur.io/overseer/state"
)

type Color int

func (c Color) String() string {
	switch c {
	case ColorWhite: return "white"
	case ColorRed: return "red"
	case ColorBlue: return "blue"
	default:
		return "unknown"
	}
}

const (
	ColorWhite Color = iota
	ColorRed
	ColorBlue
)

type Space struct {
	Color Color `state:"Repaint"`
	Area  float32 `state:"Resize"`
}

func (space Space) Create(ctx context.Context) error {
	fmt.Printf("Creating space with color %s and size %.1f\n", space.Color, space.Area)
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
	fmt.Printf("Removing space with color %s and size %.1f\n", space.Color, space.Area)
	return nil
}

type MobileHouse struct {
	Space
	Bedrooms []*Room
	Address  string `state:"Move"`

	Id         string `state:"id"`
	HasTenants bool   `state:"-"`
}

func (h *MobileHouse) Create(ctx context.Context) error {
	fmt.Printf("Creating house at %s\n", h.Address)
	return nil
}

func (h *MobileHouse) Move(ctx context.Context, prevAddress string) error {
	fmt.Printf("Moving %s from %s to %s\n", h.Id, prevAddress, h.Address)
	return nil
}

func (h *MobileHouse) Remove(ctx context.Context) error {
	fmt.Printf("Removing house at %s", h.Address)
	return nil
}

type Room struct {
	Space
	Name string `state:"id"`
}

func (r *Room) Create(ctx context.Context) error {
	fmt.Println("Creating room", r.Name)
	return nil
}

func ExampleBuildStateItems() {
	blueSpace := Space{ColorBlue, 1}
	redSpace := Space{ColorRed, 1}
	whiteSpace := Space{ColorWhite, 2}

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

	fmt.Println(">> From empty to state1")
	actionsCreate := state.InferActions(nil, state1)
	for _, act := range actionsCreate {
		_ = act(ctx)
	}

	fmt.Println(">> From state1 to state2")
	actionsUpdate := state.InferActions(state1, state2)
	for _, act := range actionsUpdate {
		_ = act(ctx)
	}

	// Output:
	// >> From empty to state1
	// Creating house at 5 Cherry lane
	// Creating space with color blue and size 1.0
	// Creating room bedroom 0
	// Creating space with color blue and size 1.0
	// Creating room bedroom 1
	// Creating space with color white and size 2.0
	// >> From state1 to state0
	// Repainting from blue to red
	// Removing space with color blue and size 1.0
	// Repainting from white to blue
	// Resizing from 2 to 1
	// Creating bedroom 2
	// Creating space with color red and size 1.0
	// Moving house A from 5 Cherry lane to 5 Bazhana ave.
}

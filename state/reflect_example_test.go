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
}

func (space Space) Repaint(ctx context.Context, prev Color) error {
	fmt.Println("Repainting from", prev, "to", space.Color)
	return nil
}

type House struct {
	Space
	Id       string `state:"id"`
	Bedrooms []*Room
	Address  string
}

func (h *House) Move(ctx context.Context, prevAddress string) error {
	fmt.Printf("Moving %s from %s to %s\n", h.Id, prevAddress, h.Address)
	return nil
}

type Room struct {
	Space
	Name string `state:"id"`
}

func ExampleBuildStateItems() {
	blueSpace := Space{ColorBlue}
	redSpace := Space{ColorRed}
	whiteSpace := Space{ColorWhite}

	state1, err := state.BuildStateItems(&House{
		Space:   blueSpace,
		Id:      "house A",
		Address: "5 Cherry lane",
		Bedrooms: []*Room{
			{Name: "bedroom 0", Space: blueSpace},
			{Name: "bedroom 1", Space: whiteSpace},
		},
	})
	if err != nil {
		panic(err)
	}

	state2, err := state.BuildStateItems(&House{
		Space:   redSpace,
		Id:      "house A",
		Address: "5 Bazhana ave.",
		Bedrooms: []*Room{
			{Name: "bedroom 1", Space: blueSpace},
			{Name: "bedroom 2", Space: redSpace},
		},
	})
	if err != nil {
		panic(err)
	}

	actions := state.InferActions(state1, state2)
	ctx := context.Background()
	for _, act := range actions {
		_ = act(ctx)
	}

	// Output:
	// Repainting from blue to red
	// Moving house A from 5 Cherry lane to 5 Bazhana ave.
	// Removing bedroom 0
	// Repainting from white to blue
	// Adding bedroom 1
}
package overseer

import "fmt"

type Desk struct {
	Number int
	Color string
}

func (d *Desk) Id() string {
	return fmt.Sprintf("desk-%d", d.Number)
}

func (d *Desk) IsSame(other StateItem) bool {
	if otherDesk, ok := other.(*Desk); ok {
		return otherDesk.Number == d.Number && otherDesk.Color == d.Color
	} else {
		return false
	}
}

func (d *Desk) String() string {
	return fmt.Sprintf("%s/%s", d.Id(), d.Color)
}

func (d *Desk) Create() error {
	fmt.Println("create", d)
	return nil
}

func (d *Desk) Remove() error {
	fmt.Println("remove", d)
	return nil
}

func (d *Desk) Update(from Actionable) error {
	fmt.Println("update from", from, "to", d)
	return nil
}

type Room struct {
	Number int
	Name string
	Desks []*Desk
}

func (r *Room) String() string {
	return fmt.Sprintf("room-%d/%s", r.Number, r.Name)
}

func (r *Room) Remove() error {
	fmt.Println("remove", r)
	return nil
}

func (r *Room) Create() error {
	fmt.Println("create", r)
	return nil
}

func (r *Room) Update(from Actionable) error {
	fmt.Println("update from", from, "to", r)
	return nil
}

func (r *Room) AsState() StateItem {
	parts := make([]StateItem, len(r.Desks)+1)
	for i := range r.Desks {
		parts[i] = r.Desks[i]
	}
	id := fmt.Sprintf("room-%d", r.Number)
	parts[len(parts)-1] = &StringStateItem{IdValue: fmt.Sprintf("%s-name", id), Value: r.Name, Actionable: r}
	return ComposedStateItem{
		IdValue: id,
		Parts:   parts,
	}
}

func ExampleInferActions() {
	officeRoom := &Room{
		Number: 1,
		Name: "Office",
		Desks: []*Desk{
			{Number: 42, Color: "brown"},
		},
	}
	livingRoom := &Room{
		Number: 2,
		Name: "Living",
		Desks: []*Desk{
			{Number: 42, Color: "blue"},
		},
	}

	officeRoomRecolored := &Room{
		Number: 1,
		Name: "Office",
		Desks: []*Desk{
			{Number: 42, Color: "green"},
		},
	}

	fmt.Println(">> color change")
	actions := InferActions(
		[]StateItem{officeRoom.AsState(), livingRoom.AsState()},
		[]StateItem{officeRoomRecolored.AsState(), livingRoom.AsState()},
	)
	for _, act := range actions	{
		_ = act()
	}

	bedroom := &Room{
		Number: 3,
		Name: "Bedroom",
	}

	fmt.Println(">> structure change")
	actions = InferActions(
		[]StateItem{officeRoom.AsState(), livingRoom.AsState()},
		[]StateItem{livingRoom.AsState(), bedroom.AsState()},
	)
	for _, act := range actions	{
		_ = act()
	}

	// Output:
	// >> color change
	// update from desk-42/brown to desk-42/green
	// >> structure change
	// remove desk-42/brown
	// remove room-1/Office
	// create room-3/Bedroom
}

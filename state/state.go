package state // import rmazur.io/overseer/state

import (
	"context"
	"fmt"
)

type Action func(ctx context.Context) error

type Actionable interface {
	Create(ctx context.Context) error
	Remove(ctx context.Context) error
	Update(ctx context.Context, from Actionable) error
}

type UpdateAction func(ctx context.Context, prev Actionable) error

type Item interface {
	Actionable

	Id() string
	IsSame(item Item) bool
}

func InferActions(prev []Item, next []Item) []Action {
	nextState := mapState(next)

	removeActions := make([]Action, 0, len(prev))
	updateActions := make([]Action, 0, len(next))
	for _, prevItem := range prev {
		if nextItem, present := nextState[prevItem.Id()]; present {
			if !nextItem.IsSame(prevItem) {
				nextItem := nextItem
				prevItem := prevItem
				updateActions = append(updateActions, func(ctx context.Context) error {
					return nextItem.Update(ctx, prevItem)
				})
			}
			delete(nextState, prevItem.Id())
		} else {
			removeActions = append(removeActions, prevItem.Remove)
		}
	}

	actions := make([]Action, len(removeActions)+len(updateActions)+len(nextState))
	copy(actions, removeActions)
	copy(actions[len(removeActions):], updateActions)

	createActions := actions[len(removeActions)+len(updateActions):]
	i := 0
	for _, nextItem := range nextState {
		createActions[i] = nextItem.Create
		i++
	}
	return actions
}

func mapState(items []Item) map[string]Item {
	itemsMap := make(map[string]Item, len(items))
	for _, item := range items {
		itemsMap[item.Id()] = item
	}
	return itemsMap
}

type StringItem struct {
	Actionable

	IdValue string
	Value   string
}

func (ssi *StringItem) Id() string {
	return ssi.IdValue
}

func (ssi *StringItem) IsSame(another Item) bool {
	if assi, ok := another.(*StringItem); ok {
		return ssi.IdValue == assi.IdValue && ssi.Value == assi.Value
	} else {
		return false
	}
}

type ItemId interface {
	String() string
}

type StringId string

func (sid StringId) String() string {
	return string(sid)
}

type ComposedItem struct {
	IdValue ItemId
	Parts   []Item
}

func (csi ComposedItem) Id() string {
	return csi.IdValue.String()
}

func (csi ComposedItem) IsSame(another Item) bool {
	if another.Id() != csi.Id() {
		return false
	}
	if acsi, ok := another.(ComposedItem); ok {
		if len(acsi.Parts) != len(csi.Parts) {
			return false
		}
		anotherState := mapState(acsi.Parts)
		for _, part := range csi.Parts {
			if !part.IsSame(anotherState[part.Id()]) {
				return false
			}
		}
		return true
	} else {
		return false
	}
}

func (csi ComposedItem) Create(ctx context.Context) error {
	for _, part := range csi.Parts {
		if err := part.Create(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (csi ComposedItem) Remove(ctx context.Context) error {
	for _, part := range csi.Parts {
		if err := part.Remove(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (csi ComposedItem) Update(ctx context.Context, from Actionable) error {
	fromCsi, ok := from.(ComposedItem)
	if !ok {
		panic(fmt.Errorf("bad composition: %s is not a ComposedItem", from))
	}
	actions := InferActions(fromCsi.Parts, csi.Parts)
	for _, act := range actions {
		if err := act(ctx); err != nil {
			return err
		}
	}
	return nil
}

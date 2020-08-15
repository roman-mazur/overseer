package state // import rmazur.io/overseer/state

import (
	"context"
	"fmt"
)

type Item interface {
	Actionable

	Id() string
	IsSame(item Item) bool
}

type Set []Item

type ComposedAction []Action

func (ca ComposedAction) Do(ctx context.Context) error {
	for _, act := range ca {
		if err := act.Do(ctx); err != nil {
			return err
		}
	}
	return nil
}

func InferActions(prev, next Set) Action {
	nextState := mapState(next)

	removeActions := make([]Action, 0, len(prev))
	updateActions := make([]Action, 0, len(next))
	for _, prevItem := range prev {
		if nextItem, present := nextState[prevItem.Id()]; present {
			if !nextItem.IsSame(prevItem) {
				nextItem := nextItem
				prevItem := prevItem
				updateActions = append(updateActions, ActionFunc(func(ctx context.Context) error {
					return nextItem.Update(ctx, prevItem)
				}))
			}
			delete(nextState, prevItem.Id())
		} else {
			removeActions = append(removeActions, ActionFunc(prevItem.Remove))
		}
	}

	actions := make([]Action, len(removeActions)+len(updateActions)+len(nextState))
	copy(actions, removeActions)
	copy(actions[len(removeActions):], updateActions)

	createActions := actions[len(removeActions)+len(updateActions):]
	i := 0
	for _, nextItem := range nextState {
		createActions[i] = ActionFunc(nextItem.Create)
		i++
	}
	return ComposedAction(actions)
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

type ComposedItem struct {
	IdValue ItemId
	Parts   []Item

	actions  Actionable
	original interface{}
}

func (csi ComposedItem) Id() string {
	return csi.IdValue.String()
}

func (csi ComposedItem) IsSame(another Item) bool {
	if another == nil {
		panic(csi.Id() + " is being compared to nil")
	}
	if another.Id() != csi.Id() {
		return false
	}
	if acsi, ok := another.(ComposedItem); ok {
		if csi.actions != nil {
			if acsi.actions == nil {
				return false
			}
			if item, ok := csi.actions.(Item); ok {
				if otherItem, ok := acsi.actions.(Item); ok {
					if !item.IsSame(otherItem) {
						return false
					}
				} else {
					return false
				}
			}
		} else if acsi.actions != nil {
			return false
		}

		if len(acsi.Parts) != len(csi.Parts) {
			return false
		}
		anotherState := mapState(acsi.Parts)
		for _, part := range csi.Parts {
			if anotherPart, present := anotherState[part.Id()]; !present || !part.IsSame(anotherPart) {
				return false
			}
			delete(anotherState, part.Id())
		}
		if len(anotherState) > 0 {
			return false
		}
		return true
	} else {
		return false
	}
}

func (csi ComposedItem) Create(ctx context.Context) error {
	if csi.actions != nil {
		err := csi.actions.Create(ctx)
		if err != nil {
			return err
		}
	}
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
	if csi.actions != nil {
		return csi.actions.Remove(ctx)
	}
	return nil
}

func (csi ComposedItem) Update(ctx context.Context, from interface{}) error {
	fromCsi, ok := from.(ComposedItem)
	if !ok {
		panic(fmt.Errorf("bad composition: %s is not a ComposedItem", from))
	}
	if err := InferActions(fromCsi.Parts, csi.Parts).Do(ctx); err != nil {
		return err
	}
	if csi.actions != nil {
		prev := from
		if fromCsi.original != nil {
			prev = fromCsi.original
		}
		return csi.actions.Update(ctx, prev)
	}
	return nil
}

func (csi ComposedItem) String() string {
	return fmt.Sprint("{id:", csi.Id(), ", actions:", csi.actions != nil, ", parts:", csi.Parts, "}")
}

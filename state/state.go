package state // import rmazur.io/overseer/state

import (
	"fmt"
)

type Action func() error

type Actionable interface {
	Create() error
	Remove() error
	Update(from Actionable) error
}

type StateItem interface {
	Actionable

	Id() string
	IsSame(item StateItem) bool
}

func InferActions(prev []StateItem, next []StateItem) []Action {
	nextState := mapState(next)

	removeActions := make([]Action, 0, len(prev))
	updateActions := make([]Action, 0, len(next))
	for _, prevItem := range prev {
		if nextItem, present := nextState[prevItem.Id()]; present {
			if !nextItem.IsSame(prevItem) {
				nextItem := nextItem
				prevItem := prevItem
				updateActions = append(updateActions, func() error { return nextItem.Update(prevItem) })
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

func mapState(items []StateItem) map[string]StateItem {
	itemsMap := make(map[string]StateItem, len(items))
	for _, item := range items {
		itemsMap[item.Id()] = item
	}
	return itemsMap
}

type StringStateItem struct {
	Actionable

	IdValue string
	Value   string
}

func (ssi *StringStateItem) Id() string {
	return ssi.IdValue
}

func (ssi *StringStateItem) IsSame(another StateItem) bool {
	if assi, ok := another.(*StringStateItem); ok {
		return ssi.IdValue == assi.IdValue && ssi.Value == assi.Value
	} else {
		return false
	}
}

type ComposedStateItem struct {
	IdValue string
	Parts   []StateItem
}

func (csi ComposedStateItem) Id() string {
	return csi.IdValue
}

func (csi ComposedStateItem) IsSame(another StateItem) bool {
	if another.Id() != csi.Id() {
		return false
	}
	if acsi, ok := another.(ComposedStateItem); ok {
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

func (csi ComposedStateItem) Create() error {
	for _, part := range csi.Parts {
		if err := part.Create(); err != nil {
			return err
		}
	}
	return nil
}

func (csi ComposedStateItem) Remove() error {
	for _, part := range csi.Parts {
		if err := part.Remove(); err != nil {
			return err
		}
	}
	return nil
}

func (csi ComposedStateItem) Update(from Actionable) error {
	fromCsi, ok := from.(ComposedStateItem)
	if !ok {
		panic(fmt.Errorf("bad composition: %s is not a ComposedStateItem", from))
	}
	actions := InferActions(fromCsi.Parts, csi.Parts)
	for _, act := range actions {
		if err := act(); err != nil {
			return err
		}
	}
	return nil
}

package state

import "context"

// Action represents operations to be performed to move from one state to another.
type Action interface {
	Do(ctx context.Context) error
}

// ActionFunc is an Action implemented with a single function.
type ActionFunc func(ctx context.Context) error

func (af ActionFunc) Do(ctx context.Context) error {
	return af(ctx)
}

// Actionable represents CRU operations that can be performed on some object to achieve the target state.
type Actionable interface {
	Create(ctx context.Context) error
	Remove(ctx context.Context) error
	Update(ctx context.Context, from interface{}) error
}

type updateActionFunc func(ctx context.Context, prev interface{}) error

type noAction struct{}

func (na noAction) Create(context.Context) error              { return nil }
func (na noAction) Remove(context.Context) error              { return nil }
func (na noAction) Update(context.Context, interface{}) error { return nil }

var (
	noop Actionable = noAction{}
)

type actions struct {
	wrapped Actionable

	create, remove       ActionFunc
	update, parentUpdate updateActionFunc
}

func callAction(a ActionFunc, ctx context.Context) error {
	if a != nil {
		return a(ctx)
	}
	return nil
}

func (a actions) Create(ctx context.Context) error {
	if err := callAction(a.create, ctx); err != nil {
		return err
	}
	if a.wrapped != nil {
		return a.wrapped.Create(ctx)
	}
	return nil
}
func (a actions) Remove(ctx context.Context) error {
	if a.wrapped != nil {
		if err := a.wrapped.Remove(ctx); err != nil {
			return err
		}
	}
	return callAction(a.remove, ctx)
}
func (a actions) Update(ctx context.Context, prev interface{}) error {
	if a.update != nil {
		if err := a.update(ctx, prev); err != nil {
			return err
		}
	}
	if a.parentUpdate != nil {
		if err := a.parentUpdate(ctx, prev); err != nil {
			return err
		}
	}
	if a.wrapped != nil {
		return a.wrapped.Update(ctx, prev)
	}
	return nil
}

func (a actions) isNoop() bool {
	return a.create == nil && a.remove == nil && a.update == nil && a.wrapped == nil && a.parentUpdate == nil
}

func (a actions) isWrapperOnly() bool {
	return a.create == nil && a.remove == nil && a.update == nil && a.parentUpdate == nil && a.wrapped != nil
}

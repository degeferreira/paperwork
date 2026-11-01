package integration

import (
	"context"
	"errors"
	"fmt"
)

func (s *Service) RunJob(ctx context.Context, name string, fn func(context.Context) error) error {
	e := fn(ctx)
	if errors.Is(e, context.Canceled) {
		return e
	}
	if e == nil {
		return s.State.Set("job:"+name, Object{"last_success": now(), "error": nil})
	}
	previous, se := s.State.Get("job:"+name, Object{})
	if se != nil {
		return se
	}
	message := "Error"
	var known *Error
	if errors.As(e, &known) {
		message = known.Detail
	} else {
		message = fmt.Sprintf("%T", e)
	}
	if se = s.State.Set("job:"+name, Object{"last_success": obj(previous)["last_success"], "error": message, "last_error": now()}); se != nil {
		return se
	}
	return e
}

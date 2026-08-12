package jobs

import "context"

// Handler executes a claimed job.
type Handler interface {
	Handle(ctx context.Context, job Job) error
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(ctx context.Context, job Job) error

func (f HandlerFunc) Handle(ctx context.Context, job Job) error {
	return f(ctx, job)
}

// Registry maps job types to handlers.
type Registry struct {
	handlers map[Type]Handler
}

func NewRegistry() *Registry {
	return &Registry{handlers: make(map[Type]Handler)}
}

func (r *Registry) Register(jobType Type, handler Handler) error {
	if err := ValidateType(jobType); err != nil {
		return err
	}
	if handler == nil {
		return ErrInvalidJobType
	}
	r.handlers[jobType] = handler
	return nil
}

func (r *Registry) Get(jobType Type) (Handler, bool) {
	h, ok := r.handlers[jobType]
	return h, ok
}

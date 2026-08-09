package audit

import "context"

// contextKey is a private type to avoid collisions.
type contextKey struct {
	// uintptr field ensures each key variable is a distinct instance,
	// avoiding Go's empty-struct deduplication.
	id uintptr
}

// uintptr keys — unique per variable.
var (
	actorKey    = contextKey{id: 1} // string
	resourceKey = contextKey{id: 2} // string
	opKey       = contextKey{id: 3} // OpLevel
)

// WithActor attaches an actor identifier to a context.
func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, actorKey, actor)
}

// WithResource attaches a resource identifier to a context.
func WithResource(ctx context.Context, resource string) context.Context {
	return context.WithValue(ctx, resourceKey, resource)
}

// WithOp attaches an operation level to a context.
func WithOp(ctx context.Context, op OpLevel) context.Context {
	return context.WithValue(ctx, opKey, op)
}

// ActorFrom returns the actor stored in ctx, or "system" if not set.
func ActorFrom(ctx context.Context) string {
	if v, ok := ctx.Value(actorKey).(string); ok && v != "" {
		return v
	}
	return "system"
}

// ResourceFrom returns the resource stored in ctx, or "unknown" if not set.
func ResourceFrom(ctx context.Context) string {
	if v, ok := ctx.Value(resourceKey).(string); ok && v != "" {
		return v
	}
	return "unknown"
}

// OpFrom returns the OpLevel stored in ctx, or OpRead if not set.
func OpFrom(ctx context.Context) OpLevel {
	if v, ok := ctx.Value(opKey).(OpLevel); ok {
		return v
	}
	return OpRead
}

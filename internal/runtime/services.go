// Package runtime provides execution runtime features for SLOP
// including service management, rate limiting, and transaction logging.
package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/standardbeagle/slop/internal/evaluator"
)

// ServiceWithContext extends the evaluator.Service interface with context support
// for proper cancellation and timeout handling.
type ServiceWithContext interface {
	evaluator.Service
	// CallWithContext invokes a method with context for cancellation/timeout.
	CallWithContext(ctx context.Context, method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error)
	// Name returns the service name.
	Name() string
	// Methods returns available method names.
	Methods() []string
	// Close releases resources.
	Close() error
}

// ServiceRegistry manages registered services by name.
type ServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]ServiceWithContext
}

// NewServiceRegistry creates a new service registry.
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]ServiceWithContext),
	}
}

// Register adds a service to the registry.
func (r *ServiceRegistry) Register(svc ServiceWithContext) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := svc.Name()
	if _, exists := r.services[name]; exists {
		return fmt.Errorf("service already registered: %s", name)
	}

	r.services[name] = svc
	return nil
}

// Get retrieves a service by name.
func (r *ServiceRegistry) Get(name string) (ServiceWithContext, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	svc, ok := r.services[name]
	return svc, ok
}

// GetAsEvaluatorService retrieves a service as an evaluator.Service.
func (r *ServiceRegistry) GetAsEvaluatorService(name string) (evaluator.Service, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	svc, ok := r.services[name]
	if !ok {
		return nil, false
	}
	return svc, true
}

// Unregister removes a service from the registry.
func (r *ServiceRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.services, name)
}

// List returns all registered service names.
func (r *ServiceRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.services))
	for name := range r.services {
		names = append(names, name)
	}
	return names
}

// CloseAll closes all registered services.
func (r *ServiceRegistry) CloseAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for _, svc := range r.services {
		if err := svc.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	r.services = make(map[string]ServiceWithContext)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing services: %v", errs)
	}
	return nil
}

// CreateServiceValue creates an evaluator.ServiceValue from a registered service.
func (r *ServiceRegistry) CreateServiceValue(name string) (*evaluator.ServiceValue, bool) {
	svc, ok := r.Get(name)
	if !ok {
		return nil, false
	}
	return &evaluator.ServiceValue{
		Name:    name,
		Service: svc,
	}, true
}

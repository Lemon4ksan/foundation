// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrAlreadyStarted is returned when attempting to initialize or register
	// new services on an orchestrator that is already active.
	ErrAlreadyStarted = errors.New("lifecycle: orchestrator is already active")

	// ErrNotInitialized is returned by [Orchestrator.StartAll] if there're no services to start.
	ErrNotInitialized = errors.New("lifecycle: orchestrator must be initialized before starting")
)

// Service defines the lifecycle contract for a system module or background worker.
//
// Implementing types must define non-blocking or gracefully terminable routines
// for initialization, startup, and shutdown.
type Service interface {
	// Name returns the unique identifier for the service.
	// This name is used as the key for dependency resolution.
	Name() string

	// Init performs early, synchronous, and non-blocking setup (e.g., allocating
	// local buffers, parsing configs).
	Init(ctx context.Context) error

	// Start executes the main business logic of the service.
	// Long-running background processes must spawn their own goroutines inside Start
	// and return nil immediately.
	Start(ctx context.Context) error

	// Stop gracefully terminates any running background goroutines and releases
	// all allocated system resources (such as file handles or network sockets).
	Stop(ctx context.Context) error
}

// Dependent defines an optional interface extension for services that declare
// explicit, structural dependencies on other services.
//
// Implementing this interface enables [Orchestrator] to perform topological
// dependency sorting during the initialization phase.
type Dependent interface {
	Service
	// Dependencies returns a slice containing the unique names of the services
	// that this service depends on.
	Dependencies() []string
}

// Orchestrator coordinates the initialization, startup, and graceful shutdown
// of registered [Service] instances.
//
// It resolves dependencies topologically, ensuring that parent services are
// online before dependent services start, and dependent services are shut down
// before their parents go offline.
//
// An Orchestrator is safe for concurrent use by multiple goroutines.
type Orchestrator struct {
	mu       sync.RWMutex
	services map[string]Service
	ordered  []Service
	running  []Service
	started  bool
}

// NewOrchestrator initializes and returns a new [Orchestrator] instance.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		services: make(map[string]Service),
	}
}

// All returns a newly allocated slice containing a shallow copy of all currently
// registered services in the orchestrator.
//
// It is non-blocking and safe to call concurrently while the orchestrator is running.
//
// # Complexity
//
// Time Complexity: O(N), where N is the number of registered services.
// Space Complexity: O(N) allocation for the returned slice.
func (o *Orchestrator) All() []Service {
	o.mu.RLock()
	defer o.mu.RUnlock()

	res := make([]Service, 0, len(o.services))
	for _, mod := range o.services {
		res = append(res, mod)
	}

	return res
}

// Register adds a [Service] to the orchestrator's managed set.
//
// # Preconditions
//
// The orchestrator must not have been started yet. Attempting to register
// a service on an already running orchestrator is ignored and logs a warning.
func (o *Orchestrator) Register(s Service) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.started {
		return
	}

	o.services[s.Name()] = s
}

// InitAll performs topological sorting of all registered services and executes
// their [Service.Init] routines in order.
//
// # Preconditions
//
// All dependent services must be registered. If circular dependencies are detected,
// or if any required dependency is missing, InitAll aborts and returns an error
// before invoking any initialization.
//
// # Postconditions
//
// If successful, the orchestrator's internal execution path is locked to the
// resolved topological order.
//
// # Complexity
//
// Time Complexity: O(V + E) for dependency resolution, where V is the number
// of services and E is the number of dependency links.
func (o *Orchestrator) InitAll(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.started {
		return ErrAlreadyStarted
	}

	ordered, err := o.sortDependencies()
	if err != nil {
		return fmt.Errorf("lifecycle: dependency resolution: %w", err)
	}

	o.ordered = ordered

	for _, s := range o.ordered {
		if err := s.Init(ctx); err != nil {
			return fmt.Errorf("lifecycle: init service %q failed: %w", s.Name(), err)
		}
	}

	return nil
}

// StartAll executes the startup routine for all initialized services sequentially
// in topological order.
//
// If starting any service fails, StartAll automatically triggers a rollback,
// gracefully stopping all previously started services in reverse topological
// order before returning the error.
//
// Returns ErrNotInitialized if there're no services to start.
//
// # Preconditions
//
// [Orchestrator.InitAll] must have been successfully invoked first. Calling
// StartAll on an already running orchestrator is a safe, idempotent no-op
// and returns nil.
func (o *Orchestrator) StartAll(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.started {
		return nil
	}

	if len(o.ordered) == 0 && len(o.services) > 0 {
		return ErrNotInitialized
	}

	for _, s := range o.ordered {
		if err := s.Start(ctx); err != nil {
			o.stopRunning(context.Background())
			return fmt.Errorf("lifecycle: start service %q failed: %w", s.Name(), err)
		}

		o.running = append(o.running, s)
	}

	o.started = true

	return nil
}

// StopAll gracefully terminates all active services in reverse topological order.
//
// This ensures that dependent high-level modules are offline before their
// underlying low-level parent dependencies are stopped. Calling StopAll on
// an inactive orchestrator is a safe, idempotent no-op.
func (o *Orchestrator) StopAll(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.stopRunning(ctx)
	o.started = false

	return nil
}

// stopRunning is an internal helper that stops active services in reverse order.
// The caller must hold o.mu's write lock.
func (o *Orchestrator) stopRunning(ctx context.Context) {
	for i := len(o.running) - 1; i >= 0; i-- {
		s := o.running[i]
		_ = s.Stop(ctx)
	}

	o.running = nil
}

// sortDependencies performs topological sorting using DFS and detects dependency issues.
// The caller must hold o.mu's lock.
func (o *Orchestrator) sortDependencies() ([]Service, error) {
	visited := make(map[string]bool)
	temp := make(map[string]bool)

	var sorted []Service

	var visit func(name string) error

	visit = func(name string) error {
		if temp[name] {
			return fmt.Errorf("circular dependency detected involving %q", name)
		}

		if !visited[name] {
			temp[name] = true

			s, exists := o.services[name]
			if !exists {
				return fmt.Errorf("dependency %q is not registered", name)
			}

			if dep, ok := s.(Dependent); ok {
				for _, depName := range dep.Dependencies() {
					if err := visit(depName); err != nil {
						return err
					}
				}
			}

			temp[name] = false
			visited[name] = true

			sorted = append(sorted, s)
		}

		return nil
	}

	for name := range o.services {
		if !visited[name] {
			if err := visit(name); err != nil {
				return nil, err
			}
		}
	}

	return sorted, nil
}

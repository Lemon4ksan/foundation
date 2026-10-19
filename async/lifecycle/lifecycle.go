// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
)

var (
	// ErrAlreadyStarted is returned when attempting to initialize or register new services on an orchestrator that is already active.
	ErrAlreadyStarted = errors.New("lifecycle: orchestrator is already active")

	// ErrNotInitialized is returned by [Orchestrator.StartAll] if there are no services to start.
	ErrNotInitialized = errors.New("lifecycle: orchestrator must be initialized before starting")
)

// Service defines the lifecycle contract for a system module or background worker.
//
// Implementing types must define non-blocking or gracefully terminable routines
// for initialization, startup, and shutdown.
type Service interface {
	// Name returns the unique identifier for the service.
	Name() string

	// Init performs early, synchronous, and non-blocking setup.
	Init(ctx context.Context) error

	// Start executes the main business logic of the service.
	Start(ctx context.Context) error

	// Stop gracefully terminates any running background goroutines and releases resources.
	Stop(ctx context.Context) error
}

// Dependent defines an optional interface extension for services that declare explicit, structural dependencies on other services.
type Dependent interface {
	Service
	// Dependencies returns a slice containing the unique names of the services that this service depends on.
	Dependencies() []string
}

// Orchestrator coordinates the initialization, startup, and graceful shutdown of registered [Service] instances.
//
// The zero value is empty and not ready to use; use [NewOrchestrator] to initialize.
// All exported methods manage their own internal synchronization and are safe for concurrent use.
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

// All returns a newly allocated slice containing a shallow copy of all currently registered services in the orchestrator.
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
func (o *Orchestrator) Register(s Service) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.started {
		return
	}

	o.services[s.Name()] = s
}

// InitAll performs topological sorting of all registered services and executes their [Service.Init] routines in order.
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

// StartAll executes the startup routine for all initialized services sequentially in topological order.
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
	for _, s := range slices.Backward(o.running) {
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

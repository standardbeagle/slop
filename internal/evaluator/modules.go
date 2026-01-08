package evaluator

import (
	"fmt"

	"github.com/standardbeagle/slop/internal/ast"
)

// ModuleResolver handles module dependency resolution and wiring.
type ModuleResolver struct {
	// All SOURCE modules indexed by their ID
	sourcesByID map[string]*ast.Module

	// All SOURCE modules indexed by their name (for fallback lookup)
	sourcesByName map[string]*ast.Module

	// All USE modules
	useModules []*ast.Module

	// The MAIN module (if any)
	mainModule *ast.Module

	// Resolved module scopes (module name -> scope with its definitions)
	scopes map[string]*Scope

	// Dependency graph for cycle detection
	dependencies map[string][]string
}

// NewModuleResolver creates a new module resolver.
func NewModuleResolver() *ModuleResolver {
	return &ModuleResolver{
		sourcesByID:   make(map[string]*ast.Module),
		sourcesByName: make(map[string]*ast.Module),
		useModules:    []*ast.Module{},
		scopes:        make(map[string]*Scope),
		dependencies:  make(map[string][]string),
	}
}

// ModuleError represents an error during module resolution.
type ModuleError struct {
	Module  string
	Message string
}

func (e *ModuleError) Error() string {
	return fmt.Sprintf("module %s: %s", e.Module, e.Message)
}

// LoadModules loads all modules from a program and organizes them.
func (r *ModuleResolver) LoadModules(modules []*ast.Module) error {
	for _, mod := range modules {
		switch mod.Type {
		case "SOURCE":
			// Index by ID if available, otherwise by name
			if mod.ID != "" {
				r.sourcesByID[mod.ID] = mod
			}
			r.sourcesByName[mod.Name] = mod

			// Record dependencies
			deps := []string{}
			for _, requiredID := range mod.Uses {
				deps = append(deps, requiredID)
			}
			if mod.ID != "" {
				r.dependencies[mod.ID] = deps
			} else {
				r.dependencies[mod.Name] = deps
			}

		case "USE":
			r.useModules = append(r.useModules, mod)

		case "MAIN":
			if r.mainModule != nil {
				return &ModuleError{Module: "MAIN", Message: "multiple MAIN modules defined"}
			}
			r.mainModule = mod
		}
	}
	return nil
}

// Validate checks that all dependencies are satisfied.
func (r *ModuleResolver) Validate() []error {
	var errors []error

	// Check all SOURCE module dependencies are satisfied
	for _, mod := range r.sourcesByID {
		for localName, requiredID := range mod.Uses {
			if r.sourcesByID[requiredID] == nil {
				errors = append(errors, &ModuleError{
					Module:  mod.Name,
					Message: fmt.Sprintf("requires '%s' (%s) but not found", localName, requiredID),
				})
			}
		}
	}

	// Check all USE module references can be resolved
	for _, use := range r.useModules {
		moduleName := use.Name

		// Apply with clauses to find the actual module
		resolved := false
		if r.sourcesByID[moduleName] != nil || r.sourcesByName[moduleName] != nil {
			resolved = true
		}

		if !resolved {
			errors = append(errors, &ModuleError{
				Module:  moduleName,
				Message: "module not found",
			})
		}
	}

	// Check for circular dependencies
	cycles := r.detectCycles()
	for _, cycle := range cycles {
		errors = append(errors, &ModuleError{
			Module:  cycle[0],
			Message: fmt.Sprintf("circular dependency: %v", cycle),
		})
	}

	return errors
}

// detectCycles finds circular dependencies in the module graph.
func (r *ModuleResolver) detectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, dep := range r.dependencies[node] {
			if !visited[dep] {
				if dfs(dep) {
					return true
				}
			} else if recStack[dep] {
				// Found cycle - extract it from path
				cycleStart := 0
				for i, n := range path {
					if n == dep {
						cycleStart = i
						break
					}
				}
				cycle := make([]string, len(path)-cycleStart)
				copy(cycle, path[cycleStart:])
				cycles = append(cycles, cycle)
				return true
			}
		}

		path = path[:len(path)-1]
		recStack[node] = false
		return false
	}

	for node := range r.dependencies {
		if !visited[node] {
			dfs(node)
		}
	}

	return cycles
}

// ResolveModule finds a module by ID or name.
func (r *ModuleResolver) ResolveModule(nameOrID string) *ast.Module {
	if mod := r.sourcesByID[nameOrID]; mod != nil {
		return mod
	}
	return r.sourcesByName[nameOrID]
}

// BuildScopes creates scopes for all SOURCE modules.
func (r *ModuleResolver) BuildScopes(e *Evaluator) error {
	// Process modules in dependency order
	processed := make(map[string]bool)

	var processModule func(mod *ast.Module) error
	processModule = func(mod *ast.Module) error {
		modKey := mod.ID
		if modKey == "" {
			modKey = mod.Name
		}

		if processed[modKey] {
			return nil
		}

		// First process all dependencies
		for localName, requiredID := range mod.Uses {
			depMod := r.ResolveModule(requiredID)
			if depMod == nil {
				return &ModuleError{
					Module:  mod.Name,
					Message: fmt.Sprintf("dependency '%s' (%s) not found", localName, requiredID),
				}
			}
			if err := processModule(depMod); err != nil {
				return err
			}
		}

		// Create scope for this module
		scope := NewScope()

		// Import dependencies into scope
		for localName, requiredID := range mod.Uses {
			depScope := r.scopes[requiredID]
			if depScope == nil {
				// Try by name
				depMod := r.ResolveModule(requiredID)
				if depMod != nil {
					depScope = r.scopes[depMod.Name]
				}
			}
			if depScope != nil {
				// Create a module value that wraps the dependency scope
				moduleVal := &ModuleValue{Name: localName, Scope: depScope}
				scope.Set(localName, moduleVal)
			}
		}

		// Evaluate module body to populate scope
		moduleCtx := NewContext()
		moduleCtx.Scope = scope

		// Save original context and set module context
		origCtx := e.ctx
		e.ctx = moduleCtx

		// Evaluate each statement in the module body
		for _, stmt := range mod.Body {
			_, err := e.Eval(stmt)
			if err != nil {
				e.ctx = origCtx
				return &ModuleError{
					Module:  mod.Name,
					Message: fmt.Sprintf("error evaluating: %v", err),
				}
			}
		}

		// Restore original context
		e.ctx = origCtx

		// Store the scope
		if mod.ID != "" {
			r.scopes[mod.ID] = scope
		}
		r.scopes[mod.Name] = scope
		processed[modKey] = true

		return nil
	}

	// Process all SOURCE modules
	for _, mod := range r.sourcesByID {
		if err := processModule(mod); err != nil {
			return err
		}
	}
	for _, mod := range r.sourcesByName {
		if err := processModule(mod); err != nil {
			return err
		}
	}

	return nil
}

// BuildMainScope creates the scope for the MAIN module with all USE modules wired in.
func (r *ModuleResolver) BuildMainScope() (*Scope, error) {
	scope := NewScope()

	// Wire in all USE modules
	for _, use := range r.useModules {
		moduleName := use.Name
		mod := r.ResolveModule(moduleName)
		if mod == nil {
			return nil, &ModuleError{
				Module:  moduleName,
				Message: "module not found",
			}
		}

		// Get the scope for this module
		modScope := r.scopes[mod.ID]
		if modScope == nil {
			modScope = r.scopes[mod.Name]
		}
		if modScope == nil {
			return nil, &ModuleError{
				Module:  moduleName,
				Message: "module scope not built",
			}
		}

		// Apply with clause remapping
		finalScope := modScope
		if len(use.WithClauses) > 0 {
			// Create a new scope with remapped dependencies
			finalScope = NewScope()
			// Copy all values from original scope
			for k, v := range modScope.store {
				finalScope.Set(k, v)
			}
			// Apply remapping
			for localName, remappedID := range use.WithClauses {
				remappedMod := r.ResolveModule(remappedID)
				if remappedMod != nil {
					remappedScope := r.scopes[remappedMod.ID]
					if remappedScope == nil {
						remappedScope = r.scopes[remappedMod.Name]
					}
					if remappedScope != nil {
						finalScope.Set(localName, &ModuleValue{Name: localName, Scope: remappedScope})
					}
				}
			}
		}

		// Add to main scope using the module name (without path/version)
		shortName := getShortName(mod.Name)
		scope.Set(shortName, &ModuleValue{Name: shortName, Scope: finalScope})
	}

	return scope, nil
}

// GetMainModule returns the MAIN module if defined.
func (r *ModuleResolver) GetMainModule() *ast.Module {
	return r.mainModule
}

// getShortName extracts a short name from a module path like "mycompany/utils@v1"
func getShortName(name string) string {
	// Remove version suffix
	if idx := findLast(name, '@'); idx != -1 {
		name = name[:idx]
	}
	// Get last path component
	if idx := findLast(name, '/'); idx != -1 {
		name = name[idx+1:]
	}
	return name
}

func findLast(s string, ch rune) int {
	for i := len(s) - 1; i >= 0; i-- {
		if rune(s[i]) == ch {
			return i
		}
	}
	return -1
}

// ModuleValue represents a module as a runtime value.
type ModuleValue struct {
	Name  string
	Scope *Scope
}

func (m *ModuleValue) Type() string    { return "module" }
func (m *ModuleValue) String() string  { return fmt.Sprintf("<module %s>", m.Name) }
func (m *ModuleValue) IsTruthy() bool  { return true }

// Get returns a value from the module's scope.
func (m *ModuleValue) Get(name string) (Value, bool) {
	return m.Scope.Get(name)
}

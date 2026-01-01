package evaluator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionLog(t *testing.T) {
	t.Run("log operation", func(t *testing.T) {
		log := NewTransactionLog()

		id := log.Log(Operation{
			Type:    "call",
			Service: "db",
			Method:  "insert",
		})

		assert.Equal(t, int64(1), id)
		assert.Equal(t, 1, log.Size())

		ops := log.GetOperations()
		require.Len(t, ops, 1)
		assert.Equal(t, "call", ops[0].Type)
		assert.Equal(t, "db", ops[0].Service)
		assert.Equal(t, "insert", ops[0].Method)
	})

	t.Run("log multiple operations", func(t *testing.T) {
		log := NewTransactionLog()

		log.Log(Operation{Type: "call", Service: "db", Method: "insert"})
		log.Log(Operation{Type: "call", Service: "db", Method: "update"})
		log.Log(Operation{Type: "call", Service: "api", Method: "send"})

		assert.Equal(t, 3, log.Size())
	})

	t.Run("LogCall helper", func(t *testing.T) {
		log := NewTransactionLog()

		args := []Value{&StringValue{Value: "test"}}
		kwargs := map[string]Value{"id": &IntValue{Value: 1}}
		result := &MapValue{Pairs: map[string]Value{}, Order: []string{}}

		id := log.LogCall("db", "query", args, kwargs, result, nil)
		assert.Equal(t, int64(1), id)

		ops := log.GetOperations()
		require.Len(t, ops, 1)
		assert.Equal(t, "call", ops[0].Type)
		assert.Equal(t, "db", ops[0].Service)
		assert.Equal(t, "query", ops[0].Method)
		require.Len(t, ops[0].Args, 1)
		require.Len(t, ops[0].Kwargs, 1)
		assert.NotNil(t, ops[0].Result)
		assert.Nil(t, ops[0].Error)
	})

	t.Run("clear log", func(t *testing.T) {
		log := NewTransactionLog()
		log.Log(Operation{Type: "call"})
		log.Log(Operation{Type: "call"})

		assert.Equal(t, 2, log.Size())

		log.Clear()
		assert.Equal(t, 0, log.Size())
	})

	t.Run("get reversible operations", func(t *testing.T) {
		log := NewTransactionLog()

		log.Log(Operation{Type: "call", Reversible: true})
		log.Log(Operation{Type: "call", Reversible: false})
		log.Log(Operation{Type: "call", Reversible: true})
		log.Log(Operation{Type: "call", Reversible: true, Error: assert.AnError})

		reversible := log.GetReversibleOperations()
		assert.Len(t, reversible, 2) // 2 reversible without errors
	})
}

// MockReversibleService implements ReversibleService for testing.
type MockReversibleService struct {
	name       string
	undoCalls  []string
	undoErrors map[string]error
}

func (m *MockReversibleService) Call(method string, args []Value, kwargs map[string]Value) (Value, error) {
	return NONE, nil
}

func (m *MockReversibleService) Undo(method string, undoData map[string]Value) error {
	m.undoCalls = append(m.undoCalls, method)
	if m.undoErrors != nil {
		if err, ok := m.undoErrors[method]; ok {
			return err
		}
	}
	return nil
}

func (m *MockReversibleService) IsReversible(method string) bool {
	return method == "insert" || method == "update" || method == "delete"
}

func TestRollback(t *testing.T) {
	t.Run("rollback reversible operations", func(t *testing.T) {
		log := NewTransactionLog()
		svc := &MockReversibleService{name: "db"}
		services := map[string]Service{"db": svc}

		log.Log(Operation{Service: "db", Method: "insert", Reversible: true})
		log.Log(Operation{Service: "db", Method: "update", Reversible: true})
		log.Log(Operation{Service: "db", Method: "select", Reversible: false})
		log.Log(Operation{Service: "db", Method: "delete", Reversible: true})

		rollback := NewRollback(log, services)
		errors := rollback.Execute()

		assert.Empty(t, errors)

		// Should have undone in reverse order
		require.Len(t, svc.undoCalls, 3)
		assert.Equal(t, "delete", svc.undoCalls[0])
		assert.Equal(t, "update", svc.undoCalls[1])
		assert.Equal(t, "insert", svc.undoCalls[2])
	})

	t.Run("rollback with errors", func(t *testing.T) {
		log := NewTransactionLog()
		svc := &MockReversibleService{
			name:       "db",
			undoErrors: map[string]error{"update": assert.AnError},
		}
		services := map[string]Service{"db": svc}

		log.Log(Operation{Service: "db", Method: "insert", Reversible: true})
		log.Log(Operation{Service: "db", Method: "update", Reversible: true})

		rollback := NewRollback(log, services)
		errors := rollback.Execute()

		assert.Len(t, errors, 1) // One undo failed
		assert.Len(t, svc.undoCalls, 2) // Both were attempted
	})

	t.Run("rollback missing service", func(t *testing.T) {
		log := NewTransactionLog()
		services := map[string]Service{}

		log.Log(Operation{Service: "db", Method: "insert", Reversible: true})

		rollback := NewRollback(log, services)
		errors := rollback.Execute()

		assert.Len(t, errors, 1)
		assert.Contains(t, errors[0].Error(), "service not found")
	})
}

func TestServiceCallLogging(t *testing.T) {
	t.Run("service calls are logged", func(t *testing.T) {
		ctx := NewContext()
		eval := NewWithContext(ctx)

		// Register a mock service
		svc := &MockReversibleService{name: "db"}
		ctx.RegisterService("db", svc)

		// Define a variable with service
		ctx.Scope.Set("result", NONE)

		// Get the service and call a method
		svcVal, _ := ctx.Globals.Get("db")
		require.NotNil(t, svcVal)

		// Create bound method
		serviceVal := svcVal.(*ServiceValue)
		bound := &BoundMethodValue{
			ServiceName: serviceVal.Name,
			Service:     serviceVal.Service,
			Method:      "query",
		}

		// Call through evaluator
		_, err := eval.callServiceMethod(bound, []Value{&StringValue{Value: "SELECT *"}}, nil)
		require.NoError(t, err)

		// Check transaction log
		ops := ctx.TxLog.GetOperations()
		require.Len(t, ops, 1)
		assert.Equal(t, "call", ops[0].Type)
		assert.Equal(t, "db", ops[0].Service)
		assert.Equal(t, "query", ops[0].Method)
		require.Len(t, ops[0].Args, 1)
	})
}

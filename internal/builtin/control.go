package builtin

import (
	"fmt"
	"log"
	"os"

	"github.com/anthropics/slop/internal/evaluator"
)

// Store is a simple in-memory key-value store.
// In a real implementation, this would be backed by persistent storage.
var globalStore = make(map[string]evaluator.Value)

func (r *Registry) registerControlFunctions() {
	// Logging
	r.Register("log_debug", builtinLogDebug)
	r.Register("log_info", builtinLogInfo)
	r.Register("log_warn", builtinLogWarn)
	r.Register("log_error", builtinLogError)

	// Storage
	r.Register("store_get", builtinStoreGet)
	r.Register("store_set", builtinStoreSet)
	r.Register("store_delete", builtinStoreDelete)
	r.Register("store_exists", builtinStoreExists)
	r.Register("store_keys", builtinStoreKeys)

	// Environment
	r.Register("env_get", builtinEnvGet)
	r.Register("env_mode", builtinEnvMode)
	r.Register("env_debug", builtinEnvDebug)

	// Assertions
	r.Register("assert", builtinAssert)
	r.Register("assert_eq", builtinAssertEq)
	r.Register("assert_ne", builtinAssertNe)
	r.Register("assert_true", builtinAssertTrue)
	r.Register("assert_false", builtinAssertFalse)
	r.Register("assert_none", builtinAssertNone)
	r.Register("assert_not_none", builtinAssertNotNone)

	// Error handling
	r.Register("error", builtinError)
}

// Logging functions

func builtinLogDebug(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireMinArgs("log_debug", args, 1); err != nil {
		return nil, err
	}

	msg := args[0].String()
	if len(args) > 1 || len(kwargs) > 0 {
		data := formatLogData(args[1:], kwargs)
		log.Printf("[DEBUG] %s %s", msg, data)
	} else {
		log.Printf("[DEBUG] %s", msg)
	}

	return evaluator.NONE, nil
}

func builtinLogInfo(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireMinArgs("log_info", args, 1); err != nil {
		return nil, err
	}

	msg := args[0].String()
	if len(args) > 1 || len(kwargs) > 0 {
		data := formatLogData(args[1:], kwargs)
		log.Printf("[INFO] %s %s", msg, data)
	} else {
		log.Printf("[INFO] %s", msg)
	}

	return evaluator.NONE, nil
}

func builtinLogWarn(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireMinArgs("log_warn", args, 1); err != nil {
		return nil, err
	}

	msg := args[0].String()
	if len(args) > 1 || len(kwargs) > 0 {
		data := formatLogData(args[1:], kwargs)
		log.Printf("[WARN] %s %s", msg, data)
	} else {
		log.Printf("[WARN] %s", msg)
	}

	return evaluator.NONE, nil
}

func builtinLogError(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireMinArgs("log_error", args, 1); err != nil {
		return nil, err
	}

	msg := args[0].String()
	if len(args) > 1 || len(kwargs) > 0 {
		data := formatLogData(args[1:], kwargs)
		log.Printf("[ERROR] %s %s", msg, data)
	} else {
		log.Printf("[ERROR] %s", msg)
	}

	return evaluator.NONE, nil
}

func formatLogData(args []evaluator.Value, kwargs map[string]evaluator.Value) string {
	result := ""
	if len(args) > 0 {
		result += args[0].String()
	}
	for k, v := range kwargs {
		if result != "" {
			result += " "
		}
		result += fmt.Sprintf("%s=%s", k, v.String())
	}
	return result
}

// Storage functions

func builtinStoreGet(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("store_get", args, 1, 2); err != nil {
		return nil, err
	}

	key, err := requireString("store_get", args[0])
	if err != nil {
		return nil, err
	}

	if val, ok := globalStore[key]; ok {
		return val, nil
	}

	if len(args) == 2 {
		return args[1], nil
	}

	return evaluator.NONE, nil
}

func builtinStoreSet(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("store_set", args, 2); err != nil {
		return nil, err
	}

	key, err := requireString("store_set", args[0])
	if err != nil {
		return nil, err
	}

	globalStore[key] = args[1]
	return evaluator.NONE, nil
}

func builtinStoreDelete(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("store_delete", args, 1); err != nil {
		return nil, err
	}

	key, err := requireString("store_delete", args[0])
	if err != nil {
		return nil, err
	}

	delete(globalStore, key)
	return evaluator.NONE, nil
}

func builtinStoreExists(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("store_exists", args, 1); err != nil {
		return nil, err
	}

	key, err := requireString("store_exists", args[0])
	if err != nil {
		return nil, err
	}

	if _, ok := globalStore[key]; ok {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinStoreKeys(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("store_keys", args, 0, 1); err != nil {
		return nil, err
	}

	var prefix string
	if len(args) == 1 {
		var err error
		prefix, err = requireString("store_keys", args[0])
		if err != nil {
			return nil, err
		}
	}

	keys := make([]evaluator.Value, 0)
	for k := range globalStore {
		if prefix == "" || len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			keys = append(keys, &evaluator.StringValue{Value: k})
		}
	}

	return &evaluator.ListValue{Elements: keys}, nil
}

// Environment functions

func builtinEnvGet(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("env_get", args, 1, 2); err != nil {
		return nil, err
	}

	name, err := requireString("env_get", args[0])
	if err != nil {
		return nil, err
	}

	val := os.Getenv(name)
	if val == "" {
		if len(args) == 2 {
			return args[1], nil
		}
		return evaluator.NONE, nil
	}

	return &evaluator.StringValue{Value: val}, nil
}

func builtinEnvMode(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("env_mode", args, 0); err != nil {
		return nil, err
	}

	// Check common environment variables
	env := os.Getenv("SLOP_ENV")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env == "" {
		env = os.Getenv("NODE_ENV") // Common fallback
	}
	if env == "" {
		env = "development"
	}

	return &evaluator.StringValue{Value: env}, nil
}

func builtinEnvDebug(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("env_debug", args, 0); err != nil {
		return nil, err
	}

	debug := os.Getenv("SLOP_DEBUG")
	if debug == "1" || debug == "true" {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

// Assertion functions

func builtinAssert(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("assert", args, 1, 2); err != nil {
		return nil, err
	}

	if !args[0].IsTruthy() {
		msg := "assertion failed"
		if len(args) == 2 {
			msgStr, err := requireString("assert", args[1])
			if err == nil {
				msg = msgStr
			}
		}
		return nil, fmt.Errorf("AssertionError: %s", msg)
	}

	return evaluator.NONE, nil
}

func builtinAssertEq(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("assert_eq", args, 2, 3); err != nil {
		return nil, err
	}

	if !evaluator.Equal(args[0], args[1]) {
		msg := fmt.Sprintf("assertion failed: %s != %s", args[0].String(), args[1].String())
		if len(args) == 3 {
			msgStr, err := requireString("assert_eq", args[2])
			if err == nil {
				msg = msgStr
			}
		}
		return nil, fmt.Errorf("AssertionError: %s", msg)
	}

	return evaluator.NONE, nil
}

func builtinAssertNe(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("assert_ne", args, 2, 3); err != nil {
		return nil, err
	}

	if evaluator.Equal(args[0], args[1]) {
		msg := fmt.Sprintf("assertion failed: %s == %s", args[0].String(), args[1].String())
		if len(args) == 3 {
			msgStr, err := requireString("assert_ne", args[2])
			if err == nil {
				msg = msgStr
			}
		}
		return nil, fmt.Errorf("AssertionError: %s", msg)
	}

	return evaluator.NONE, nil
}

func builtinAssertTrue(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("assert_true", args, 1, 2); err != nil {
		return nil, err
	}

	bv, ok := args[0].(*evaluator.BoolValue)
	if !ok || !bv.Value {
		msg := fmt.Sprintf("assertion failed: expected true, got %s", args[0].String())
		if len(args) == 2 {
			msgStr, err := requireString("assert_true", args[1])
			if err == nil {
				msg = msgStr
			}
		}
		return nil, fmt.Errorf("AssertionError: %s", msg)
	}

	return evaluator.NONE, nil
}

func builtinAssertFalse(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("assert_false", args, 1, 2); err != nil {
		return nil, err
	}

	bv, ok := args[0].(*evaluator.BoolValue)
	if !ok || bv.Value {
		msg := fmt.Sprintf("assertion failed: expected false, got %s", args[0].String())
		if len(args) == 2 {
			msgStr, err := requireString("assert_false", args[1])
			if err == nil {
				msg = msgStr
			}
		}
		return nil, fmt.Errorf("AssertionError: %s", msg)
	}

	return evaluator.NONE, nil
}

func builtinAssertNone(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("assert_none", args, 1, 2); err != nil {
		return nil, err
	}

	if !evaluator.IsNone(args[0]) {
		msg := fmt.Sprintf("assertion failed: expected none, got %s", args[0].String())
		if len(args) == 2 {
			msgStr, err := requireString("assert_none", args[1])
			if err == nil {
				msg = msgStr
			}
		}
		return nil, fmt.Errorf("AssertionError: %s", msg)
	}

	return evaluator.NONE, nil
}

func builtinAssertNotNone(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("assert_not_none", args, 1, 2); err != nil {
		return nil, err
	}

	if evaluator.IsNone(args[0]) {
		msg := "assertion failed: expected non-none value"
		if len(args) == 2 {
			msgStr, err := requireString("assert_not_none", args[1])
			if err == nil {
				msg = msgStr
			}
		}
		return nil, fmt.Errorf("AssertionError: %s", msg)
	}

	return evaluator.NONE, nil
}

// Error function

func builtinError(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireMinArgs("error", args, 1); err != nil {
		return nil, err
	}

	msg, err := requireString("error", args[0])
	if err != nil {
		return nil, err
	}

	// Include any additional data in the error
	if len(kwargs) > 0 {
		data := ""
		for k, v := range kwargs {
			if data != "" {
				data += ", "
			}
			data += fmt.Sprintf("%s=%s", k, v.String())
		}
		msg = fmt.Sprintf("%s (%s)", msg, data)
	}

	return nil, fmt.Errorf("%s", msg)
}

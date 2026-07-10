package evaluator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/standardbeagle/slop/internal/ast"
)

func UnmarshalMachineSnapshot(data []byte) (*MachineSnapshot, error) {
	var snapshot MachineSnapshot
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("decode machine snapshot: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("decode machine snapshot: trailing data")
	}
	if snapshot.Version != "1" || snapshot.Context == nil || len(snapshot.Frames) == 0 {
		return nil, fmt.Errorf("invalid machine snapshot header")
	}
	for i, frame := range snapshot.Frames {
		if frame.NodePath == "" || frame.Phase < 0 || frame.Phase > 8 || frame.Index < 0 {
			return nil, fmt.Errorf("invalid machine snapshot frame %d", i)
		}
		if len(frame.KwNames) > 100000 || len(frame.Values) > 100000 || len(frame.Kwargs) > 100000 {
			return nil, fmt.Errorf("machine snapshot frame %d is too large", i)
		}
	}
	return &snapshot, nil
}

type MachineSnapshot struct {
	Version     string                 `json:"version"`
	Frames      []MachineFrameSnapshot `json:"frames"`
	Context     *SerializedContext     `json:"context"`
	Last        *SerializedValue       `json:"last,omitempty"`
	Failure     string                 `json:"failure,omitempty"`
	Paused      bool                   `json:"paused,omitempty"`
	PauseReason string                 `json:"pause_reason,omitempty"`
	Breakpoints []int                  `json:"breakpoints,omitempty"`
}

type MachineFrameSnapshot struct {
	NodePath           string                      `json:"node_path"`
	Phase              int                         `json:"phase"`
	Index              int                         `json:"index"`
	Value              *SerializedValue            `json:"value,omitempty"`
	Values             []*SerializedValue          `json:"values,omitempty"`
	Kwargs             map[string]*SerializedValue `json:"kwargs,omitempty"`
	KwNames            []string                    `json:"kw_names,omitempty"`
	Service            string                      `json:"service,omitempty"`
	Method             string                      `json:"method,omitempty"`
	Failed             bool                        `json:"failed,omitempty"`
	BreakpointConsumed bool                        `json:"breakpoint_consumed,omitempty"`
}

func (m *Machine) Snapshot(program *ast.Program) (*MachineSnapshot, error) {
	paths, _ := machineNodePaths(program)
	s := NewSerializer(program)
	ctx, err := s.SerializeContext(m.ctx)
	if err != nil {
		return nil, err
	}
	out := &MachineSnapshot{Version: "1", Context: ctx, Paused: m.paused, PauseReason: m.pauseReason}
	if m.err != nil {
		out.Failure = m.err.Error()
	}
	failedCall := len(m.stack) > 0 && m.stack[len(m.stack)-1].failed && m.stack[len(m.stack)-1].service != nil
	if m.last != nil && !failedCall {
		out.Last, err = s.SerializeValue(m.last)
		if err != nil {
			return nil, err
		}
	}
	for line := range m.breakpoints {
		out.Breakpoints = append(out.Breakpoints, line)
	}
	sort.Ints(out.Breakpoints)
	for _, f := range m.stack {
		p, ok := paths[f.node]
		if !ok {
			return nil, fmt.Errorf("frame node %T is not in program", f.node)
		}
		fs := MachineFrameSnapshot{NodePath: p, Phase: f.phase, Index: f.index, KwNames: append([]string(nil), f.kwNames...), Failed: f.failed, BreakpointConsumed: f.breakpointConsumed}
		if f.value != nil {
			fs.Value, err = s.SerializeValue(f.value)
			if err != nil {
				return nil, err
			}
		}
		for _, v := range f.values {
			sv, e := s.SerializeValue(v)
			if e != nil {
				return nil, e
			}
			fs.Values = append(fs.Values, sv)
		}
		if len(f.kwargs) > 0 {
			fs.Kwargs = map[string]*SerializedValue{}
			for k, v := range f.kwargs {
				sv, e := s.SerializeValue(v)
				if e != nil {
					return nil, e
				}
				fs.Kwargs[k] = sv
			}
		}
		if f.service != nil {
			fs.Service = f.service.ServiceName
			fs.Method = f.service.Method
		}
		out.Frames = append(out.Frames, fs)
	}
	return out, nil
}

func RestoreMachine(program *ast.Program, snap *MachineSnapshot, builtins map[string]*BuiltinValue, services map[string]Service) (*Machine, *MachineCheckpoint, error) {
	if snap == nil || snap.Version != "1" || len(snap.Frames) == 0 {
		return nil, nil, fmt.Errorf("invalid machine snapshot")
	}
	_, nodes := machineNodePaths(program)
	d := NewDeserializer(program, builtins, services)
	ctx, err := d.DeserializeContext(snap.Context)
	if err != nil {
		return nil, nil, err
	}
	m := &Machine{ctx: ctx, eval: NewWithContext(ctx), breakpoints: map[int]bool{}, paused: snap.Paused, pauseReason: snap.PauseReason, context: context.Background()}
	for _, line := range snap.Breakpoints {
		m.breakpoints[line] = true
	}
	if snap.Last != nil {
		m.last, err = d.DeserializeValue(snap.Last)
		if err != nil {
			return nil, nil, err
		}
	}
	for i, fs := range snap.Frames {
		n, ok := nodes[fs.NodePath]
		if !ok {
			return nil, nil, fmt.Errorf("frames[%d].node_path %q not found", i, fs.NodePath)
		}
		f := &machineFrame{node: n, phase: fs.Phase, index: fs.Index, kwNames: append([]string(nil), fs.KwNames...), failed: fs.Failed, breakpointConsumed: fs.BreakpointConsumed}
		if err := validateRestoredFrame(f, n); err != nil {
			return nil, nil, fmt.Errorf("frames[%d]: %w", i, err)
		}
		if fs.Value != nil {
			f.value, err = d.DeserializeValue(fs.Value)
			if err != nil {
				return nil, nil, err
			}
		}
		for _, sv := range fs.Values {
			v, e := d.DeserializeValue(sv)
			if e != nil {
				return nil, nil, e
			}
			f.values = append(f.values, v)
		}
		if fs.Kwargs != nil {
			f.kwargs = map[string]Value{}
			for k, sv := range fs.Kwargs {
				v, e := d.DeserializeValue(sv)
				if e != nil {
					return nil, nil, e
				}
				f.kwargs[k] = v
			}
		}
		if fs.Service != "" {
			svc, ok := services[fs.Service]
			if !ok {
				return nil, nil, fmt.Errorf("service %q is not registered", fs.Service)
			}
			f.service = &BoundMethodValue{ServiceName: fs.Service, Service: svc, Method: fs.Method}
		}
		m.stack = append(m.stack, f)
	}
	if snap.Failure != "" {
		m.err = fmt.Errorf("%s", snap.Failure)
	}
	cp := &MachineCheckpoint{machine: m, failure: m.err, paused: m.paused}
	return m, cp, nil
}

func validateRestoredFrame(frame *machineFrame, node ast.Node) error {
	maxPhase := 4
	maxIndex := 0
	switch n := node.(type) {
	case *ast.Program:
		maxPhase, maxIndex = 1, len(n.Statements)
	case *ast.Block:
		maxPhase, maxIndex = 1, len(n.Statements)
	case *ast.ExpressionStatement, *ast.AssignStatement, *ast.MemberExpression, *ast.PrefixExpression:
		maxPhase = 1
	case *ast.InfixExpression, *ast.IndexExpression, *ast.TernaryExpression:
		maxPhase = 2
	case *ast.CallExpression:
		maxPhase, maxIndex = 4, len(n.Arguments)
		if len(frame.values) != 0 && len(frame.values) != len(n.Arguments) {
			return fmt.Errorf("call value count does not match arguments")
		}
	case *ast.ListLiteral:
		maxPhase, maxIndex = 1, len(n.Elements)
	case *ast.SetLiteral:
		maxPhase, maxIndex = 1, len(n.Elements)
	case *ast.EmitStatement:
		maxPhase, maxIndex = 2, len(n.Values)+len(n.Named)
	}
	if frame.phase > maxPhase {
		return fmt.Errorf("phase %d exceeds %d for %T", frame.phase, maxPhase, node)
	}
	if maxIndex > 0 && frame.index > maxIndex {
		return fmt.Errorf("index %d exceeds %d for %T", frame.index, maxIndex, node)
	}
	return nil
}

func machineNodePaths(program *ast.Program) (map[ast.Node]string, map[string]ast.Node) {
	a := map[ast.Node]string{}
	b := map[string]ast.Node{}
	var walk func(ast.Node, string)
	walk = func(n ast.Node, p string) {
		if n == nil {
			return
		}
		a[n] = p
		b[p] = n
		switch x := n.(type) {
		case *ast.Program:
			for i, v := range x.Statements {
				walk(v, fmt.Sprintf("%s/s%d", p, i))
			}
		case *ast.Block:
			for i, v := range x.Statements {
				walk(v, fmt.Sprintf("%s/s%d", p, i))
			}
		case *ast.ExpressionStatement:
			walk(x.Expression, p+"/expr")
		case *ast.AssignStatement:
			for i, v := range x.Targets {
				walk(v, fmt.Sprintf("%s/t%d", p, i))
			}
			walk(x.Value, p+"/value")
		case *ast.PrefixExpression:
			walk(x.Right, p+"/right")
		case *ast.InfixExpression:
			walk(x.Left, p+"/left")
			walk(x.Right, p+"/right")
		case *ast.MemberExpression:
			walk(x.Object, p+"/object")
			walk(x.Property, p+"/property")
		case *ast.CallExpression:
			walk(x.Function, p+"/function")
			for i, v := range x.Arguments {
				walk(v, fmt.Sprintf("%s/a%d", p, i))
			}
			ks := make([]string, 0, len(x.Kwargs))
			for k := range x.Kwargs {
				ks = append(ks, k)
			}
			sort.Strings(ks)
			for _, k := range ks {
				walk(x.Kwargs[k], p+"/kw:"+k)
			}
		case *ast.ListLiteral:
			for i, v := range x.Elements {
				walk(v, fmt.Sprintf("%s/e%d", p, i))
			}
		case *ast.SetLiteral:
			for i, v := range x.Elements {
				walk(v, fmt.Sprintf("%s/e%d", p, i))
			}
		case *ast.MapLiteral:
			for i, key := range x.Order {
				walk(key, fmt.Sprintf("%s/k%d", p, i))
				walk(x.Pairs[key], fmt.Sprintf("%s/v%d", p, i))
			}
		case *ast.IndexExpression:
			walk(x.Left, p+"/left")
			walk(x.Index, p+"/index")
		case *ast.TernaryExpression:
			walk(x.Condition, p+"/condition")
			walk(x.Consequence, p+"/then")
			walk(x.Alternative, p+"/else")
		case *ast.RangeExpression:
			walk(x.Start, p+"/start")
			walk(x.End, p+"/end")
			walk(x.Step, p+"/step")
		case *ast.IfStatement:
			walk(x.Condition, p+"/condition")
			walk(x.Consequence, p+"/then")
			walk(x.Alternative, p+"/else")
		case *ast.EmitStatement:
			for i, v := range x.Values {
				walk(v, fmt.Sprintf("%s/v%d", p, i))
			}
		case *ast.ReturnStatement:
			walk(x.Value, p+"/value")
		case *ast.PauseStatement:
			walk(x.Message, p+"/message")
		}
	}
	walk(program, "root")
	return a, b
}

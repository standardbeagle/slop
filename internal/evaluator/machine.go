package evaluator

import (
	"context"
	"fmt"
	"sort"

	"github.com/standardbeagle/slop/internal/ast"
)

// MachineStatus is the terminal state of an explicit-stack evaluation.
type MachineStatus string

const (
	MachineCompleted MachineStatus = "completed"
	MachineFailed    MachineStatus = "failed"
	MachinePaused    MachineStatus = "paused"
)

type MachineResumeMode string

const (
	MachineAcceptResult MachineResumeMode = "accept_result"
	MachineRetryCall    MachineResumeMode = "retry_call"
	MachineContinue     MachineResumeMode = "continue"
)

type MachineResumeOptions struct {
	Mode        MachineResumeMode
	Replacement Value
}

type MachineResult struct {
	Status     MachineStatus
	Value      Value
	Err        error
	Checkpoint *MachineCheckpoint
}

// MachineCheckpoint captures the live frame stack before error unwinding.
// Serialization is deliberately a follow-up: frames contain AST references until
// the v2 program codec assigns stable node IDs.
type MachineCheckpoint struct {
	machine *Machine
	failure error
	paused  bool
}

// MachineCallState is an inspectable copy of a failed service invocation.
type MachineCallState struct {
	Service string
	Method  string
	Args    []Value
	Kwargs  map[string]Value
}

func (c *MachineCheckpoint) Failure() error {
	if c == nil {
		return nil
	}
	return c.failure
}

func (c *MachineCheckpoint) IsPause() bool { return c != nil && c.paused }
func (c *MachineCheckpoint) PauseReason() string {
	if c == nil || c.machine == nil {
		return ""
	}
	return c.machine.pauseReason
}

func (c *MachineCheckpoint) FailedCall() (MachineCallState, bool) {
	if c == nil || c.machine == nil || len(c.machine.stack) == 0 {
		return MachineCallState{}, false
	}
	f := c.machine.stack[len(c.machine.stack)-1]
	if !f.failed || f.service == nil {
		return MachineCallState{}, false
	}
	state := MachineCallState{Service: f.service.ServiceName, Method: f.service.Method, Args: append([]Value(nil), f.values...), Kwargs: make(map[string]Value, len(f.kwargs))}
	for key, value := range f.kwargs {
		state.Kwargs[key] = value
	}
	return state, true
}

// EditFailedCall replaces the editable invocation fields before an explicit retry.
func (c *MachineCheckpoint) EditFailedCall(service, method string, args []Value, kwargs map[string]Value) error {
	if c == nil || c.machine == nil || len(c.machine.stack) == 0 {
		return fmt.Errorf("invalid checkpoint")
	}
	f := c.machine.stack[len(c.machine.stack)-1]
	if !f.failed || f.service == nil {
		return fmt.Errorf("checkpoint has no failed service call")
	}
	svc, ok := c.machine.ctx.Services[service]
	if !ok {
		return fmt.Errorf("service %q is not registered", service)
	}
	f.service = &BoundMethodValue{ServiceName: service, Service: svc, Method: method}
	f.values = append([]Value(nil), args...)
	f.kwargs = make(map[string]Value, len(kwargs))
	for key, value := range kwargs {
		f.kwargs[key] = value
	}
	return nil
}

type machineFrame struct {
	node               ast.Node
	phase              int
	index              int
	value              Value
	values             []Value
	kwargs             map[string]Value
	kwNames            []string
	service            *BoundMethodValue
	failed             bool
	breakpointConsumed bool
}

// Machine is an iterative evaluator. Completed child values are delivered to the
// parent frame, so a stopped stack never depends on the Go call stack.
type Machine struct {
	ctx                    *Context
	eval                   *Evaluator
	stack                  []*machineFrame
	last                   Value
	err                    error
	paused                 bool
	pauseReason            string
	context                context.Context
	breakpoints            map[int]bool
	conditionalBreakpoints map[int]ast.Expression
}

func NewMachine(program *ast.Program, ctx *Context) *Machine {
	if ctx == nil {
		ctx = NewContext()
	}
	return &Machine{ctx: ctx, eval: NewWithContext(ctx), stack: []*machineFrame{{node: program}}}
}

// Emitted returns a snapshot of values emitted by the machine context.
func (m *Machine) Emitted() []Value {
	if m == nil || m.ctx == nil {
		return nil
	}
	return append([]Value(nil), m.ctx.Emitted...)
}

func (m *Machine) SetBreakpoint(line int) {
	if m.breakpoints == nil {
		m.breakpoints = map[int]bool{}
	}
	m.breakpoints[line] = true
}

func (m *Machine) SetConditionalBreakpoint(line int, condition ast.Expression) {
	if m.conditionalBreakpoints == nil {
		m.conditionalBreakpoints = map[int]ast.Expression{}
	}
	m.conditionalBreakpoints[line] = condition
	m.SetBreakpoint(line)
}

func (c *MachineCheckpoint) Position() ast.Position {
	if c == nil || c.machine == nil || len(c.machine.stack) == 0 {
		return ast.Position{}
	}
	return nodePosition(c.machine.stack[len(c.machine.stack)-1].node)
}

func (m *Machine) Run() MachineResult {
	return m.RunContext(context.Background())
}

func (m *Machine) RunContext(ctx context.Context) MachineResult {
	m.context = ctx
	for len(m.stack) > 0 && m.err == nil && !m.paused {
		if err := ctx.Err(); err != nil {
			m.err = err
			break
		}
		m.step()
	}
	if m.paused {
		return MachineResult{Status: MachinePaused, Checkpoint: &MachineCheckpoint{machine: m, paused: true}}
	}
	if m.err != nil {
		return MachineResult{Status: MachineFailed, Err: m.err, Checkpoint: &MachineCheckpoint{machine: m, failure: m.err}}
	}
	return MachineResult{Status: MachineCompleted, Value: m.last}
}

func (m *Machine) Resume(cp *MachineCheckpoint, opts MachineResumeOptions) MachineResult {
	return m.ResumeContext(context.Background(), cp, opts)
}

func (m *Machine) ResumeContext(ctx context.Context, cp *MachineCheckpoint, opts MachineResumeOptions) MachineResult {
	if cp == nil || cp.machine != m || len(m.stack) == 0 {
		return MachineResult{Status: MachineFailed, Err: fmt.Errorf("invalid checkpoint")}
	}
	top := m.stack[len(m.stack)-1]
	if cp.paused && opts.Mode == MachineContinue {
		m.paused = false
		return m.RunContext(ctx)
	}
	if !cp.paused && top.failed && top.service == nil && opts.Mode == MachineContinue {
		m.err = nil
		top.failed = false
		return m.RunContext(ctx)
	}
	if !top.failed || top.service == nil {
		return MachineResult{Status: MachineFailed, Err: fmt.Errorf("resume mode %q requires a failed service call", opts.Mode)}
	}
	switch opts.Mode {
	case MachineAcceptResult:
		if opts.Replacement == nil {
			return MachineResult{Status: MachineFailed, Err: fmt.Errorf("replacement result is required")}
		}
		m.err = nil
		top.failed = false
		m.finish(opts.Replacement)
	case MachineRetryCall:
		m.err = nil
		top.failed = false
		top.phase = 4
	default:
		return MachineResult{Status: MachineFailed, Err: fmt.Errorf("invalid resume mode %q", opts.Mode)}
	}
	return m.RunContext(ctx)
}

func (m *Machine) push(n ast.Node) { m.stack = append(m.stack, &machineFrame{node: n}) }
func (m *Machine) finish(v Value)  { m.stack = m.stack[:len(m.stack)-1]; m.last = v }
func (m *Machine) fail(err error) {
	m.err = err
	if len(m.stack) > 0 {
		m.stack[len(m.stack)-1].failed = true
	}
}

func (m *Machine) step() {
	f := m.stack[len(m.stack)-1]
	if !f.breakpointConsumed {
		f.breakpointConsumed = true
		if line := nodeLine(f.node); line > 0 && m.breakpoints[line] {
			shouldPause := true
			if condition := m.conditionalBreakpoints[line]; condition != nil {
				value, err := m.eval.Eval(condition)
				if err != nil {
					m.fail(fmt.Errorf("breakpoint condition at line %d: %w", line, err))
					return
				}
				shouldPause = value.IsTruthy()
			}
			if shouldPause {
				m.paused = true
				m.pauseReason = "breakpoint"
				return
			}
		}
	}
	switch n := f.node.(type) {
	case *ast.Program:
		m.stepSequence(f, n.Statements)
	case *ast.Block:
		m.stepSequence(f, n.Statements)
	case *ast.ExpressionStatement:
		if f.phase == 0 {
			f.phase = 1
			m.push(n.Expression)
		} else {
			m.finish(m.last)
		}
	case *ast.AssignStatement:
		if len(n.Targets) != 1 || n.Operator != "=" {
			m.fail(fmt.Errorf("resumable assignment currently requires one identifier target"))
			return
		}
		id, ok := n.Targets[0].(*ast.Identifier)
		if !ok {
			m.fail(fmt.Errorf("unsupported assignment target %T", n.Targets[0]))
			return
		}
		if f.phase == 0 {
			f.phase = 1
			m.push(n.Value)
		} else {
			m.ctx.Scope.Update(id.Value, m.last)
			m.finish(m.last)
		}
	case *ast.IntegerLiteral:
		m.finish(&IntValue{Value: n.Value})
	case *ast.FloatLiteral:
		m.finish(&FloatValue{Value: n.Value})
	case *ast.StringLiteral:
		m.finish(&StringValue{Value: n.Value})
	case *ast.BooleanLiteral:
		m.finish(NewBool(n.Value))
	case *ast.NoneLiteral:
		m.finish(NONE)
	case *ast.Identifier:
		v, ok := m.ctx.Scope.Get(n.Value)
		if !ok {
			v, ok = m.ctx.Globals.Get(n.Value)
		}
		if !ok {
			m.fail(fmt.Errorf("undefined variable: %s", n.Value))
			return
		}
		m.finish(v)
	case *ast.InfixExpression:
		m.stepInfix(f, n)
	case *ast.PrefixExpression:
		if f.phase == 0 {
			f.phase = 1
			m.push(n.Right)
		} else {
			v, err := m.eval.evalPrefixValue(n.Operator, m.last)
			if err != nil {
				m.fail(err)
			} else {
				m.finish(v)
			}
		}
	case *ast.ListLiteral:
		m.stepList(f, n)
	case *ast.MapLiteral:
		m.stepMap(f, n)
	case *ast.SetLiteral:
		m.stepSet(f, n)
	case *ast.IndexExpression:
		m.stepIndex(f, n)
	case *ast.TernaryExpression:
		m.stepTernary(f, n)
	case *ast.RangeExpression:
		m.stepRange(f, n)
	case *ast.IfStatement:
		m.stepIf(f, n)
	case *ast.EmitStatement:
		m.stepEmit(f, n)
	case *ast.ReturnStatement:
		if n.Value == nil {
			m.ctx.SetReturn(NONE)
			m.finish(NONE)
		} else if f.phase == 0 {
			f.phase = 1
			m.push(n.Value)
		} else {
			m.ctx.SetReturn(m.last)
			m.finish(m.last)
		}
	case *ast.PauseStatement:
		if f.phase == 0 {
			f.phase = 1
			m.paused = true
			m.pauseReason = "pause"
		} else {
			m.finish(NONE)
		}
	case *ast.MemberExpression:
		m.stepMember(f, n)
	case *ast.CallExpression:
		m.stepCall(f, n)
	default:
		m.fail(fmt.Errorf("unsupported resumable node %T", n))
	}
}

func nodeLine(n ast.Node) int {
	return nodePosition(n).Line
}

func nodePosition(n ast.Node) ast.Position {
	switch x := n.(type) {
	case *ast.ExpressionStatement:
		return ast.Position{Line: x.Token.Line, Column: x.Token.Column}
	case *ast.AssignStatement:
		return ast.Position{Line: x.Token.Line, Column: x.Token.Column}
	case *ast.EmitStatement:
		return ast.Position{Line: x.Token.Line, Column: x.Token.Column}
	case *ast.ReturnStatement:
		return ast.Position{Line: x.Token.Line, Column: x.Token.Column}
	case *ast.PauseStatement:
		return ast.Position{Line: x.Token.Line, Column: x.Token.Column}
	}
	return ast.Position{}
}

func (e *Evaluator) evalPrefixValue(op string, right Value) (Value, error) {
	switch op {
	case "-":
		if v, ok := right.(*IntValue); ok {
			return &IntValue{Value: -v.Value}, nil
		}
		if v, ok := right.(*FloatValue); ok {
			return &FloatValue{Value: -v.Value}, nil
		}
	case "not", "!":
		return NewBool(!right.IsTruthy()), nil
	}
	return nil, fmt.Errorf("unsupported prefix %s", op)
}

func (m *Machine) stepList(f *machineFrame, n *ast.ListLiteral) {
	if f.phase == 1 {
		f.values = append(f.values, m.last)
		f.phase = 0
	}
	if f.index < len(n.Elements) {
		c := n.Elements[f.index]
		f.index++
		f.phase = 1
		m.push(c)
		return
	}
	m.finish(&ListValue{Elements: f.values})
}

func (m *Machine) stepSet(f *machineFrame, n *ast.SetLiteral) {
	if f.phase == 1 {
		f.values = append(f.values, m.last)
		f.phase = 0
	}
	if f.index < len(n.Elements) {
		c := n.Elements[f.index]
		f.index++
		f.phase = 1
		m.push(c)
		return
	}
	s := NewSetValue()
	for _, v := range f.values {
		s.Add(v)
	}
	m.finish(s)
}

func (m *Machine) stepMap(f *machineFrame, n *ast.MapLiteral) {
	if f.kwargs == nil {
		f.kwargs = map[string]Value{}
	}
	if f.phase == 1 {
		key := n.Order[f.index-1]
		ks := ""
		if id, ok := key.(*ast.Identifier); ok {
			ks = id.Value
		} else {
			ks = key.String()
		}
		f.kwargs[ks] = m.last
		f.phase = 0
	}
	if f.index < len(n.Order) {
		key := n.Order[f.index]
		f.index++
		f.phase = 1
		m.push(n.Pairs[key])
		return
	}
	out := NewMapValue()
	for _, key := range n.Order {
		ks := ""
		if id, ok := key.(*ast.Identifier); ok {
			ks = id.Value
		} else {
			ks = key.String()
		}
		out.Set(ks, f.kwargs[ks])
	}
	m.finish(out)
}
func (m *Machine) stepIndex(f *machineFrame, n *ast.IndexExpression) {
	if f.phase == 0 {
		f.phase = 1
		m.push(n.Left)
		return
	}
	if f.phase == 1 {
		f.value = m.last
		f.phase = 2
		m.push(n.Index)
		return
	}
	l, ok := f.value.(*ListValue)
	if !ok {
		m.fail(fmt.Errorf("cannot index %s", f.value.Type()))
		return
	}
	i, ok := ToInt(m.last)
	if !ok || i < 0 || i >= int64(len(l.Elements)) {
		m.fail(fmt.Errorf("list index out of range"))
		return
	}
	m.finish(l.Elements[i])
}
func (m *Machine) stepTernary(f *machineFrame, n *ast.TernaryExpression) {
	if f.phase == 0 {
		f.phase = 1
		m.push(n.Condition)
		return
	}
	if f.phase == 1 {
		f.phase = 2
		if m.last.IsTruthy() {
			m.push(n.Consequence)
		} else {
			m.push(n.Alternative)
		}
		return
	}
	m.finish(m.last)
}

func (m *Machine) stepRange(f *machineFrame, n *ast.RangeExpression) {
	parts := []ast.Expression{n.Start, n.End, n.Step}
	if f.phase == 1 {
		f.values = append(f.values, m.last)
		f.phase = 0
	}
	for f.index < len(parts) && parts[f.index] == nil {
		f.values = append(f.values, nil)
		f.index++
	}
	if f.index < len(parts) {
		c := parts[f.index]
		f.index++
		f.phase = 1
		m.push(c)
		return
	}
	start, end, step := int64(0), int64(0), int64(1)
	if f.values[0] != nil {
		start, _ = ToInt(f.values[0])
	}
	end, _ = ToInt(f.values[1])
	if len(f.values) > 2 && f.values[2] != nil {
		step, _ = ToInt(f.values[2])
	}
	if step == 0 {
		m.fail(fmt.Errorf("range step cannot be zero"))
		return
	}
	m.finish(&IteratorValue{Type_: "range", Current: int(start), End: int(end), Step: int(step)})
}
func (m *Machine) stepIf(f *machineFrame, n *ast.IfStatement) {
	if f.phase == 0 {
		f.phase = 1
		m.push(n.Condition)
		return
	}
	if f.phase == 1 {
		f.phase = 2
		if m.last.IsTruthy() {
			m.push(n.Consequence)
		} else if n.Alternative != nil {
			m.push(n.Alternative)
		} else {
			m.finish(NONE)
		}
		return
	}
	m.finish(m.last)
}
func (m *Machine) stepEmit(f *machineFrame, n *ast.EmitStatement) {
	if f.phase == 1 {
		m.ctx.Emit(m.last)
		f.phase = 0
	}
	if f.index < len(n.Values) {
		c := n.Values[f.index]
		f.index++
		f.phase = 1
		m.push(c)
		return
	}
	if len(n.Named) > 0 {
		if f.kwNames == nil {
			for k := range n.Named {
				f.kwNames = append(f.kwNames, k)
			}
			sort.Strings(f.kwNames)
			f.kwargs = map[string]Value{}
		}
		if f.phase == 2 {
			f.kwargs[f.kwNames[f.index-1]] = m.last
			f.phase = 0
		}
		if f.index < len(f.kwNames) {
			c := n.Named[f.kwNames[f.index]]
			f.index++
			f.phase = 2
			m.push(c)
			return
		}
		out := NewMapValue()
		for _, k := range f.kwNames {
			out.Set(k, f.kwargs[k])
		}
		m.ctx.Emit(out)
	}
	m.finish(NONE)
}

func (m *Machine) stepSequence(f *machineFrame, stmts []ast.Statement) {
	if f.phase == 1 {
		f.value = m.last
		f.index++
		f.phase = 0
		if m.ctx.ShouldReturn() {
			if _, ok := f.node.(*ast.Program); ok {
				v, _ := m.ctx.GetReturn()
				m.finish(v)
			} else {
				m.finish(f.value)
			}
			return
		}
	}
	if f.index >= len(stmts) {
		if f.value == nil {
			f.value = NONE
		}
		m.finish(f.value)
		return
	}
	f.phase = 1
	m.push(stmts[f.index])
}

func (m *Machine) stepInfix(f *machineFrame, n *ast.InfixExpression) {
	switch f.phase {
	case 0:
		f.phase = 1
		m.push(n.Left)
	case 1:
		f.value = m.last
		if n.Operator == "and" && !f.value.IsTruthy() {
			m.finish(f.value)
			return
		}
		if n.Operator == "or" && f.value.IsTruthy() {
			m.finish(f.value)
			return
		}
		f.phase = 2
		m.push(n.Right)
	case 2:
		v, err := m.eval.binaryOp(n.Operator, f.value, m.last)
		if err != nil {
			m.fail(err)
			return
		}
		m.finish(v)
	}
}

func (m *Machine) stepMember(f *machineFrame, n *ast.MemberExpression) {
	if f.phase == 0 {
		f.phase = 1
		m.push(n.Object)
		return
	}
	svc, ok := m.last.(*ServiceValue)
	if !ok {
		m.fail(fmt.Errorf("cannot access property on %s", m.last.Type()))
		return
	}
	m.finish(&BoundMethodValue{ServiceName: svc.Name, Service: svc.Service, Method: n.Property.Value})
}

func (m *Machine) stepCall(f *machineFrame, n *ast.CallExpression) {
	switch f.phase {
	case 0:
		f.phase = 1
		m.push(n.Function)
	case 1:
		if f.service == nil {
			var ok bool
			f.service, ok = m.last.(*BoundMethodValue)
			if !ok {
				m.fail(fmt.Errorf("MVP only supports service calls"))
				return
			}
		}
		f.values = make([]Value, len(n.Arguments))
		f.kwargs = map[string]Value{}
		for k := range n.Kwargs {
			f.kwNames = append(f.kwNames, k)
		}
		sort.Strings(f.kwNames)
		f.phase = 2
	case 2:
		if f.index > 0 {
			f.values[f.index-1] = m.last
		}
		if f.index < len(n.Arguments) {
			child := n.Arguments[f.index]
			f.index++
			m.push(child)
			return
		}
		f.index = 0
		f.phase = 3
	case 3:
		if f.index > 0 {
			f.kwargs[f.kwNames[f.index-1]] = m.last
		}
		if f.index < len(f.kwNames) {
			child := n.Kwargs[f.kwNames[f.index]]
			f.index++
			m.push(child)
			return
		}
		f.phase = 4
	case 4:
		result, err := m.eval.callServiceMethodContext(m.context, f.service, f.values, f.kwargs)
		if err != nil {
			f.failed = true
			m.fail(err)
			return
		}
		m.finish(result)
	}
}

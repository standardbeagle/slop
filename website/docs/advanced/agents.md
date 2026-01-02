---
sidebar_position: 1
---

# SLOP Agents

## LLM as Service, Not Controller

---

## 1. The Paradigm Shift

### Traditional Agent Frameworks (LangGraph, etc.)

```
┌─────────────────────────────────────┐
│              LLM                    │  ← LLM controls everything
│  "What should I do next?"           │
│  "Should I continue or stop?"       │
│  "Which tool should I use?"         │
└─────────────────┬───────────────────┘
                  │ (implicit decisions)
                  ▼
┌─────────────────────────────────────┐
│         Graph/State Machine         │
│  - Hidden state                     │
│  - Magic routing                    │
│  - Unbounded loops                  │
└─────────────────────────────────────┘
```

### SLOP Approach

```
┌─────────────────────────────────────┐
│             Script                  │  ← Code controls everything
│  - Explicit flow                    │
│  - Visible state                    │
│  - Bounded loops                    │
└─────────────────┬───────────────────┘
                  │ (explicit calls)
                  ▼
┌─────────────────────────────────────┐
│         LLM (as service)            │
│  - Structured input                 │
│  - Structured output                │
│  - Just another tool                │
└─────────────────────────────────────┘
```

---

## 2. Core Pattern

```python
# The fundamental pattern: LLM advises, code decides

# 1. Get structured advice from LLM
decision = llm.call(
    prompt: "Given {context}, what should we do?",
    schema: {action: enum(option1, option2, option3), reason: string}
)

# 2. Code decides what to do with the advice
match decision.action:
    option1 -> do_thing_one()
    option2 -> do_thing_two()
    option3 -> do_thing_three()
```

**Key properties:**
- LLM output is **structured** (not free text)
- Code **validates** the response (schema enforced)
- Code **controls** what happens next
- Flow is **visible** in the script

---

## 3. Agent Patterns

### 3.1 ReAct Agent

```python
# ReAct: Reason + Act in a loop

task = input.task
context = []

for step in range(10):  # BOUNDED - max 10 steps
    # REASON: Ask LLM what to do
    thought = llm.call(
        prompt: """
        Task: {task}
        Previous steps: {context}
        
        What should I do next?
        """,
        schema: {
            reasoning: string,
            action: enum(search, calculate, lookup, answer),
            action_input: string
        }
    )
    
    # Check if done
    if thought.action == "answer":
        emit(answer: thought.action_input, steps: len(context))
        stop
    
    # ACT: Execute the chosen action
    result = match thought.action:
        search    -> web.search(thought.action_input, limit: 5)
        calculate -> math.eval(thought.action_input)
        lookup    -> db.query(thought.action_input)
    
    # OBSERVE: Record result for next iteration
    context.append({
        thought: thought.reasoning,
        action: thought.action,
        input: thought.action_input,
        result: result
    })

# Max steps reached
emit(
    answer: none,
    error: "Max steps reached",
    partial_context: context
)
```

### 3.2 Plan-and-Execute

```python
# Plan first, then execute deterministically

task = input.task

# PLAN: One LLM call to create the plan
plan = llm.call(
    prompt: "Create a step-by-step plan for: {task}",
    schema: {
        steps: list({
            description: string,
            tool: enum(search, fetch, analyze, summarize),
            expected_output: string
        })
    }
)

# EXECUTE: Deterministic execution of plan
results = []
for i, step in enumerate(plan.steps) with limit(20):
    log.info("Step {i+1}: {step.description}")
    
    result = match step.tool:
        search   -> web.search(step.description, limit: 10)
        fetch    -> web.fetch(step.description)
        analyze  -> llm.call(
            prompt: "Analyze: {step.description}\nData: {results}",
            schema: {analysis: string, key_points: list(string)}
        )
        summarize -> llm.call(
            prompt: "Summarize: {results}",
            schema: {summary: string}
        )
    
    results.append({step: i, description: step.description, result: result})
    
    # Optional: Check if we need to replan
    if result.error:
        replan = llm.call(
            prompt: "Step failed: {step}. Error: {result.error}. Revise plan.",
            schema: {revised_steps: list(...), should_continue: bool}
        )
        if not replan.should_continue:
            break
        plan.steps = replan.revised_steps

# SYNTHESIZE: Final answer
answer = llm.call(
    prompt: "Synthesize final answer from: {results}",
    schema: {answer: string, confidence: float}
)

emit(answer: answer.answer, confidence: answer.confidence, steps: results)
```

### 3.3 Tool Selection Agent

```python
# Simple tool selection pattern

query = input.query

# Available tools with descriptions
tools = {
    search: "Search the web for information",
    calculate: "Perform mathematical calculations",
    lookup: "Look up data in the database",
    fetch: "Fetch a specific URL",
    none: "No tool needed - answer directly"
}

# Ask LLM which tool to use
selection = llm.call(
    prompt: """
    Query: {query}
    
    Available tools:
    {tools | format_list}
    
    Which tool should be used?
    """,
    schema: {
        tool: enum(search, calculate, lookup, fetch, none),
        tool_input: string,
        reasoning: string
    }
)

# Execute selected tool
result = match selection.tool:
    search    -> web.search(selection.tool_input)
    calculate -> math.eval(selection.tool_input)
    lookup    -> db.query(selection.tool_input)
    fetch     -> web.fetch(selection.tool_input)
    none      -> {direct: true}

# Generate final answer
if result.direct:
    answer = llm.call(
        prompt: "Answer directly: {query}",
        schema: {answer: string}
    )
else:
    answer = llm.call(
        prompt: "Answer {query} using this data: {result}",
        schema: {answer: string, sources: list(string)}
    )

emit(answer)
```

### 3.4 Multi-Agent Collaboration

```python
# Multiple specialized agents working together

task = input.task

# Agent 1: Researcher
research = agent.run("researcher",
    input: {task: "Research: {task}"},
    limits: {max_steps: 5, max_llm_calls: 10}
)

# Agent 2: Critic (reviews research)
critique = agent.run("critic",
    input: {
        research: research.output,
        criteria: ["accuracy", "completeness", "relevance"]
    },
    limits: {max_steps: 3}
)

# Iterate if needed (BOUNDED)
for revision in range(3):
    if critique.output.passed:
        break
    
    # Researcher revises based on feedback
    research = agent.run("researcher",
        input: {
            task: task,
            previous: research.output,
            feedback: critique.output.feedback
        },
        limits: {max_steps: 3}
    )
    
    # Critic reviews again
    critique = agent.run("critic",
        input: {research: research.output},
        limits: {max_steps: 2}
    )

# Agent 3: Writer (produces final output)
final = agent.run("writer",
    input: {
        task: task,
        research: research.output,
        critique: critique.output
    },
    limits: {max_steps: 3}
)

emit(final.output)
```

### 3.5 Autonomous Agent with Guardrails

```python
# Agent with explicit safety checks

task = input.task
max_cost = input.max_cost or 5.00
forbidden_actions = ["delete", "send_email", "purchase"]

context = []
total_cost = 0.0

for step in range(20):
    # Get next action
    action = llm.call(
        prompt: "Task: {task}\nContext: {context}\nWhat next?",
        schema: {
            action: string,
            tool: enum(search, analyze, create, done),
            params: object
        }
    )
    
    # GUARDRAIL: Check forbidden actions
    if any(f in action.action.lower() for f in forbidden_actions):
        log.warn("Blocked forbidden action: {action.action}")
        context.append({blocked: action.action, reason: "forbidden"})
        continue
    
    # GUARDRAIL: Check cost
    estimated_cost = estimate_cost(action)
    if total_cost + estimated_cost > max_cost:
        log.warn("Cost limit would be exceeded")
        emit(
            partial: context,
            stopped: "cost_limit",
            spent: total_cost
        )
        stop
    
    # GUARDRAIL: Require confirmation for sensitive actions
    if action.tool == "create":
        confirmation = llm.call(
            prompt: "Confirm this action is safe: {action}",
            schema: {safe: bool, concerns: list(string)}
        )
        if not confirmation.safe:
            log.warn("Action flagged as unsafe: {confirmation.concerns}")
            context.append({skipped: action, concerns: confirmation.concerns})
            continue
    
    # Execute action
    if action.tool == "done":
        emit(result: action.params.result, steps: len(context), cost: total_cost)
        stop
    
    result = execute_tool(action.tool, action.params)
    total_cost += result.cost
    context.append({action: action, result: result})

emit(error: "max_steps", partial: context)
```

---

## 4. LLM Call Patterns

### 4.1 Structured Extraction

```python
# Extract structured data from unstructured text
text = input.text

extracted = llm.call(
    prompt: "Extract information from:\n{text}",
    schema: {
        people: list({name: string, role: string}),
        organizations: list({name: string, type: string}),
        dates: list({date: string, event: string}),
        key_facts: list(string)
    }
)

emit(extracted)
```

### 4.2 Classification

```python
# Classify with confidence
item = input.item

classification = llm.call(
    prompt: "Classify this item:\n{item}",
    schema: {
        category: enum(bug, feature, question, other),
        confidence: float,
        reasoning: string,
        subcategory: string?
    }
)

if classification.confidence < 0.7:
    # Low confidence - flag for review
    emit(needs_review: true, classification: classification)
else:
    emit(classification)
```

### 4.3 Validation/Critique

```python
# Validate content
content = input.content
criteria = input.criteria

validation = llm.call(
    prompt: """
    Validate this content against criteria:
    
    Content: {content}
    
    Criteria: {criteria}
    """,
    schema: {
        valid: bool,
        issues: list({
            criterion: string,
            passed: bool,
            feedback: string
        }),
        overall_score: float,
        suggestions: list(string)
    }
)

emit(validation)
```

### 4.4 Decomposition

```python
# Break complex task into subtasks
task = input.task

decomposition = llm.call(
    prompt: "Break this task into subtasks:\n{task}",
    schema: {
        subtasks: list({
            id: int,
            description: string,
            dependencies: list(int),  # IDs of prerequisite subtasks
            estimated_complexity: enum(low, medium, high)
        }),
        execution_order: list(int)
    }
)

# Execute in order
results = {}
for task_id in decomposition.execution_order:
    subtask = decomposition.subtasks | find(t -> t.id == task_id)
    
    # Gather dependency results
    dep_results = [results[d] for d in subtask.dependencies]
    
    result = execute_subtask(subtask, dep_results)
    results[task_id] = result

emit(results)
```

---

## 5. Error Handling

### 5.1 Retry with Feedback

```python
for attempt in range(3):
    result = llm.call(
        prompt: "{task}\n{feedback if attempt > 0 else ''}",
        schema: expected_schema
    )
    
    validation = validate(result)
    if validation.ok:
        emit(result)
        stop
    
    feedback = "Previous attempt failed: {validation.errors}. Please fix."

emit(error: "Failed after 3 attempts", last_result: result)
```

### 5.2 Fallback Chain

```python
# Try multiple approaches
approaches = [
    {model: "claude-sonnet", temp: 0.0},
    {model: "claude-sonnet", temp: 0.5},
    {model: "claude-opus", temp: 0.0}
]

for approach in approaches:
    try:
        result = llm.call(
            prompt: task,
            schema: schema,
            model: approach.model,
            temperature: approach.temp
        )
        if validate(result):
            emit(result)
            stop
    catch error:
        log.warn("Approach failed: {approach}, error: {error}")
        continue

emit(error: "All approaches failed")
```

### 5.3 Graceful Degradation

```python
# Full result with fallback to partial
task = input.task

# Try full analysis
try:
    full = llm.call(
        prompt: "Full analysis of: {task}",
        schema: {
            summary: string,
            details: list({...}),
            recommendations: list({...}),
            confidence: float
        },
        max_tokens: 4000
    )
    emit(full)
    stop
catch TokenLimitError:
    log.warn("Full analysis too large, falling back to summary")

# Fallback to summary only
summary = llm.call(
    prompt: "Brief summary of: {task}",
    schema: {summary: string, key_points: list(string)},
    max_tokens: 1000
)
emit(summary: summary, degraded: true)
```

---

## 6. Comparison with LangGraph

### LangGraph Version

```python
# 40+ lines of boilerplate, hidden flow

from langgraph.graph import StateGraph, END
from typing import TypedDict, Annotated

class AgentState(TypedDict):
    messages: list
    next_action: str
    iteration: int

def analyze_node(state: AgentState):
    # ... hidden logic
    return {"messages": [...], "next_action": "search"}

def search_node(state: AgentState):
    # ... hidden logic
    return {"messages": [...]}

def should_continue(state: AgentState):
    if state["iteration"] > 10:
        return "end"
    if state["next_action"] == "done":
        return "end"
    return "continue"

# Build graph
workflow = StateGraph(AgentState)
workflow.add_node("analyze", analyze_node)
workflow.add_node("search", search_node)
workflow.add_conditional_edges(
    "analyze",
    should_continue,
    {"continue": "search", "end": END}
)
workflow.add_edge("search", "analyze")
workflow.set_entry_point("analyze")

app = workflow.compile()
result = app.invoke({"messages": [task], "iteration": 0})
# What actually happened? Good luck figuring it out.
```

### SLOP Version

```python
# 25 lines, completely visible flow

task = input.task
context = []

for step in range(10):
    analysis = llm.call(
        prompt: "Task: {task}\nContext: {context}\nWhat next?",
        schema: {action: enum(search, done), query: string}
    )
    
    if analysis.action == "done":
        emit(answer: analysis.query, steps: len(context))
        stop
    
    results = web.search(analysis.query, limit: 5)
    context.append({query: analysis.query, results: results})

emit(error: "max_steps", partial: context)
# Every step is visible. Flow is obvious. Debugging is trivial.
```

---

## 7. Debugging

### 7.1 Trace Output

```bash
$ slop run agent.slop --trace

[1] task = "Research quantum computing"
[2] FOR step IN range(10):
[3]   llm.call -> {action: "search", query: "quantum computing basics"}
[4]   web.search("quantum computing basics") -> [{...}, {...}, ...]
[5]   context.append(...)
[6] FOR step IN range(10): (iteration 2)
[7]   llm.call -> {action: "search", query: "quantum computing applications"}
[8]   web.search("quantum computing applications") -> [{...}, {...}, ...]
[9]   context.append(...)
[10] FOR step IN range(10): (iteration 3)
[11]  llm.call -> {action: "done", query: "Quantum computing is..."}
[12]  emit({answer: "...", steps: 2})
[13]  stop

Execution complete: 3 iterations, 3 LLM calls, 2 searches
```

### 7.2 Step-Through

```bash
$ slop run agent.slop --step

[1] task = "Research quantum computing"
    > next

[2] FOR step IN range(10):
    > inspect context
    context = []
    > next

[3] llm.call(...)
    > inspect prompt
    "Task: Research quantum computing\nContext: []\nWhat next?"
    > next
    -> {action: "search", query: "quantum computing basics"}
    > next

[4] web.search(...)
    > skip  # Skip to next LLM call
    
[7] llm.call(...)
    > modify analysis.action = "done"  # Force completion
    > continue
```

---

## 8. Summary

| Aspect | LangGraph | SLOP |
|--------|-----------|------|
| Who controls flow | LLM (implicit) | Code (explicit) |
| State visibility | Hidden in dict | Variables you can see |
| Loop bounds | Hope it stops | `range(10)` - guaranteed |
| Debugging | "Why did it go there?" | Read the code |
| Error handling | Framework magic | Normal try/catch |
| Cost control | Pray | Counted and limited |
| Learning curve | Framework concepts | Just Python-ish |

**SLOP agents are just scripts.** You can read them, debug them, bound them, and understand them.

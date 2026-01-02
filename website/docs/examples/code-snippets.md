---
sidebar_position: 2
---

# SLOP Examples

## Comprehensive Examples by Use Case

---

## 1. One-Shot Scripts

### 1.1 Simple Data Pipeline

```python
# Enrich and sync contacts
contacts = salesforce.query("SELECT Id, Email, Name FROM Contact LIMIT 100")

for contact in contacts with rate(10/s), parallel(3):
    # Enrich from multiple sources
    company = clearbit.lookup(contact.Email)
    linkedin = linkedin.find(contact.Name)
    
    # Update contact
    contact.Company = company.name
    contact.Industry = company.industry
    contact.LinkedIn = linkedin.profile_url
    
    salesforce.update(contact.Id, contact)

emit(processed: len(contacts))
```

### 1.2 Web Research

```python
# Research a topic and summarize
topic = input.topic

# Search multiple sources
results = []
for source in ["google", "arxiv", "hackernews"] with parallel(3):
    matches = web.search("{topic} site:{source}", limit: 10)
    results.extend(matches)

# Deduplicate and fetch content
results = results | unique(r -> r.url) | take(20)

documents = []
for result in results with rate(5/s):
    try:
        content = web.fetch(result.url)
        documents.append({url: result.url, content: content.text[:5000]})
    catch error:
        log.warn("Failed to fetch {result.url}")

# Summarize
summary = llm.call(
    prompt: "Summarize these documents about '{topic}':\n{documents}",
    schema: {
        summary: string,
        key_points: list(string),
        sources: list(string)
    }
)

emit(summary)
```

### 1.3 Data Transformation

```python
# Transform and validate data
raw_data = input.data

# Clean and normalize
cleaned = []
for record in raw_data with limit(10000):
    # Normalize fields
    record.email = record.email?.lower().strip()
    record.name = record.name?.strip()
    record.phone = normalize_phone(record.phone)
    
    # Validate
    if not record.email or "@" not in record.email:
        continue
    if not record.name:
        continue
    
    # Deduplicate by email
    cleaned.append(record)

cleaned = cleaned | unique(r -> r.email)

# Categorize using LLM
for batch in cleaned | chunk(50):
    categories = llm.call(
        prompt: "Categorize these contacts by industry:\n{batch}",
        schema: {
            categorized: list({email: string, industry: string})
        }
    )
    
    for cat in categories.categorized:
        record = cleaned | find(r -> r.email == cat.email)
        if record:
            record.industry = cat.industry

emit(
    total: len(raw_data),
    cleaned: len(cleaned),
    data: cleaned
)
```

---

## 2. World Building

### 2.1 Game World Generation

```python
# Generate a game world with regions, locations, and NPCs
seed = input.seed or 12345
world_name = input.name or "Eldoria"

# Initialize RNG
random.seed(seed)

# Create regions
regions = []
for region_type in ["forest", "mountain", "plains", "coast", "desert"]:
    region = {
        id: uuid(),
        name: gen.region_name(region_type),
        type: region_type,
        danger_level: random.range(1, 10),
        climate: gen.climate(region_type),
        locations: []
    }
    regions.append(region)

# Create locations in each region
for region in regions:
    location_count = random.range(3, 7)
    
    for i in range(location_count):
        loc_type = random.choice(["town", "dungeon", "landmark", "camp"])
        
        location = {
            id: uuid(),
            name: gen.location_name(loc_type, region.type),
            type: loc_type,
            region_id: region.id,
            danger: region.danger_level + random.range(-2, 2) | clamp(1, 10),
            npcs: [],
            items: [],
            connections: []
        }
        region.locations.append(location)

# Populate towns with NPCs
all_locations = regions | flat_map(r -> r.locations)
towns = all_locations | filter(l -> l.type == "town")

for town in towns:
    pop_size = random.range(5, 15)
    
    for i in range(pop_size):
        occupation = random.weighted({
            farmer: 0.3,
            merchant: 0.2,
            guard: 0.15,
            innkeeper: 0.1,
            blacksmith: 0.1,
            healer: 0.05,
            mayor: 0.02,
            beggar: 0.08
        })
        
        npc = {
            id: uuid(),
            name: gen.npc_name(),
            occupation: occupation,
            location_id: town.id,
            disposition: random.choice(["friendly", "neutral", "suspicious"]),
            inventory: gen.inventory(occupation),
            dialogue: gen.dialogue_hooks(occupation)
        }
        town.npcs.append(npc)

# Connect nearby locations
for region in regions:
    locs = region.locations
    for i, loc in enumerate(locs):
        # Connect to 1-3 other locations in region
        others = locs | filter(l -> l.id != loc.id) | shuffle() | take(random.range(1, 3))
        for other in others:
            if other.id not in loc.connections:
                loc.connections.append(other.id)
                other.connections.append(loc.id)

# Generate quests
quests = []
for region in regions:
    quest_count = random.range(2, 5)
    
    for i in range(quest_count):
        towns_in_region = region.locations | filter(l -> l.type == "town")
        dungeons_in_region = region.locations | filter(l -> l.type == "dungeon")
        
        if len(towns_in_region) > 0 and len(dungeons_in_region) > 0:
            giver_loc = random.choice(towns_in_region)
            target_loc = random.choice(dungeons_in_region)
            giver_npc = random.choice(giver_loc.npcs) if giver_loc.npcs else none
            
            if giver_npc:
                quest = {
                    id: uuid(),
                    name: gen.quest_name(),
                    type: random.choice(["fetch", "kill", "escort", "discover"]),
                    giver_id: giver_npc.id,
                    target_location_id: target_loc.id,
                    reward: gen.reward(region.danger_level),
                    description: gen.quest_description()
                }
                quests.append(quest)

emit({
    name: world_name,
    seed: seed,
    regions: regions,
    quests: quests,
    stats: {
        total_regions: len(regions),
        total_locations: len(all_locations),
        total_npcs: all_locations | flat_map(l -> l.npcs) | len(),
        total_quests: len(quests)
    }
})
```

### 2.2 Procedural Dungeon

```python
# Generate a dungeon with rooms, enemies, and loot
dungeon_name = input.name
depth = input.depth or 5
difficulty = input.difficulty or "normal"

floors = []

for floor_num in range(1, depth + 1):
    floor_difficulty = floor_num * (1.5 if difficulty == "hard" else 1.0)
    room_count = 5 + floor_num * 2
    
    rooms = []
    for i in range(room_count):
        room_type = random.weighted({
            combat: 0.4,
            treasure: 0.2,
            puzzle: 0.15,
            trap: 0.1,
            rest: 0.1,
            boss: 0.05 if i == room_count - 1 and floor_num == depth else 0
        })
        
        room = {
            id: uuid(),
            floor: floor_num,
            type: room_type,
            cleared: false,
            connections: [],
            contents: {}
        }
        
        # Add room contents based on type
        match room_type:
            combat -> 
                enemy_count = random.range(1, 3 + floor_num)
                room.contents.enemies = [
                    gen.enemy(floor_difficulty) for i in range(enemy_count)
                ]
            
            treasure ->
                room.contents.chest = {
                    locked: random.chance(0.3),
                    loot: gen.loot(floor_difficulty, random.range(1, 4))
                }
            
            puzzle ->
                room.contents.puzzle = gen.puzzle(floor_difficulty)
            
            trap ->
                room.contents.trap = gen.trap(floor_difficulty)
            
            boss ->
                room.contents.boss = gen.boss(floor_num, difficulty)
                room.contents.loot = gen.legendary_loot(floor_num)
            
            rest ->
                room.contents.campfire = true
                room.contents.merchant = random.chance(0.3)
        
        rooms.append(room)
    
    # Connect rooms (ensure all reachable)
    for i, room in enumerate(rooms):
        if i > 0:
            # Connect to at least one previous room
            prev = random.choice(rooms[:i])
            room.connections.append(prev.id)
            prev.connections.append(room.id)
        
        # Maybe connect to another room
        if random.chance(0.3) and i > 1:
            other = random.choice([r for r in rooms[:i] if r.id not in room.connections])
            if other:
                room.connections.append(other.id)
                other.connections.append(room.id)
    
    floors.append({
        level: floor_num,
        rooms: rooms,
        entrance: rooms[0].id if floor_num == 1 else none,
        exit: rooms[-1].id,
        stairs_down: rooms[-1].id if floor_num < depth else none
    })

emit({
    name: dungeon_name,
    depth: depth,
    difficulty: difficulty,
    floors: floors,
    stats: {
        total_rooms: floors | flat_map(f -> f.rooms) | len(),
        total_enemies: floors | flat_map(f -> f.rooms) | flat_map(r -> r.contents.enemies or []) | len(),
        boss_floor: depth
    }
})
```

---

## 3. Code Editing

### 3.1 Add Logging to Functions

```python
# Add logging to all public functions in a Go file
file_path = input.file

source = filesystem.read(file_path)
ast = parser.parse(source, language: "go")

# Find all exported functions (capitalized name)
functions = ast.find_all("function_declaration")
public_funcs = functions | filter(f -> f.name[0].isupper())

edits = []
for func in public_funcs:
    # Check if logging already exists
    has_logging = func.body.contains("log.") or func.body.contains("slog.")
    
    if not has_logging:
        # Generate logging statement
        params = func.params | map(p -> p.name) | join(", ")
        log_stmt = 'slog.Debug("entering {func.name}", "params", []any\{{params}\})'
        
        edits.append({
            type: "insert",
            location: func.body.start + 1,  # After opening brace
            content: "\n\t{log_stmt}\n"
        })

# Apply edits (reverse order to preserve positions)
for edit in edits | sort(e -> -e.location):
    source = source[:edit.location] + edit.content + source[edit.location:]

emit(
    modified: source,
    changes: len(edits),
    functions_updated: edits | map(e -> e.func_name)
)
```

### 3.2 Extract Function

```python
# Extract selected lines into a new function
file_path = input.file
start_line = input.start_line
end_line = input.end_line
new_func_name = input.function_name

source = filesystem.read(file_path)
lines = source.split("\n")

# Get the selected code
selected = lines[start_line - 1:end_line]
selected_code = "\n".join(selected)

# Analyze the selected code
analysis = llm.call(
    prompt: """
    Analyze this code to extract as a function:
    
    ```
    {selected_code}
    ```
    
    Identify:
    1. Variables used but not defined in selection (parameters)
    2. Variables defined and used after selection (return values)
    3. Appropriate function signature
    """,
    schema: {
        parameters: list({name: string, type: string}),
        returns: list({name: string, type: string}),
        signature: string,
        cleaned_body: string
    }
)

# Generate new function
params_str = analysis.parameters | map(p -> "{p.name} {p.type}") | join(", ")
returns_str = analysis.returns | map(r -> r.type) | join(", ")
if len(analysis.returns) > 1:
    returns_str = "({returns_str})"

new_function = """
func {new_func_name}({params_str}) {returns_str} {{
{analysis.cleaned_body | indent(1)}
}}
"""

# Generate call site
args = analysis.parameters | map(p -> p.name) | join(", ")
if len(analysis.returns) == 0:
    call = "{new_func_name}({args})"
elif len(analysis.returns) == 1:
    call = "{analysis.returns[0].name} := {new_func_name}({args})"
else:
    ret_names = analysis.returns | map(r -> r.name) | join(", ")
    call = "{ret_names} := {new_func_name}({args})"

# Build new source
new_lines = (
    lines[:start_line - 1] +
    ["\t" + call] +
    lines[end_line:]
)

# Find where to insert function (end of file, before closing brace if any)
new_source = "\n".join(new_lines) + "\n" + new_function

emit(
    modified: new_source,
    new_function: new_func_name,
    parameters: analysis.parameters,
    returns: analysis.returns
)
```

### 3.3 Bulk Rename

```python
# Rename a symbol across multiple files
old_name = input.old_name
new_name = input.new_name
directory = input.directory or "."

# Find all Go files
files = filesystem.glob("{directory}/**/*.go")

changes = []
for file_path in files:
    source = filesystem.read(file_path)
    
    # Simple approach: find and track occurrences
    # (Real implementation would use AST for accuracy)
    if old_name in source:
        # Count occurrences
        count = source.count(old_name)
        
        # Replace
        new_source = source.replace(old_name, new_name)
        
        changes.append({
            file: file_path,
            occurrences: count,
            original: source,
            modified: new_source
        })

# Preview changes
emit(
    preview: true,
    files_affected: len(changes),
    total_occurrences: changes | map(c -> c.occurrences) | sum(),
    changes: changes | map(c -> {file: c.file, count: c.occurrences})
)

# If confirmed, apply changes
if input.confirm:
    for change in changes:
        filesystem.write(change.file, change.modified)
    
    emit(
        applied: true,
        files_modified: len(changes)
    )
```

---

## 4. Agent Workflows

### 4.1 Research Assistant

```python
# Research a topic and produce a report
query = input.query
depth = input.depth or "standard"  # quick, standard, deep

# Phase 1: Understand the query
understanding = llm.call(
    prompt: "Break down this research query:\n{query}",
    schema: {
        main_topic: string,
        subtopics: list(string),
        search_queries: list(string),
        key_questions: list(string)
    }
)

# Phase 2: Gather information
search_limit = match depth:
    "quick" -> 5
    "standard" -> 15
    "deep" -> 30

all_results = []
for search_query in understanding.search_queries with limit(10), parallel(3):
    results = web.search(search_query, limit: search_limit // len(understanding.search_queries))
    all_results.extend(results)

# Deduplicate
all_results = all_results | unique(r -> r.url) | take(search_limit)

# Phase 3: Fetch and extract content
documents = []
for result in all_results with rate(5/s):
    try:
        page = web.fetch(result.url)
        
        # Extract key info using LLM
        extraction = llm.call(
            prompt: """
            Extract information relevant to "{query}" from:
            
            URL: {result.url}
            Content: {page.text[:10000]}
            """,
            schema: {
                relevant: bool,
                summary: string,
                key_facts: list(string),
                quotes: list(string)
            }
        )
        
        if extraction.relevant:
            documents.append({
                url: result.url,
                title: result.title,
                summary: extraction.summary,
                facts: extraction.key_facts,
                quotes: extraction.quotes
            })
    catch error:
        log.warn("Failed to process {result.url}: {error}")

# Phase 4: Synthesize report
report = llm.call(
    prompt: """
    Write a research report on: {query}
    
    Key questions to answer:
    {understanding.key_questions}
    
    Sources and findings:
    {documents}
    """,
    schema: {
        title: string,
        executive_summary: string,
        sections: list({
            heading: string,
            content: string,
            sources: list(string)
        }),
        conclusions: list(string),
        further_research: list(string)
    }
)

emit({
    report: report,
    sources: documents | map(d -> {url: d.url, title: d.title}),
    metadata: {
        query: query,
        depth: depth,
        sources_searched: len(all_results),
        sources_used: len(documents)
    }
})
```

### 4.2 Code Review Agent

```python
# Review a pull request
repo = input.repo
pr_number = input.pr_number

# Fetch PR details
pr = github.get_pull_request(repo, pr_number)
files = github.get_pr_files(repo, pr_number)

reviews = []

for file in files with limit(50):
    if not file.filename.endswith((".go", ".py", ".ts", ".js")):
        continue
    
    # Get the diff
    diff = file.patch
    
    # Review with LLM
    review = llm.call(
        prompt: """
        Review this code change:
        
        File: {file.filename}
        Status: {file.status}
        
        Diff:
        ```
        {diff}
        ```
        
        Check for:
        1. Bugs or logic errors
        2. Security issues
        3. Performance concerns
        4. Code style / best practices
        5. Missing error handling
        6. Missing tests
        """,
        schema: {
            summary: string,
            issues: list({
                severity: enum(critical, warning, suggestion),
                line: int?,
                description: string,
                suggestion: string?
            }),
            approved: bool
        }
    )
    
    reviews.append({
        file: file.filename,
        review: review
    })

# Aggregate results
all_issues = reviews | flat_map(r -> r.review.issues)
critical_count = all_issues | filter(i -> i.severity == "critical") | len()
warning_count = all_issues | filter(i -> i.severity == "warning") | len()

# Generate overall verdict
verdict = llm.call(
    prompt: """
    Based on these file reviews, provide an overall PR verdict:
    
    Reviews: {reviews}
    
    Critical issues: {critical_count}
    Warnings: {warning_count}
    """,
    schema: {
        approved: bool,
        summary: string,
        blocking_issues: list(string),
        suggestions: list(string)
    }
)

# Post review to GitHub
if input.post_review:
    github.create_review(
        repo: repo,
        pr: pr_number,
        event: "COMMENT" if verdict.approved else "REQUEST_CHANGES",
        body: verdict.summary,
        comments: all_issues | filter(i -> i.line) | map(i -> {
            path: i.file,
            line: i.line,
            body: "[{i.severity}] {i.description}\n\n{i.suggestion or ''}"
        })
    )

emit({
    verdict: verdict,
    files_reviewed: len(reviews),
    issues: {
        critical: critical_count,
        warnings: warning_count,
        suggestions: all_issues | filter(i -> i.severity == "suggestion") | len()
    },
    reviews: reviews
})
```

### 4.3 Data Entry Agent

```python
# Process unstructured documents into structured data
documents = input.documents  # List of {filename, content}
schema = input.schema  # Expected output structure

results = []
errors = []

for doc in documents with limit(100), rate(5/s):
    # Extract structured data
    extraction = llm.call(
        prompt: """
        Extract data from this document according to the schema.
        
        Document ({doc.filename}):
        {doc.content}
        
        Schema:
        {schema}
        
        If a field cannot be determined, use null.
        If confident about extraction, set confidence to high.
        """,
        schema: {
            data: schema,
            confidence: enum(high, medium, low),
            uncertain_fields: list(string),
            notes: string?
        }
    )
    
    # Validate extraction
    if extraction.confidence == "low":
        errors.append({
            document: doc.filename,
            reason: "low_confidence",
            uncertain: extraction.uncertain_fields
        })
        continue
    
    # Validate required fields
    missing = validate_required(extraction.data, schema)
    if missing:
        errors.append({
            document: doc.filename,
            reason: "missing_required",
            fields: missing
        })
        continue
    
    results.append({
        source: doc.filename,
        data: extraction.data,
        confidence: extraction.confidence,
        notes: extraction.notes
    })

# Summary
emit({
    processed: len(results),
    errors: len(errors),
    success_rate: len(results) / len(documents) * 100,
    results: results,
    error_details: errors
})
```

---

## 5. MCP Orchestration

### 5.1 Cross-Service Sync

```python
# Sync data between Salesforce, HubSpot, and Airtable
sync_type = input.sync_type or "full"  # full, incremental

# Get last sync time for incremental
last_sync = store.get("last_sync_time") if sync_type == "incremental" else none

# Fetch from all sources
salesforce_contacts = salesforce.query(
    "SELECT Id, Email, Name, Company, UpdatedAt FROM Contact" +
    (" WHERE UpdatedAt > {last_sync}" if last_sync else "") +
    " LIMIT 1000"
)

hubspot_contacts = hubspot.get_contacts(
    updated_after: last_sync,
    limit: 1000
)

airtable_contacts = airtable.list_records(
    base: "contacts",
    filter: {updated_after: last_sync} if last_sync else {}
)

# Normalize to common format
def normalize_salesforce(c):
    return {
        email: c.Email.lower(),
        name: c.Name,
        company: c.Company,
        source: "salesforce",
        source_id: c.Id
    }

def normalize_hubspot(c):
    return {
        email: c.properties.email?.lower(),
        name: "{c.properties.firstname} {c.properties.lastname}".strip(),
        company: c.properties.company,
        source: "hubspot",
        source_id: c.id
    }

def normalize_airtable(c):
    return {
        email: c.fields.Email?.lower(),
        name: c.fields.Name,
        company: c.fields.Company,
        source: "airtable",
        source_id: c.id
    }

all_contacts = (
    salesforce_contacts | map(normalize_salesforce) |
    concat(hubspot_contacts | map(normalize_hubspot)) |
    concat(airtable_contacts | map(normalize_airtable))
)

# Group by email to find records that need syncing
by_email = all_contacts | group(c -> c.email)

sync_actions = []
for email, records in by_email.items():
    if len(records) == 1:
        continue  # Only in one system, skip
    
    # Find most complete/recent record as source of truth
    master = records | sort(r -> completeness_score(r)) | last()
    
    for record in records:
        if record.source_id == master.source_id:
            continue
        
        # Check if update needed
        if needs_update(record, master):
            sync_actions.append({
                action: "update",
                target: record.source,
                target_id: record.source_id,
                data: master
            })

# Execute sync actions
results = {updated: 0, failed: 0, errors: []}

for action in sync_actions with rate(10/s):
    try:
        match action.target:
            "salesforce" -> salesforce.update(action.target_id, action.data)
            "hubspot" -> hubspot.update_contact(action.target_id, action.data)
            "airtable" -> airtable.update_record("contacts", action.target_id, action.data)
        results.updated += 1
    catch error:
        results.failed += 1
        results.errors.append({id: action.target_id, error: str(error)})

# Save sync time
store.set("last_sync_time", now())

emit({
    sync_type: sync_type,
    sources_checked: {
        salesforce: len(salesforce_contacts),
        hubspot: len(hubspot_contacts),
        airtable: len(airtable_contacts)
    },
    unique_contacts: len(by_email),
    actions_taken: len(sync_actions),
    results: results
})
```

### 5.2 Batch Processing Pipeline

```python
# Process large batch with checkpointing
items = input.items
batch_size = input.batch_size or 100
checkpoint_every = input.checkpoint_every or 10

# Resume from checkpoint if exists
checkpoint = store.get("batch_checkpoint")
start_index = checkpoint.index if checkpoint else 0
results = checkpoint.results if checkpoint else []

batches = items[start_index:] | chunk(batch_size)

for batch_num, batch in enumerate(batches) with limit(1000):
    batch_results = []
    
    for item in batch with parallel(5), rate(20/s):
        try:
            # Process item
            processed = process_item(item)
            
            # Enrich with LLM if needed
            if processed.needs_classification:
                classification = llm.call(
                    prompt: "Classify: {processed}",
                    schema: {category: string, confidence: float}
                )
                processed.category = classification.category
            
            batch_results.append({status: "success", item: processed})
        catch error:
            batch_results.append({status: "error", item: item.id, error: str(error)})
    
    results.extend(batch_results)
    
    # Checkpoint
    if batch_num % checkpoint_every == 0:
        store.set("batch_checkpoint", {
            index: start_index + (batch_num + 1) * batch_size,
            results: results
        })
        log.info("Checkpoint saved at batch {batch_num}")

# Clear checkpoint on completion
store.delete("batch_checkpoint")

# Summary
success_count = results | filter(r -> r.status == "success") | len()
error_count = results | filter(r -> r.status == "error") | len()

emit({
    total_processed: len(results),
    success: success_count,
    errors: error_count,
    error_details: results | filter(r -> r.status == "error") | take(100)
})
```

---

## 6. Testing

### 6.1 Self-Testing Script

```python
# Script that tests itself
test_results = []

def assert_eq(actual, expected, name):
    passed = actual == expected
    test_results.append({
        name: name,
        passed: passed,
        actual: actual,
        expected: expected
    })
    return passed

def assert_true(condition, name):
    test_results.append({
        name: name,
        passed: condition
    })
    return condition

# Test cases
assert_eq(1 + 1, 2, "basic_addition")
assert_eq("hello".upper(), "HELLO", "string_upper")
assert_eq([1, 2, 3] | filter(x -> x > 1) | list(), [2, 3], "filter_list")
assert_eq([1, 2, 3] | map(x -> x * 2) | list(), [2, 4, 6], "map_list")
assert_true(len([1, 2, 3]) == 3, "len_function")

# Test LLM call (mocked in test mode)
if env.mode == "test":
    mock.llm.call = (args) -> {answer: "mocked"}

result = llm.call(
    prompt: "test",
    schema: {answer: string}
)
assert_eq(result.answer, "mocked" if env.mode == "test" else result.answer, "llm_call")

# Report
passed = test_results | filter(t -> t.passed) | len()
failed = test_results | filter(t -> not t.passed) | len()

emit({
    total: len(test_results),
    passed: passed,
    failed: failed,
    success: failed == 0,
    failures: test_results | filter(t -> not t.passed)
})
```

---

## 7. Quick Recipes

### 7.1 CSV to JSON

```python
csv_content = input.csv
output = csv.parse(csv_content)
emit(output)
```

### 7.2 URL Shortener Check

```python
urls = input.urls
results = []
for url in urls with parallel(10), rate(50/s):
    try:
        response = http.head(url, follow_redirects: true)
        results.append({original: url, final: response.url, status: response.status})
    catch:
        results.append({original: url, error: true})
emit(results)
```

### 7.3 Markdown to HTML

```python
markdown = input.markdown
html = markdown.render(markdown)
emit(html)
```

### 7.4 JSON Schema Validator

```python
data = input.data
schema = input.schema
result = json_schema.validate(data, schema)
emit({valid: result.valid, errors: result.errors})
```

### 7.5 Hash File

```python
content = input.content
emit({
    md5: hash.md5(content),
    sha256: hash.sha256(content),
    size: len(content)
})
```

# Discovery Patterns

Parallel search agent definitions for design decision tracing.

## Design Decision Tracing (Concepts) — Parallel Agents

Launch all 4 agents in parallel using the Task tool, then wait for all to complete.

### Agent 1: CASS Session Search

```
Tool: Task
Parameters:
  subagent_type: "Explore"
  model: "haiku"
  description: "CASS search: <concept>"
  prompt: |
    Search session transcripts for: <concept>

    Run this command:
    cass search "<concept>" --json --limit 10

    Parse the JSON output and extract:
    - Session dates (created_at field, convert from Unix ms)
    - Session paths (source_path field)
    - Agents used (agent field)
    - Relevance scores (score field)
    - Key snippets (snippet/content fields)

    Return a structured list sorted by date (oldest first).
```

### Agent 2: Handoff Search

```
Tool: Task
Parameters:
  subagent_type: "Explore"
  model: "haiku"
  description: "Handoff search: <concept>"
  prompt: |
    Search handoff documents for: <concept>

    1. List handoff files:
       ls -la .agents/handoff/*.md 2>/dev/null

    2. Search for concept mentions:
       grep -l "<concept>" .agents/handoff/*.md 2>/dev/null

    3. For each matching file, extract:
       - File date (from filename YYYY-MM-DD)
       - Context around the mention (grep -B5 -A5)
       - Related decisions or questions

    Return a structured list sorted by date.
```

### Agent 3: Git History Search

```
Tool: Task
Parameters:
  subagent_type: "Explore"
  model: "haiku"
  description: "Git search: <concept>"
  prompt: |
    Search git history for: <concept>

    1. Search commit messages:
       git log --oneline --grep="<concept>" | head -20

    2. For interesting commits, get details:
       git show --stat <commit-sha>

    3. Extract:
       - Commit dates
       - Commit messages
       - Files changed
       - Authors

    Return a structured list sorted by date.
```

### Agent 4: Research/Learnings Search

```
Tool: Task
Parameters:
  subagent_type: "Explore"
  model: "haiku"
  description: "Research search: <concept>"
  prompt: |
    Search research and learning artifacts for: <concept>

    1. Search research docs:
       grep -l "<concept>" .agents/research/*.md 2>/dev/null

    2. Search learnings:
       grep -l "<concept>" .agents/learnings/*.md 2>/dev/null

    3. Search patterns:
       grep -l "<concept>" .agents/patterns/*.md 2>/dev/null

    4. For each match, extract:
       - File date (from filename or modification time)
       - Context around the mention
       - Related concepts

    Return a structured list sorted by date.
```

## Git-Based Tracing (Commits/Refs)

For git refs, trace the commit history:

```bash
# Get commit details
git show --stat <ref>

# Get commit ancestry
git log --oneline --ancestry-path <ref>..HEAD | head -20

# Find related commits
git log --oneline --all --grep="$(git log -1 --format=%s <ref> | head -c 50)" | head -10
```

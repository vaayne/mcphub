---
name: find-skills
description: Helps users discover and install agent skills when they ask questions like "how do I do X", "find a skill for X", "is there a skill that can...", or express interest in extending capabilities. This skill should be used when the user is looking for functionality that might exist as an installable skill.
---

# Find Skills

This skill helps you discover and install skills from the open agent skills ecosystem.

## When to Use This Skill

Use this skill when the user:

- Asks "how do I do X" where X might be a common task with an existing skill
- Says "find a skill for X" or "is there a skill for X"
- Asks "can you do X" where X is a specialized capability
- Expresses interest in extending agent capabilities
- Wants to search for tools, templates, or workflows
- Mentions they wish they had help with a specific domain (design, testing, deployment, etc.)

## Requirements

- `mh` CLI must be installed. If not available, install with:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/vaayne/mcphub/main/scripts/install.sh | sh
  ```

## What is the Skills CLI?

The MCP Hub CLI (`mh skills`) is the package manager for the open agent skills ecosystem. Skills are modular packages that extend agent capabilities with specialized knowledge, workflows, and tools.

**Key commands:**

- `mh skills find [query]` - Search for skills by keyword
- `mh skills add <owner/repo@skill>` - Install a skill from GitHub

**Browse skills at:** https://skills.sh/

## How to Help Users Find Skills

### Step 1: Understand What They Need

When a user asks for help with something, identify:

1. The domain (e.g., React, testing, design, deployment)
2. The specific task (e.g., writing tests, creating animations, reviewing PRs)
3. Whether this is a common enough task that a skill likely exists

### Step 2: Search for Skills

Run the find command with a relevant query:

```bash
mh skills find [query]
```

For example:

- User asks "how do I make my React app faster?" → `mh skills find react performance`
- User asks "can you help me with PR reviews?" → `mh skills find pr review`
- User asks "I need to create a changelog" → `mh skills find changelog`

The command will return results like:

```
Found 10 skills:

Install with: mh skills add <owner/repo@skill>

anthropics/skills@mcp-builder
  mcp-builder (4060 installs)
  └ https://skills.sh/anthropics/skills/mcp-builder
```

### Step 3: Present Options to the User

When you find relevant skills, present them to the user with:

1. The skill name and what it does
2. The install command they can run
3. A link to learn more at skills.sh

Example response:

```
I found a skill that might help! The "mcp-builder" skill helps build MCP servers.

To install it:
mh skills add anthropics/skills@mcp-builder

Learn more: https://skills.sh/anthropics/skills/mcp-builder
```

### Step 4: Offer to Install

If the user wants to proceed, you can install the skill for them:

```bash
mh skills add <owner/repo@skill>
```

Skills are installed to `.agents/skills/<skill>` in the current directory.

## Common Skill Categories

When searching, consider these common categories:

| Category        | Example Queries                          |
| --------------- | ---------------------------------------- |
| Web Development | react, nextjs, typescript, css, tailwind |
| Testing         | testing, jest, playwright, e2e           |
| DevOps          | deploy, docker, kubernetes, ci-cd        |
| Documentation   | docs, readme, changelog, api-docs        |
| Code Quality    | review, lint, refactor, best-practices   |
| Design          | ui, ux, design-system, accessibility     |
| Productivity    | workflow, automation, git                |

## Tips for Effective Searches

1. **Use specific keywords**: "react testing" is better than just "testing"
2. **Try alternative terms**: If "deploy" doesn't work, try "deployment" or "ci-cd"
3. **Check popular sources**: Many skills come from `anthropics/skills` or `vercel-labs/agent-skills`

## When No Skills Are Found

If no relevant skills exist:

1. Acknowledge that no existing skill was found
2. Offer to help with the task directly using your general capabilities

Example:

```
I searched for skills related to "xyz" but didn't find any matches.
I can still help you with this task directly! Would you like me to proceed?
```

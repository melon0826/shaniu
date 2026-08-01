# User Instruction Memory

This file records user instructions, preferences, and teachings for reference in future interactions.

## Format

### User Instruction Entry
User instruction entries should follow this format:

[User Instruction Summary]
- Date: [YYYY-MM-DD]
- Context: [Mentioned scenario or time]
- Instructions:
  - [Content of user teaching or instruction, described line by line]

### Project Knowledge Entry
Entries discovered by the Agent during task execution should follow this format:

[Project Knowledge Summary]
- Date: [YYYY-MM-DD]
- Context: Discovered by Agent while performing [specific task description]
- Category: [Operations & Deployment|Build Methods|Testing Methods|Troubleshooting & Debugging|Workflow & Collaboration|Environment Configuration]
- Instructions:
  - [Specific knowledge points, described line by line]

## Deduplication Strategy
- Before adding a new entry, check for similar or identical instructions.
- If a duplicate is found, skip the new entry or merge it with the existing one.
- When merging, update the context or date information.
- This helps avoid redundant entries and keeps the memory file tidy.

## Entries

[Project Knowledge Summary]
- Date: 2026-08-01
- Context: Discovered by Agent while fixing failed CI and Docker image publishing for shaniu repo
- Category: Environment Configuration
- Instructions:
  - The GitHub repo has a security policy requiring ALL actions pinned to full-length commit SHA (version tags like @v7 are rejected)
  - Docker Hub secrets DOCKER_USERNAME/DOCKER_PASSWORD are NOT configured; CI Docker Hub steps are skipped, so images only get pushed to GHCR (ghcr.io/melon0826/shaniu, package is public)
  - Docker Hub publishing only activates after the user adds DOCKER_USERNAME/DOCKER_PASSWORD repo secrets

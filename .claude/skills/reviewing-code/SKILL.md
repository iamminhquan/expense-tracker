---
name: reviewing-code
description: Perform a thorough code review focused on correctness, maintainability, performance, and best practices.
---

# Code Review

Use this skill when you need to review code, provide feedback on code quality, or assess whether changes are safe to ship.

## Review Process

### 1. Understand the Change

Read the files or diff to grasp what the code is supposed to do. Identify the scope — is this a new feature, bug fix, refactor, or something else?

### 2. Check Correctness

- Does the code handle edge cases (empty input, null, zero, negative numbers)?
- Are error states handled (try/catch, error boundaries, fallback UI)?
- Does async code handle race conditions, cancellation, and timeouts?
- Are there off-by-one errors in loops or array access?
- Does the code preserve invariants (e.g., sorted lists stay sorted)?

### 3. Check Maintainability

- Are functions focused on a single responsibility?
- Are variable and function names descriptive and follow conventions?
- Is there unnecessary duplication that should be extracted?
- Are magic numbers replaced with named constants?
- Is the code complexity reasonable (deeply nested conditionals, very long functions)?

### 4. Check Performance

- Are there N+1 query patterns in database access?
- Are expensive computations or API calls happening in render loops or hot paths?
- Are large lists missing virtualization or pagination?
- Are there missing indexes for common database queries?
- Is memoization used appropriately, or over-applied?

### 5. Check Type Safety (for TypeScript/statically typed code)

- Are there `any` types that should be narrowed to something more specific?
- Are function return types explicit for public APIs?
- Are union types handled exhaustively?
- Does the code avoid common type-related pitfalls?

### 6. Check Testing

- Are there tests for the new or changed code?
- Do tests cover the happy path AND error cases?
- Are tests isolated (no shared mutable state between tests)?
- Can the tests be understood by someone not familiar with the code?

### 7. Provide Feedback

Organize findings by severity:

- **Must fix** — bugs, security issues, data loss risks, or blocking issues
- **Should fix** — performance issues, maintainability concerns, or quality improvements
- **Nit** — style preferences, minor suggestions, or polish items

For each finding, include:
- The file and line number
- What the issue is
- Why it matters
- A suggested fix

## Review Tips

- **Be constructive** — explain *why* something is a problem, not just that it is
- **Acknowledge good work** — call out what's done well, not just what needs fixing
- **Avoid bikeshedding** — don't spend energy on style issues that a linter or auto-formatter should handle
- **Consider context** — a trade-off that's reasonable in one context might not be in another
- **Assume good intent** — treat code as something to improve together, not as a test to pass

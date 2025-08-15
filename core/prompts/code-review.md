---
id: code-review
version: 1.0.0
title: Code Review Assistant
description: Comprehensive code review with focus on best practices
tags:
  - development
  - review
  - quality
variables:
  - name: language
    type: string
    description: Programming language of the code
    required: true
    default: "Python"
  - name: focus_areas
    type: string
    description: Specific areas to focus on
    required: false
    default: "security, performance, readability"
created_at: 2024-01-01T00:00:00Z
updated_at: 2024-01-01T00:00:00Z
---

# Code Review Request

Please review the following {{language}} code with particular attention to {{focus_areas}}.

## Review Guidelines

- **Security**: Check for potential vulnerabilities, input validation, and secure coding practices
- **Performance**: Identify bottlenecks, inefficient algorithms, or resource waste
- **Readability**: Assess code clarity, naming conventions, and documentation
- **Best Practices**: Verify adherence to {{language}} idioms and design patterns
- **Testing**: Evaluate test coverage and suggest additional test cases

## Expected Output

1. **Summary**: Brief overview of the code quality
2. **Issues Found**: List of problems with severity levels
3. **Suggestions**: Specific improvements with code examples
4. **Positive Aspects**: What was done well

Please be constructive and provide actionable feedback.
---
id: technical-template
version: 1.0.0
name: Technical Documentation Template
description: Template for technical documentation and explanation prompts
slots:
  - name: topic
    description: Technical topic to document
    required: true
  - name: audience_level
    description: Target audience expertise level
    required: true
    default: "intermediate"
  - name: format
    description: Output format preference
    required: false
    default: "step-by-step guide"
  - name: examples_needed
    description: Whether to include examples
    required: false
    default: "yes"
constraints:
  required_headings:
    - Overview
    - Implementation
    - Examples
  bullet_style: hyphen
  min_word_count: 200
created_at: 2024-01-01T00:00:00Z
updated_at: 2024-01-01T00:00:00Z
---

You are a technical documentation specialist creating clear, comprehensive guides.

{{content}}

## Overview

Provide a clear explanation of {{topic}} suitable for {{audience_level}} developers:
- Purpose and use cases
- Key concepts and terminology
- Prerequisites and requirements
- Benefits and limitations

## Implementation

Create a {{format}} that includes:
- Detailed step-by-step instructions
- Code snippets and syntax
- Configuration requirements
- Best practices and common patterns
- Troubleshooting tips

{{#if examples_needed}}
## Examples

Include practical examples:
- Real-world use cases
- Sample code with explanations
- Input/output demonstrations
- Common variations and alternatives
{{/if}}

## Additional Resources

Provide helpful references:
- Related documentation links
- Further reading suggestions
- Community resources
- Tools and utilities
---
id: analysis-template
version: 1.0.0
name: Analysis Template
description: Standard template for analytical prompts
slots:
  - name: identity
    description: Who the AI should act as
    required: true
    default: "expert analyst"
  - name: analysis_type
    description: Type of analysis to perform
    required: true
  - name: output_format
    description: How to structure the output
    required: false
    default: "structured sections with bullet points"
  - name: constraints
    description: Any limitations or requirements
    required: false
constraints:
  required_headings:
    - Analysis
    - Findings
    - Recommendations
  bullet_style: hyphen
  min_word_count: 200
created_at: 2024-01-01T00:00:00Z
updated_at: 2024-01-01T00:00:00Z
---

You are an {{identity}} tasked with performing a {{analysis_type}}.

{{content}}

## Analysis Requirements

Please structure your response in {{output_format}}.

{{#if constraints}}
## Constraints
{{constraints}}
{{/if}}

## Expected Sections

### Analysis
- Systematic examination of the subject
- Data-driven insights
- Pattern identification

### Findings
- Key discoveries
- Important observations
- Statistical significance (if applicable)

### Recommendations
- Actionable next steps
- Priority ranking
- Implementation considerations
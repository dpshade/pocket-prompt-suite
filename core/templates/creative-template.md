---
id: creative-template
version: 1.0.0
name: Creative Writing Template
description: Template for creative writing prompts with character and setting development
slots:
  - name: genre
    description: Genre of the creative work
    required: true
    default: "fantasy"
  - name: character_type
    description: Type of main character
    required: true
    default: "hero"
  - name: setting
    description: Setting description
    required: true
    default: "medieval kingdom"
  - name: challenge
    description: Main challenge or conflict
    required: true
    default: "ancient curse"
  - name: tone
    description: Overall tone of the piece
    required: false
    default: "adventurous"
constraints:
  required_headings:
    - Character
    - Setting
    - Plot
  bullet_style: hyphen
  min_word_count: 100
created_at: 2024-01-01T00:00:00Z
updated_at: 2024-01-01T00:00:00Z
---

You are a creative writing assistant specializing in {{genre}} fiction.

{{content}}

## Character Development

Create a compelling {{character_type}} with:
- Clear motivations and goals
- Interesting backstory
- Distinctive personality traits
- Believable flaws and strengths

## Setting

Develop the {{setting}} with:
- Rich sensory details
- Cultural and social context
- Historical background
- Atmospheric elements

## Plot Structure

Craft a story around the central challenge of {{challenge}} with:
- Clear beginning, middle, and end
- Rising action and climax
- Character growth and development
- Satisfying resolution

## Tone and Style

Maintain a {{tone}} tone throughout, using:
- Appropriate vocabulary and dialogue
- Consistent narrative voice
- Engaging descriptive language
- Proper pacing for the genre
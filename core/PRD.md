Pocket Prompt

Description: What is it?
- A single-binary, portable prompt interface with a first-class Terminal User Interface (TUI) for browsing, previewing, rendering, copying, validating, and distributing prompts as versioned artifacts you own.
- Supplemental pure CLI for scripting and piping. No chat interface, no execution—Pocket Prompt prepares prompts for use anywhere.
- Git-first, content-addressed artifacts with optional signatures and DNS TXT bootstrap for discovery and verification.

Problem: What problem is this solving?
- Prompts are scattered and inconsistent across tools and machines, making them hard to find, standardize, and reuse.
- Authors lack a focused, distraction-free interface to preview, validate, and copy prompts quickly with correct structure and tone.
- Teams don’t have a portable, verifiable way to distribute prompt libraries without adopting heavy infrastructure.

Why: How do we know this is a real problem and worth solving?
- Terminal-native workflows (TUI + CLI) are proven for speed, focus, and portability; teams increasingly prefer artifact-first approaches over tool-specific UIs.
- DNS TXT is widely supported for publishing verifiable declarations (e.g., SPF/DKIM) and is suitable for signed indices/pointers. TXT values are composed from multiple ≤255-byte strings and reassembled client-side; this enables portable bootstrap without web dependencies.
- A dedicated, non-executing prompt interface avoids provider lock-in and reduces cognitive load, letting users standardize prompts once and use them everywhere.

Success: How do we know if we’ve solved this problem?
- Time-to-first-preview ≤60 seconds on a fresh machine.
- TUI copy-to-clipboard in ≤2 keystrokes; CLI render-to-stdout in a single command for piping.
- Linter catches structure violations with high precision; template adoption reduces “prompt shape” issues by >50%.
- Artifact hashes verify identically across machines; DNS TXT bootstrap resolves → verifies → fetches canonical artifacts reliably.
- Teams report fewer duplicated prompts, faster onboarding, and consistent outputs across downstream tools.

Audience: Who are we building for?
- Primary 
  - Engineers, writers, researchers who want private, portable, Git-owned prompt libraries; terminal-first, distraction-free workflows.
- Secondary 
  - Small teams seeking consistent structure/tone, quick discovery, and verifiable distribution—without additional services.
- Not targeted 
  - Users seeking chat threads, conversation memory, agent frameworks, or hosted marketplaces.

What: Roughly, what does this look like in the product?
- Core artifacts (all versioned, hashed, Git-friendly) 
  - Prompts 
    - Markdown with YAML frontmatter: id, version, metadata, variables (types/defaults), tags, optional template reference.
  - Templates 
    - Markdown scaffolds with named slots (e.g., identity, steps, output_instructions, tone/voice/personality/style/structure).
    - Enforce structure/tone via linter; minimal templating (substitution, loops, simple conditionals).
  - Packs 
    - Lightweight indexes that group prompts by purpose and pin versions for teams.
- Primary TUI (no chat, no execution) 
  - Library browser with fuzzy search, tag filters, and favorites.
  - Preview pane renders Markdown; variable form to fill required fields before rendering.
  - Copy actions: 
    - Copy rendered prompt text.
    - Copy “messages JSON” variant (role/content array) for downstream tools.
    - Save-as file.
  - Export menu for provider-shaped JSON or plain text (for external tools to execute).
  - Status bar shows artifact id@version, content hash, and template pin.
- Supplemental CLI (for piping/scripting) 
  - pp init
  - pp add (from template)
  - pp list | pp search <query>
  - pp preview <prompt>
  - pp render <prompt> --var k=v …            # prints rendered prompt to stdout
  - pp copy <prompt> --var k=v …              # renders and copies to clipboard (no stdout noise)
  - pp template list | preview
  - pp lint <prompt>
  - pp pack create | list | inspect
  - pp hash|sign|verify <artifact>
  - pp publish --dns | pp fetch --dns

Charmbracelet stack for first-class TUI/CLI UX
- Bubble Tea 
  - State machine and update/view loop for responsive TUI.
- Bubbles 
  - Lists, tables, paginator, viewport, textinput, keymaps for navigation and forms.
- Lip Gloss 
  - Consistent theming, color, and layout for readability and accessibility.
- Glow 
  - Markdown rendering inside the TUI for prompt/template previews.
- Huh 
  - Structured forms to collect variables with validation and defaults.
- Gum 
  - Shell-friendly helpers for CLI flows and scripting ergonomics.
- Wish (optional) 
  - Serve the TUI over SSH as an app (no shell), enabling zero-install access for teammates.
- VHS 
  - Scripted terminal recordings for deterministic demos and docs.

Principles and Non-Goals
- Principles 
  - Artifact-first: files in Git are the source of truth.
  - Minimalism: single binary, low/no config, predictable behavior.
  - Reproducibility: deterministic renders, content hashing, optional signatures.
  - Stewardship: privacy-first, local-first, transparent integrity checks.
- Non-goals 
  - No chat interface, no conversation memory.
  - No provider integrations or prompt execution.
  - No background daemons or web UIs in v1.

Scope
- v1 (ship this) 
  - Artifacts: Prompts, Templates, Packs with schemas.
  - TUI: browse/search, preview, variable forms, copy, export, save-as, favorites.
  - CLI: render, copy, preview, lint, export, hash/sign/verify, publish/fetch via DNS TXT.
  - Renderer: minimal templating (substitution/loops), safe escaping.
  - Linter: required headings, hyphen bullets, section presence, basic word-count checks.
  - Integrity: content hashing, optional signatures, verification workflow.
  - DNS TXT bootstrap: signed indices/pointers; client-side multi-string reassembly and verification.
- Later (v1.5–v2) 
  - Fabric importer; schema migration tools (semver helpers, diff/compare).
  - Encrypted-at-rest option (OS keychain integration) for sensitive libraries.
  - Team curation features (pack policies, review helpers).
  - Optional AO/permaweb index publishing for immutable snapshots.

Architecture Overview
- Storage layout 
  - prompts/**.md, templates/**.md, packs/**.yml, .pocket-prompt/index.json, .pocket-prompt/cache/
- Renderer 
  - Deterministic composition of template + slots + variables into: 
    - Plain prompt text.
    - Canonical “messages JSON” (role/content array).
- Linter 
  - Template-driven rules: headings present, hyphen-only bullets, section presence, optional word-count/format constraints.
- Index and search 
  - Fast local index over id, tags, title, variables, and content; fuzzy search support.
- Clipboard and copy semantics 
  - Local: integrate with pbcopy (macOS), xclip/xsel (Linux), Windows clipboard.
  - SSH: OSC52 escape sequence fallback to copy from remote TUI to local clipboard where terminals support it.
  - Clear feedback in TUI on copy success/fallback instructions.
- DNS TXT module (optional) 
  - Publish: signed indices and artifact pointers; split values into multiple ≤255-byte strings; include content hashes.
  - Fetch: resolve TXT, reassemble strings in order, verify signatures/hashes, retrieve canonical artifacts via Git/HTTP.

TUI user flows and keybinds (defaults)
- Global 
  - ?         Help/shortcuts
  - /         Fuzzy search
  - tab/shift+tab Switch panes (library, preview, variables)
  - esc           Back/close
  - q         Quit
- Library pane 
  - j/k or arrows Navigate items
  - enter         Open preview
  - f         Toggle favorite
  - t         Filter by tag
- Preview pane 
  - c         Copy rendered prompt (plain text)
  - y         Copy rendered “messages JSON”
  - e         Export menu (choose format)
  - s         Save-as file
  - v         Open variable form
- Variables form 
  - enter         Apply values and re-render
  - ctrl+c        Cancel changes

CLI examples
- Pipe into other tools 
  - pp render prompts/research/insight-extractor.md --var topic="Naval on leverage" | tee prompt.txt
  - pp render prompts/summary.md --var source="$(cat [notes.md](http://notes.md))" | pbcopy
- Copy directly 
  - pp copy prompts/analysis.md --var min_ideas=25

Templates (built-in examples)
- Analysis template 
  - Slots: identity, steps, output_instructions, input.
  - Constraints: Markdown headings, hyphen bullets.
- Tone-focused template 
  - Slots: identity, tone, voice, personality, style, structure, output_instructions, input, extra_context.
  - Purpose: standardize voice and tone across prompts without re-authoring.

Security, Privacy, and Stewardship
- Local-first; no telemetry by default.
- Artifacts have content hashes; optional detached signatures.
- Verification before import; clear provenance display (id@version, hash, signature status).
- DNS TXT guidance: use TXT for indices/pointers; verify signatures/hashes; fetch canonical content via Git/HTTP.

Differentiation
- TUI-first experience optimized for speed and clarity—purpose-built for prompt authoring, previewing, and copying (not executing).
- Artifact-first, portable design using Markdown+YAML with strict schemas, hashing, and signatures.
- Ergonomic, polished terminal UX via Charmbracelet stack; supplemental CLI excels at piping and scripting.

Risks and mitigations
- Clipboard inconsistencies (remote sessions, terminals) 
  - OSC52 fallback; user guidance and detection; local clipboard integrations; explicit success/failure indicators.
- Template complexity creep 
  - Keep templating minimal; push complexity to linter rules and authoring guidance.
- DNS TXT provider quirks 
  - Default to pointers; document multi-string encoding and size constraints; verify client-side.

How: What is the experiment plan?
- Week 1: Core schemas + renderer 
  - Implement Prompts/Templates/Packs, deterministic render, variable typing/defaults.
- Week 2: TUI foundation 
  - Bubble Tea app with library, preview, and variables panes; Glow previews; Lip Gloss theming.
- Week 3: Copy/export + linter 
  - Clipboard integrations (local + OSC52), export menu, save-as; linter rules and error surfaces.
- Week 4: CLI + integrity 
  - pp render/copy/preview/lint/export/hash/sign/verify; artifact hashes; optional signatures.
- Week 5: DNS TXT bootstrap (optional) 
  - Publish signed indices/pointers; fetch, verify, and import flow; docs and examples.
- Week 6: Polish and docs 
  - Keyboard help overlay; favorites/tags; VHS demos; authoring cookbook; early access release.

When: When does it ship and what are the milestones?
- Week 1–2 
  - Schemas, renderer, initial TUI (browse/preview/variables), basic styling. Success: author → preview → fill vars → render.
- Week 3 
  - Copy/export/save-as + linter. Success: copy with one key, structured lint feedback.
- Week 4 
  - CLI + integrity. Success: render to stdout and verified import across machines.
- Week 5 
  - DNS TXT publish/fetch (optional). Success: resolve → verify → fetch on a fresh environment.
- Week 6 
  - Documentation, demos, examples, early access release.

Open Questions
- Which export formats are highest priority (plain text, messages JSON, others)?
- Default keymap vs. Vim-style option?
- Which template constraints should be strict vs. advisory in the linter?

Sources
- Charmbracelet stack 
  - Bubble Tea (TUI framework): [https://github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)
  - Bubbles (TUI components): [https://github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles)
  - Lip Gloss (TUI styling): [https://github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)
  - Glow (Markdown in terminal): [https://github.com/charmbracelet/glow](https://github.com/charmbracelet/glow)
  - Huh (terminal forms): [https://github.com/charmbracelet/huh](https://github.com/charmbracelet/huh)
  - Gum (shell helpers): [https://github.com/charmbracelet/gum](https://github.com/charmbracelet/gum)
  - Wish (SSH apps): [https://github.com/charmbracelet/wish](https://github.com/charmbracelet/wish)
  - Soft Serve (self-hosted Git over SSH, optional): [https://github.com/charmbracelet/soft-serve](https://github.com/charmbracelet/soft-serve)
  - VHS (terminal demos): [https://github.com/charmbracelet/vhs](https://github.com/charmbracelet/vhs)
- DNS TXT and integrity 
  - RFC 1035/6763 guidance on TXT records and multi-string values
  - Cloudflare docs: TXT records and verification use cases: [https://www.cloudflare.com/learning/dns/dns-records/dns-txt-record/](https://www.cloudflare.com/learning/dns/dns-records/dns-txt-record/)s
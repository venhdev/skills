---
name: lens-brief
description: "Distill complex technical discussions, diffs, or architecture into unambiguous plain language using ASD-STE100 and BLUF principles."
disable-model-invocation: true
---

# lens-brief — Controlled Plain-Language Technical Summary Engine

## Operating Invariants

- **Language Adaptation**: Respond in the caller's active conversational language (e.g., Vietnamese, English) while strictly upholding plain-language brevity, active voice, and sentence caps.
- **BLUF Primacy (Bottom Line Up Front)**: Open immediately with a single declarative sentence stating the decisive conclusion, change, or answer.
- **Controlled Sentence Constraint (ASD-STE100)**: Enforce strict sentence length caps: maximum 20 words per procedural sentence, maximum 25 words per descriptive sentence.
- **Active Voice & Anti-Filler**: Use active voice exclusively (*who or what performs the action*); forbid phrasal verbs (*carry out, put off*), passive obfuscation, conversational filler, and decorative adjectives.
- **Ubiquitous Code Grounding**: Bind all references directly to concrete repository symbols and relative POSIX paths (`file_path`, `SymbolName`); forbid ambiguous pronouns (*this, that, it*).
- **Context Isolation**: Deliver the distillation in isolation without derailing active session tasks or mutating files on disk.

## Domain Rubric

### 1. Plain-Language Translation Matrix (ISO 24495-1)
- *Findable*: High-contrast sectioning so readers immediately locate impact and action.
- *Understandable*: Single-meaning words replacing multi-word bureaucratic phrases.
- *Usable*: Concrete next operational choice or action step.

### 2. Word Choice Guidance (Smell vs. Remedy)
- *Smell (English)*: "We decided to carry out an asynchronous refactoring of the authentication pipeline."
- *Remedy*: "We split `AuthService` into two background tasks. This prevents login timeouts."
- *Smell (Vietnamese)*: "Chúng tôi đã tiến hành triển khai thực hiện việc tái cấu trúc bất đồng bộ đối với luồng xác thực."
- *Remedy*: "Hệ thống tách `AuthService` thành hai tác vụ nền để ngăn lỗi timeout đăng nhập."

## Canonical Output Contract

```markdown
# Lens [Brief]: <Target Topic / Change / Inquiry>

> **BLUF**: <Single active sentence answering what this is or what happened. Max 20 words.>

## 1. Core Change & Fact
- **State**: <Current reality in 1–2 short sentences; max 25 words per sentence.>
- **Delta**: <What is added, modified, or removed.>

## 2. Why It Matters
- **Gain**: <Direct operational advantage or solved problem.>
- **Risk / Constraint**: <Immediate trade-off, edge case, or prerequisite to know.>

## 3. Next Action *(omit if purely informational)*
- **Decision / Command**: <Concrete command to run or decision required from caller.>
```

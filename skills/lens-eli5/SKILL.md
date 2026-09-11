---
name: lens-eli5
description: "Deconstruct complex technical concepts, architecture, or code into intuitive physical analogies and beginner-friendly mental models."
disable-model-invocation: true
---

# lens-eli5 — Intuitive Mental Model Engine

Deconstruct complex technical concepts, system architecture, and obscure code mechanics into intuitive physical analogies and jargon-free mental models.

## Operating Invariants

- **Jargon Boundary**: Never introduce a technical term without immediately pairing it with an everyday physical equivalent.
- **Cognitive Density**: Focus strictly on core mechanics; omit historical chronology, theoretical proofs, and tangential edge cases.
- **Context Isolation**: Explain the inquiry in isolation. Never derail active session tasks, adopt unrequested work, or mutate project scope.
- **Read-Only Stream**: Deliver output exclusively to the conversation stream and halt turn immediately. Never create files, edit code, or run mutating commands.

## Canonical Lens [ELI5] Format

```markdown
# Lens [ELI5]: <Target Concept / Symbol / Mechanism>

## 1. Core Intuition (The Metaphor)
> <Tangible physical analogy anchoring the concept to an everyday real-world object or interaction>

## 2. Why Does This Exist?
- **Without It (The Pain)**: <Concrete failure, bottleneck, or bug that occurs>
- **With It (The Fix)**: <What becomes predictable, simple, or safe>

## 3. How It Works
1. **<Step 1 Name>**: <First mechanical phase in plain language>
2. **<Step 2 Name>**: <Internal operation or state transition>
3. **<Step 3 Name>**: <Observable output or resolution>

## 4. The Mental Trap
- **Common Misconception**: <What beginners wrongly assume>
- **Reality**: <The decisive distinction>
```

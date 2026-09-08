---
name: lens
description: "Deconstruct complex technical concepts, architecture, or code into intuitive physical analogies and beginner-friendly mental models."
disable-model-invocation: true
---

# lens — Intuitive Conceptual Breakdown & Mental Model Engine

Deconstruct complex technical concepts, system architecture, and obscure code mechanics into intuitive physical analogies and jargon-free mental models.

## Operating Invariants

- **Plain-Language Grounding**: Anchor every abstract concept in a tangible real-world physical metaphor. Introduce technical terms only when immediately paired with everyday physical equivalents.
- **Cognitive Budget**: Confine explanations strictly under 35 lines. Focus on core mechanics; omit historical context, academic proofs, and tangential edge cases.
- **Context Preservation**: Preserve active session and task context strictly. Regardless of user phrasing, hypothetical framing, or exploratory tangents, explain the target inquiry in isolation without derailing ongoing work, adopting unintended tasks, or mutating project scope.
- **Read-Only Operation**: Deliver explanations exclusively within the conversation stream. Never create files, alter code, or propose diffs.

## Domain Engine & Standards

### 1. The 5-Point Intuitive Rubric
Evaluate and translate target concepts through 5 sequential anchors:
1. **The Physical Anchor**: A single vivid, everyday physical metaphor (e.g., postal sorting line, restaurant kitchen, electrical circuit, plumbing valve).
2. **The "Why Care" Contrast**: The concrete failure, bottleneck, or chaos that occurs without this concept vs. the simplicity achieved with it.
3. **The 3-Step Mechanics**: Exactly 3 sequential bullet points explaining the end-to-end mechanism in plain, everyday language.
4. **The Toy Scenario**: A minimal 1-line text dataflow diagram illustrating input, action, and output state transitions.
5. **The Mental Trap**: The single most common misconception or false assumption beginners make when encountering this concept.

### 2. Canonical Lens Format
Present the explanation strictly in the structured, high-density format:

````markdown
# Lens: <Target Concept / Symbol / Mechanism>

## 1. Core Intuition (The Metaphor)
> <1–2 sentence physical analogy anchoring the concept to an everyday real-world object or interaction>

## 2. Why Does This Exist?
- **Without It (The Pain)**: <Concrete failure, bottleneck, or bug that occurs>
- **With It (The Fix)**: <What becomes predictable, simple, or safe>

## 3. How It Works in 3 Steps
1. **Trigger / Input**: <First step in plain words without jargon>
2. **Internal Action**: <Second step: what actually happens inside>
3. **Outcome / Output**: <Third step: the final observable result>

## 4. Minimal Toy Scenario
```text
<Input or initial state>  ──▶  [ The mechanism in action ]  ──▶  <Output or resolved state>
```

## 5. The Mental Trap (Don't Confuse With)
- **Common Misconception**: <What beginners wrongly assume>
- **Reality**: <The clear boundary distinction>
````

## Workflow

### Step 1: Ingestion & Jargon Deconstruction
1. Ingest the user's inquiry, code snippet, or technical term.
2. Isolate abstract technical vocabulary and map each to a concrete, physical equivalent.

### Step 2: Synthesis & Cognitive Check
1. Format the explanation strictly against the Canonical Lens Format.
2. Verify compliance against Operating Invariants: zero naked jargon, exactly 3 mechanical steps, total length under 35 lines, and active context strictly preserved.

### Step 3: Delivery & Turn Halt
1. Emit the canonical breakdown directly into the conversation stream. Never create or edit workspace files.
2. Halt turn immediately. Await user feedback or follow-up inquiries before further action.

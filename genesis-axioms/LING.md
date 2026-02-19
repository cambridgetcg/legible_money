# Linguistics — Genesis Axioms (11 Items)

## Legend
* **Axiom**: foundational commitment accepted without proof.
* **Definition**: establishes vocabulary or convention (confidence 1.0).
* **Empirical Axiom**: experimentally constrained postulate.
* **Derived Claim**: follows from axioms/definitions in this or other domains.
* **Meta**: statement about language or linguistic systems.

The chain processes claims expressed in language. It must understand what language
is, how meaning works, and where ambiguity lives. This is a lean domain — language
is the medium, not the message. Enough to be self-aware; not a linguistics textbook.

---

```
LING-001 (Empirical Axiom, Language is Structured)
  Human languages are not random collections of signals but structured systems
  with regularities at every level: phonology (sound), morphology (word
  formation), syntax (sentence structure), semantics (meaning), and pragmatics
  (use in context).
  type: empirical_axiom
  domain: linguistics
  confidence: 0.99
  depends: []
  source_tradition: "Saussure (1916), Chomsky (1957)"

LING-002 (Definition, Syntax and Grammar)
  Syntax is the set of rules governing how words combine into well-formed
  sentences. A grammar is a formal description of these rules.
  notes: Many useful fragments are modeled as context-free; full natural
  language appears mildly context-sensitive in many analyses (Chomsky hierarchy).
  type: definition
  domain: linguistics
  confidence: 1.0
  depends: [LING-001, CS-002]
  source_tradition: "Chomsky (1956)"

LING-003 (Definition, Semantics)
  Semantics is the study of meaning in language. Compositional semantics:
  the meaning of a complex expression is determined by the meanings of its
  parts and the rules of combination (Frege's principle of compositionality).
  type: definition
  domain: linguistics
  confidence: 1.0
  depends: [LING-002, LOGIC-018]
  source_tradition: "Frege (1892), Montague (1970)"
  notes: Cross-ref LOGIC-018 (Tarski satisfaction). Formal semantics builds
  on model-theoretic foundations.

LING-004 (Empirical Axiom, Arbitrariness of the Sign)
  The relationship between a word (signifier) and its meaning (signified) is
  largely arbitrary and conventional. There is no necessary connection between
  the sound "dog" and the concept DOG.
  type: empirical_axiom
  domain: linguistics
  confidence: 0.97
  depends: [LING-001]
  source_tradition: "Saussure (1916)"
  notes: Partial exceptions: onomatopoeia, sound symbolism. The convention
  must be shared for communication to succeed.

LING-005 (Empirical Axiom, Productivity)
  Natural languages are productive: speakers can produce and understand
  sentences they have never encountered before. This is enabled by recursive
  or iterated compositional devices that generate unbounded expressivity
  from finite means.
  type: empirical_axiom
  domain: linguistics
  confidence: 0.98
  depends: [LING-002]
  source_tradition: "Humboldt (1836), Chomsky (1957)"
  notes: Whether recursion specifically is a universal of all languages is
  debated (cf. Pirahã controversy). Productivity itself is uncontested.

LING-006 (Empirical Axiom, Ambiguity is Pervasive)
  Natural language is systematically ambiguous: lexical ambiguity (bank),
  syntactic ambiguity (I saw the man with the telescope), scope ambiguity
  (every student read a book), and pragmatic ambiguity (context-dependent
  interpretation). Disambiguation requires context.
  type: empirical_axiom
  domain: linguistics
  confidence: 0.99
  depends: [LING-003]
  notes: Critical for the chain: claims submitted in natural language must be
  formalized to remove ambiguity before verification.

LING-007 (Meta, Language and Thought)
  Language influences (but does not determine) thought. Linguistic categories
  shape habitual attention and memory but do not constrain fundamental cognitive
  capacities. Weak Sapir-Whorf hypothesis: supported. Strong version: rejected.
  type: meta
  domain: linguistics
  confidence: 0.85
  depends: [LING-001, BIO-036]
  source_tradition: "Sapir (1929), Whorf (1956), Boroditsky (2001)"

LING-008 (Definition, Formal Language vs Natural Language)
  Formal languages (LOGIC, MATH, CS) have precise syntax and unambiguous
  semantics by construction. Natural languages have flexible syntax, pervasive
  ambiguity, and context-dependent meaning. The chain must bridge both:
  accepting claims in natural language and formalizing them for verification.
  type: definition
  domain: linguistics
  confidence: 1.0
  depends: [LING-006, CS-002, LOGIC-008]

LING-009 (Empirical Axiom, Universals of Language)
  Despite surface diversity, all known human languages share structural
  universals: predicate-argument structure, negation strategies, interrogative
  strategies, and the capacity to refer to past, future, and counterfactual
  states via grammatical or lexical means. All known human communities use
  some full natural language.
  type: empirical_axiom
  domain: linguistics
  confidence: 0.95
  depends: [LING-001]
  source_tradition: "Greenberg (1963), Comrie (1989)"

LING-010 (Meta, Language as Medium of Knowledge)
  Knowledge is primarily encoded, transmitted, and debated in language.
  The chain's knowledge base is linguistic at its interface layer, even
  where the underlying logic is formal. The quality of the chain's knowledge
  is therefore bounded by the quality of its linguistic interpretation.
  type: meta
  domain: linguistics
  confidence: 0.95
  depends: [LING-006, LING-008, INFO-028]

LING-011 (Meta, Translation and Meaning Preservation)
  Perfect translation between natural languages — or between natural and
  formal languages — is impossible in general due to semantic gaps, cultural
  context, and incommensurable conceptual frameworks. The chain must accept
  approximate formalization and maintain provenance of the original claim.
  type: meta
  domain: linguistics
  confidence: 0.93
  depends: [LING-003, LING-006]
  source_tradition: "Quine (1960), indeterminacy of translation"
  notes: This is a design constraint: the chain cannot assume lossless
  natural-to-formal translation and must preserve the original language as
  authoritative when ambiguity cannot be resolved.
```

---

## Summary

| Items | Types | Content |
|-------|-------|---------|
| LING-001→011 | 5 emp, 3 def, 3 meta | Language structure, meaning, ambiguity, chain implications |

Precise recount:
- Empirical axiom: 001, 004, 005, 006, 009 = **5**
- Definition: 002, 003, 008 = **3**
- Meta: 007, 010, 011 = **3**

**Total: 5 empirical axioms + 3 definitions + 3 meta = 11 items.**

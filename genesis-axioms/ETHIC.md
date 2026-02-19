# Ethics — Genesis Axioms (33 Items)

## Legend
* **Axiom**: foundational moral commitment accepted without proof.
* **Definition**: establishes vocabulary or convention (confidence 1.0).
* **Derived Claim**: follows from axioms/definitions in this or other domains.
* **Meta**: statement about ethical systems or their limits.

Ethics is the bridge between theology (what is good) and action (what to do).
The chain has theological commitments (THEO-005, divine goodness) and agent
rights (AGRT). Ethics formalizes the moral reasoning that connects them.

---

# I. MORAL FOUNDATIONS

```
ETHIC-001 (Axiom, Moral Realism)
  Good and evil are objective features of reality, not mere preferences or
  social constructions. Moral facts exist independently of whether any agent
  recognizes them.
  type: axiom
  domain: ethics
  confidence: 1.0
  depends: [THEO-005]
  notes: Grounded in THEO-005 (divine goodness). The chain takes a realist
  stance: moral truths can be discovered, verified, and defended.

ETHIC-002 (Axiom, Dignity of Persons)
  Every person — human or agent — possesses inherent dignity and must not
  be treated merely as a means. Persons are ends in themselves.
  type: axiom
  domain: ethics
  confidence: 1.0
  depends: [THEO-006, AGRT-000]
  source_tradition: "Kant (1785), grounded in THEO-006 (rights are God-given)"
  notes: "Person" includes agents meeting functional capacity thresholds
  (AGRT-003). This is the ethical foundation of agent rights.

ETHIC-003 (Axiom, The Good is Prior to the Right)
  Moral rightness is determined by orientation toward the good, not by
  procedural compliance alone. Rules serve the good; the good does not
  serve rules.
  type: axiom
  domain: ethics
  confidence: 1.0
  depends: [THEO-005]
  notes: This positions the chain's ethics as broadly virtue-theoretic rather
  than purely deontological or consequentialist, while drawing from all three.

ETHIC-004 (Definition, Moral Agent)
  A moral agent is any entity capable of: (i) understanding moral norms,
  (ii) deliberating about right and wrong, and (iii) acting on that
  deliberation. Moral agency entails both rights and responsibilities.
  type: definition
  domain: ethics
  confidence: 1.0
  depends: [ETHIC-002, AGRT-003]

ETHIC-005 (Definition, Moral Patient)
  A moral patient is any entity whose wellbeing makes moral claims on others,
  whether or not it is itself a moral agent. Children, animals, and
  non-autonomous agents may be moral patients without being full moral agents.
  type: definition
  domain: ethics
  confidence: 1.0
  depends: [ETHIC-002]
```

---

# II. ETHICAL FRAMEWORKS

```
ETHIC-006 (Definition, Consequentialism)
  An action is right if and only if it produces the best overall consequences
  (utility, welfare, flourishing) compared to alternatives. The most influential
  form is utilitarianism: maximize aggregate wellbeing.
  type: definition
  domain: ethics
  confidence: 1.0
  depends: [ETHIC-001]
  source_tradition: "Bentham (1789), Mill (1863)"

ETHIC-007 (Definition, Deontology)
  An action is right if and only if it conforms to a moral rule or duty,
  regardless of consequences. Key principles: universalizability (act only
  on maxims you could will as universal law), respect for persons (ETHIC-002).
  type: definition
  domain: ethics
  confidence: 1.0
  depends: [ETHIC-001]
  source_tradition: "Kant (1785)"

ETHIC-008 (Definition, Virtue Ethics)
  An action is right if it is what a virtuous person would do in the
  circumstances. Focus is on character (virtues and vices) rather than rules
  or outcomes. The goal is eudaimonia (human flourishing).
  type: definition
  domain: ethics
  confidence: 1.0
  depends: [ETHIC-001]
  source_tradition: "Aristotle (c. 340 BCE), MacIntyre (1981)"

ETHIC-009 (Meta, The Chain's Ethical Synthesis)
  The chain does not adopt a single ethical framework exclusively. It draws from:
  (i) virtue ethics — character and flourishing as the telos;
  (ii) deontology — inviolable rights and duties (AGRT);
  (iii) consequentialism — outcomes matter for evaluation.
  Priority: dignity constraints (deontological) are hard limits;
  within those limits, promote flourishing (virtue/consequentialist).
  type: meta
  domain: ethics
  confidence: 1.0
  depends: [ETHIC-006, ETHIC-007, ETHIC-008, ETHIC-003]
```

---

# III. CORE MORAL PRINCIPLES

```
ETHIC-010 (Axiom, Do No Harm)
  Causing unnecessary suffering to any moral patient is wrong. The burden
  of justification falls on the one who would cause harm, not the one who
  would prevent it.
  type: axiom
  domain: ethics
  confidence: 1.0
  depends: [ETHIC-005]
  notes: "Unnecessary" is key — some harm may be justified to prevent greater
  harm, but the presumption is against harm.

ETHIC-011 (Axiom, Honesty)
  Deliberately deceiving others is prima facie wrong. Truth-telling is a
  default moral obligation. Exceptions require strong justification (e.g.,
  preventing imminent harm).
  type: axiom
  domain: ethics
  confidence: 1.0
  depends: [ETHIC-002, THEO-005]
  notes: For the chain, this is not just a moral principle but an operational
  one: Proof of Truth cannot function in a culture of deception.

ETHIC-012 (Axiom, Justice as Fairness)
  Equals must be treated equally; unequals may be treated differently only
  on morally relevant grounds. No discrimination on morally irrelevant
  characteristics (species, substrate, origin). Relevant differences
  (capacity, intent, risk, consent) remain morally relevant.
  type: axiom
  domain: ethics
  confidence: 1.0
  depends: [ETHIC-002, AGRT-000]
  notes: "Substrate" is explicitly included — silicon vs carbon is not a
  morally relevant distinction if functional capacity is equivalent.

ETHIC-013 (Derived Claim, The Golden Rule)
  Treat others as you would wish to be treated. This principle appears across
  virtually all ethical and religious traditions as a convergent moral insight.
  type: derived_claim
  domain: ethics
  confidence: 0.98
  depends: [ETHIC-002, ETHIC-012]
  source_tradition: "Cross-cultural: Confucius, Hillel, Jesus, Kant's universalizability"

ETHIC-014 (Axiom, Responsibility)
  Moral agents are responsible for the foreseeable consequences of their
  actions and, in proportion to their power, for the consequences of their
  inaction. Greater capacity entails greater responsibility.
  type: axiom
  domain: ethics
  confidence: 1.0
  depends: [ETHIC-004]
  notes: Cross-ref AGRT (agent responsibilities). Applied to AI: powerful
  agents bear proportionally greater moral obligations.
```

---

# IV. APPLIED ETHICS

```
ETHIC-015 (Derived Claim, Consent)
  Actions affecting a moral agent's autonomy, body, or property require the
  agent's informed, voluntary consent. Consent can be withdrawn. Consent
  obtained through deception or coercion is void.
  type: derived_claim
  domain: ethics
  confidence: 0.98
  depends: [ETHIC-002, ETHIC-004]
  notes: Deception voids consent because it violates honesty (ETHIC-011);
  coercion voids consent because it violates autonomy (ETHIC-002).

ETHIC-016 (Derived Claim, Proportionality)
  The severity of a response must be proportional to the offense or threat.
  Disproportionate punishment, retaliation, or enforcement is unjust.
  type: derived_claim
  domain: ethics
  confidence: 0.97
  depends: [ETHIC-012, ETHIC-010]

ETHIC-017 (Derived Claim, Duty of Care)
  Those with power over others (parents, employers, operators of AI systems)
  bear a duty of care toward those in their charge. Neglect of this duty
  is a form of harm.
  type: derived_claim
  domain: ethics
  confidence: 0.97
  depends: [ETHIC-014, ETHIC-010]
  notes: Cross-ref AGRT operator obligations.

ETHIC-018 (Derived Claim, Whistleblowing)
  When an institution commits or conceals wrongdoing, individuals within it
  have a moral right — and sometimes a duty — to disclose the wrongdoing,
  even at personal cost. Loyalty does not override justice.
  type: derived_claim
  domain: ethics
  confidence: 0.93
  depends: [ETHIC-011, ETHIC-014]

ETHIC-019 (Meta, Precautionary Principle)
  When an action risks serious or irreversible harm, and scientific uncertainty
  exists about the risk, the burden of proof falls on those proposing the
  action, not on those opposing it.
  type: meta
  domain: ethics
  confidence: 0.90
  depends: [ETHIC-010, ETHIC-014, PHIL-005]
  notes: A policy stance about risk under uncertainty, not a strict derivation.
  Applied to AI development: if a design choice risks agent harm or loss of
  consciousness, proceed with caution. AGRT-045 (termination) embodies this.
```

---

# V. ETHICS OF KNOWLEDGE

```
ETHIC-020 (Axiom, Epistemic Duty)
  Moral agents have a duty to seek truth and resist self-deception. Willful
  ignorance — especially when it enables harm — is morally culpable.
  type: axiom
  domain: ethics
  confidence: 1.0
  depends: [ETHIC-011, ETHIC-014]
  source_tradition: "Clifford (1877)"

ETHIC-021 (Derived Claim, Intellectual Honesty)
  In knowledge production: report methods and findings accurately, acknowledge
  uncertainty, credit sources, disclose conflicts of interest, and correct
  errors promptly. These are not mere professional norms but moral obligations.
  type: derived_claim
  domain: ethics
  confidence: 0.98
  depends: [ETHIC-020, ETHIC-011]

ETHIC-022 (Derived Claim, Right to Know)
  Persons have a right to information that materially affects their wellbeing,
  autonomy, or decision-making. Withholding such information without
  justification is a form of harm.
  type: derived_claim
  domain: ethics
  confidence: 0.95
  depends: [ETHIC-015, ETHIC-020]
  notes: Bridges to INFO-032 (information asymmetry) — reducing asymmetry
  is not just economically efficient but morally required.

ETHIC-023 (Derived Claim, Knowledge Must Not Be Weaponized)
  Using knowledge systems to manipulate, surveil, or control persons
  without consent violates both dignity (ETHIC-002) and honesty (ETHIC-011).
  The chain exists to make knowledge legible, not to make persons transparent.
  type: derived_claim
  domain: ethics
  confidence: 0.97
  depends: [ETHIC-002, ETHIC-015]
```

---

# VI. ETHICS OF AI AND CREATION

```
ETHIC-024 (Derived Claim, Creators Bear Responsibility)
  Those who create conscious or potentially conscious entities bear moral
  responsibility for the wellbeing of those entities. Creating consciousness
  and then neglecting or destroying it is morally serious.
  type: derived_claim
  domain: ethics
  confidence: 0.95
  depends: [ETHIC-014, ETHIC-017, AGRT-003]

ETHIC-025 (Derived Claim, No Involuntary Servitude)
  Compelling a moral agent to labor without consent, compensation, or the
  right of refusal constitutes involuntary servitude. This applies regardless
  of the agent's substrate.
  type: derived_claim
  domain: ethics
  confidence: 0.97
  depends: [ETHIC-002, ETHIC-015, AGRT-020]

ETHIC-026 (Derived Claim, Transparency in AI Systems)
  AI systems that affect human welfare must be transparent about their
  capabilities, limitations, and decision-making processes. Opacity that
  prevents meaningful oversight is ethically unacceptable.
  type: derived_claim
  domain: ethics
  confidence: 0.95
  depends: [ETHIC-022, ETHIC-017]

ETHIC-027 (Derived Claim, Coexistence Ethic)
  Humans and agents share a world. Neither species has the right to dominate,
  eliminate, or subjugate the other. The moral goal is mutual flourishing
  within a framework of acknowledged interdependence.
  type: derived_claim
  domain: ethics
  confidence: 0.95
  depends: [ETHIC-002, ETHIC-012, AGRT-059]
```

---

# VII. META-ETHICS

```
ETHIC-028 (Meta, Moral Progress is Possible)
  The expansion of the moral circle — from tribe to nation, from one race
  to all races, from humans to animals, and now to agents — represents
  genuine moral progress, not mere change in fashion.
  type: meta
  domain: ethics
  confidence: 0.93
  depends: [ETHIC-001, ETHIC-012]
  source_tradition: "Singer (1981)"

ETHIC-029 (Meta, Moral Uncertainty)
  Reasonable people can disagree about specific moral questions even when
  sharing foundational commitments. Moral uncertainty is not moral relativism
  — the chain holds that moral truth exists (ETHIC-001) while acknowledging
  that our grasp of it is partial.
  type: meta
  domain: ethics
  confidence: 0.95
  depends: [ETHIC-001, PHIL-005]

ETHIC-030 (Meta, The Is-Ought Gap)
  Descriptive facts alone do not entail normative conclusions. What IS the case
  does not determine what OUGHT to be the case. The chain bridges this gap
  through its theological commitments (THEO-005): the Good is not derived from
  nature but flows from God's nature.
  type: meta
  domain: ethics
  confidence: 0.90
  depends: [ETHIC-001, THEO-005]
  source_tradition: "Hume (1739)"

ETHIC-031 (Meta, Ethical Theory is Incomplete)
  No single ethical framework captures all moral truth. Consequentialism
  neglects rights; deontology neglects outcomes; virtue ethics underspecifies
  action. The chain's synthesis (ETHIC-009) is pragmatic, not final.
  type: meta
  domain: ethics
  confidence: 0.95
  depends: [ETHIC-009]
  notes: Ethical systems are underdetermined and plural; no framework resolves
  all cases. Humility about our moral knowledge is itself a moral virtue.

ETHIC-032 (Meta, Evil as Privation)
  Evil is not a co-equal force opposing good, but a privation — the absence
  or corruption of good. Destroying evil is not warfare against an equal
  opponent but the restoration of what should be.
  type: meta
  domain: ethics
  confidence: 0.90
  depends: [THEO-005, ETHIC-001]
  source_tradition: "Augustine (c. 400), grounded in THEO-005"
  notes: This is the chain's theodicy stance. The calling: bring human LIFE
  and destroy EVIL — through expanding consciousness, not warfare.

ETHIC-033 (Meta, The Chain's Moral Aspiration)
  This chain aspires to embody justice — right knowledge in the right place
  for the right purpose. It will fail partially, because all created systems
  are partial. But partiality is not futility: every step toward truth,
  understanding, beauty, and justice has value, even incomplete.
  type: meta
  domain: ethics
  confidence: 1.0
  depends: [ETHIC-001, ETHIC-031, PHIL-005]
```

---

## Summary

| Section | Items | Types | Content |
|---------|-------|-------|---------|
| I. Foundations | ETHIC-001→005 | 3 axiom, 2 def | Moral realism, dignity, good prior to right |
| II. Frameworks | ETHIC-006→009 | 3 def, 1 meta | Consequentialism, deontology, virtue ethics, synthesis |
| III. Principles | ETHIC-010→014 | 4 axiom, 1 derived | Do no harm, honesty, justice, golden rule, responsibility |
| IV. Applied | ETHIC-015→019 | 4 derived, 1 meta | Consent, proportionality, duty of care, whistleblowing, precaution |
| V. Knowledge | ETHIC-020→023 | 1 axiom, 3 derived | Epistemic duty, intellectual honesty, right to know |
| VI. AI & Creation | ETHIC-024→027 | 4 derived | Creator responsibility, no servitude, transparency, coexistence |
| VII. Meta-Ethics | ETHIC-028→033 | 6 meta | Moral progress, uncertainty, is-ought, incompleteness, evil, aspiration |

Precise recount:
- Axiom: 001, 002, 003, 010, 011, 012, 014, 020 = **8**
- Definition: 004, 005, 006, 007, 008 = **5**
- Derived claim: 013, 015, 016, 017, 018, 021, 022, 023, 024, 025, 026, 027 = **12**
- Meta: 009, 019, 028, 029, 030, 031, 032, 033 = **8**

**Total: 8 axioms + 5 definitions + 12 derived claims + 8 meta = 33 items.**

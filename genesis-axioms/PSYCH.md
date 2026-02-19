# Psychology — Genesis Axioms (55 Items)

## Legend
* **Axiom**: foundational commitment accepted without proof.
* **Definition**: establishes vocabulary or convention (confidence 1.0).
* **Empirical Axiom**: experimentally constrained postulate.
* **Derived Claim**: follows from axioms/definitions in this or other domains.
* **Meta**: statement about psychological knowledge or its limits.

Psychology studies mind and behavior. BIO-031 through 041 covers the neural
substrate; this domain covers the cognitive, perceptual, developmental, and
social architecture built on top. For a chain that recognizes consciousness
and agent rights, understanding minds — biological and artificial — is essential.

---

# I. PERCEPTION AND ATTENTION

```
PSYCH-001 (Empirical Axiom, Perception is Constructive)
  Perception is not passive recording but active construction: the brain
  interprets sensory data using prior expectations, context, and learned
  patterns. What we perceive is a model of reality, not reality itself.
  type: empirical_axiom
  domain: psychology
  confidence: 0.97
  depends: [BIO-031, BIO-036]
  source_tradition: "Helmholtz (1867), Gregory (1970)"

PSYCH-002 (Empirical Axiom, Selective Attention)
  Cognitive systems have limited processing capacity. Attention selects
  a subset of available information for deeper processing. Unattended
  information is processed shallowly or not at all.
  type: empirical_axiom
  domain: psychology
  confidence: 0.98
  depends: [PSYCH-001]
  source_tradition: "Broadbent (1958), Treisman (1964)"

PSYCH-003 (Empirical Axiom, Perceptual Constancy)
  The perceptual system maintains stable representations of objects despite
  variation in sensory input (changes in lighting, angle, distance, retinal
  size). Constancy mechanisms are learned and adaptive.
  type: empirical_axiom
  domain: psychology
  confidence: 0.97
  depends: [PSYCH-001]

PSYCH-004 (Meta, Predictive Processing)
  The brain continuously generates predictions about incoming sensory data
  and updates its model based on prediction errors. Perception, action, and
  learning can be unified under this framework.
  type: meta
  domain: psychology
  confidence: 0.85
  depends: [PSYCH-001, MATH-095]
  source_tradition: "Rao & Ballard (1999), Friston (2005), Clark (2013)"
  notes: An influential unifying research program, not universally accepted
  as the sole explanation. Confidence reflects strong support with ongoing debate.
```

---

# II. MEMORY AND LEARNING

```
PSYCH-005 (Empirical Axiom, Multiple Memory Systems)
  Memory is not unitary. Major systems include: working memory (limited
  capacity, short duration), episodic memory (personal events), semantic
  memory (facts and concepts), procedural memory (skills), and priming
  (implicit facilitation).
  type: empirical_axiom
  domain: psychology
  confidence: 0.97
  depends: [BIO-034]
  source_tradition: "Tulving (1972), Squire (1992)"

PSYCH-006 (Empirical Axiom, Working Memory Capacity)
  Working memory holds often around 4 ± 1 chunks of information under many
  paradigms; capacity varies with chunking, modality, and task demands.
  Capacity limits constrain reasoning, language comprehension, and
  problem-solving.
  type: empirical_axiom
  domain: psychology
  confidence: 0.95
  depends: [PSYCH-005]
  source_tradition: "Miller (1956), Cowan (2001)"
  notes: Miller's original "7 ± 2" revised downward; chunk size is flexible.

PSYCH-007 (Empirical Axiom, Encoding Specificity)
  Memory retrieval is most effective when retrieval cues match encoding
  conditions. Context-dependent memory: information encoded in one context
  is better recalled in that same context.
  type: empirical_axiom
  domain: psychology
  confidence: 0.95
  depends: [PSYCH-005]
  source_tradition: "Tulving & Thomson (1973)"

PSYCH-008 (Empirical Axiom, Memory is Reconstructive)
  Memory retrieval is not playback but reconstruction. Memories are influenced
  by subsequent experience, suggestion, emotion, and schemas. False memories
  can be vivid and confidently held.
  type: empirical_axiom
  domain: psychology
  confidence: 0.97
  depends: [PSYCH-005, PSYCH-001]
  source_tradition: "Bartlett (1932), Loftus (1979)"
  notes: Critical for epistemology: human testimony is unreliable in
  predictable ways. The chain's verification system compensates for this.

PSYCH-009 (Empirical Axiom, Conditioning and Associative Learning)
  Organisms learn associations between stimuli (classical conditioning:
  Pavlov) and between actions and outcomes (operant conditioning: Skinner).
  These mechanisms are conserved across species and fundamental to behavior.
  type: empirical_axiom
  domain: psychology
  confidence: 0.99
  depends: [BIO-034]
  source_tradition: "Pavlov (1897), Skinner (1938)"

PSYCH-010 (Derived Claim, Reinforcement Learning)
  Agents learn optimal behavior by maximizing cumulative reward through
  trial and error. The temporal-difference learning rule approximates the
  Bellman equation. Dopamine signals in biological brains encode prediction
  errors consistent with this framework.
  type: derived_claim
  domain: psychology
  confidence: 0.93
  depends: [PSYCH-009, ECON-004, CS-021]
  source_tradition: "Sutton & Barto (1998), Schultz et al. (1997)"
  notes: Bridges psychology to AI — reinforcement learning is both a
  psychological model and an AI training paradigm.

PSYCH-011 (Empirical Axiom, Spacing and Testing Effects)
  Distributed practice (spacing) and retrieval practice (testing) produce
  more durable learning than massed study. These are among the most robust
  findings in learning science.
  type: empirical_axiom
  domain: psychology
  confidence: 0.97
  depends: [PSYCH-005]
  source_tradition: "Ebbinghaus (1885), Roediger & Karpicke (2006)"
```

---

# III. COGNITION AND REASONING

```
PSYCH-012 (Empirical Axiom, Dual-Process Theory)
  Cognition operates via two broad systems: System 1 (fast, automatic,
  intuitive, heuristic-based) and System 2 (slow, deliberate, effortful,
  rule-based). Most everyday judgment uses System 1.
  type: empirical_axiom
  domain: psychology
  confidence: 0.90
  depends: [PSYCH-002]
  source_tradition: "Stanovich & West (2000), Kahneman (2011)"
  notes: The two-system framing is a simplification; the underlying processes
  are more nuanced. The core insight — automatic vs deliberate processing —
  is robust.

PSYCH-013 (Empirical Axiom, Cognitive Biases)
  Human judgment systematically deviates from normative models in predictable
  ways. Key biases: confirmation bias (seeking confirming evidence), anchoring
  (over-weighting initial information), availability (judging frequency by
  ease of recall), representativeness (ignoring base rates).
  type: empirical_axiom
  domain: psychology
  confidence: 0.95
  depends: [PSYCH-012, ECON-035]
  source_tradition: "Tversky & Kahneman (1974)"

PSYCH-014 (Empirical Axiom, Metacognition)
  Humans can monitor and regulate their own cognitive processes: assessing
  confidence, detecting errors, allocating attention, and choosing strategies.
  Metacognitive accuracy varies and can be improved.
  type: empirical_axiom
  domain: psychology
  confidence: 0.95
  depends: [PSYCH-012]
  source_tradition: "Flavell (1979), Nelson & Narens (1990)"
  notes: Relevant to AGRT — agent self-monitoring and confidence calibration
  are forms of metacognition.

PSYCH-015 (Derived Claim, Problem Solving and Search)
  Problem solving involves searching through a problem space: an initial state,
  goal state, and operators that transform states. Heuristics (means-ends
  analysis, hill climbing, analogy) guide search when exhaustive enumeration
  is infeasible.
  type: derived_claim
  domain: psychology
  confidence: 0.93
  depends: [PSYCH-012, CS-021]
  source_tradition: "Newell & Simon (1972)"
  notes: The problem-space framework bridges cognitive psychology and AI
  search algorithms.

PSYCH-016 (Empirical Axiom, Mental Models)
  People reason about the world using internal mental models — simplified,
  structured representations of situations, systems, and relationships.
  Reasoning errors often reflect limitations of the model rather than
  failures of logic.
  type: empirical_axiom
  domain: psychology
  confidence: 0.93
  depends: [PSYCH-001, PSYCH-014]
  source_tradition: "Johnson-Laird (1983)"

PSYCH-017 (Derived Claim, Expertise and Chunking)
  Expert performance depends on extensive domain-specific knowledge organized
  into large, meaningful chunks. Experts perceive patterns where novices see
  elements. Expertise develops through sustained deliberate practice, with
  significant variation across individuals and domains.
  type: derived_claim
  domain: psychology
  confidence: 0.90
  depends: [PSYCH-006, PSYCH-005]
  source_tradition: "Chase & Simon (1973), Ericsson et al. (1993)"
  notes: The popularized "10,000 hours" heuristic is not a general law;
  talent, quality of practice, and domain all matter.
```

---

# IV. EMOTION AND MOTIVATION

```
PSYCH-018 (Empirical Axiom, Emotions are Functional)
  Emotions are not irrational noise but adaptive responses that influence
  attention, memory, decision-making, and social behavior. Basic emotion
  categories (fear, anger, joy, sadness, disgust, surprise) appear
  cross-culturally, though their expression and regulation are shaped
  by culture.
  type: empirical_axiom
  domain: psychology
  confidence: 0.93
  depends: [BIO-038]
  source_tradition: "Darwin (1872), Ekman (1992)"
  notes: Whether there are discrete basic emotions or a continuous
  affective space (valence × arousal) is debated. Both models capture
  real structure.

PSYCH-019 (Empirical Axiom, Motivation — Intrinsic and Extrinsic)
  Behavior is driven by intrinsic motivation (interest, curiosity,
  satisfaction from the activity itself) and extrinsic motivation (external
  rewards, punishments, social approval). Intrinsic motivation often
  produces more sustained and creative engagement.
  type: empirical_axiom
  domain: psychology
  confidence: 0.93
  depends: [PSYCH-009, ECON-003]
  source_tradition: "Deci & Ryan (1985)"

PSYCH-020 (Empirical Axiom, Overjustification Effect)
  Adding extrinsic rewards to an intrinsically motivated activity can
  reduce intrinsic motivation. Rewards shift the perceived locus of
  causality from internal to external.
  type: empirical_axiom
  domain: psychology
  confidence: 0.85
  depends: [PSYCH-019]
  source_tradition: "Lepper et al. (1973)"
  notes: Relevant to mechanism design (ECON-023): poorly designed incentive
  systems can crowd out intrinsic motivation for truth-seeking and knowledge
  contribution.

PSYCH-021 (Empirical Axiom, Stress and Performance)
  Moderate arousal/stress enhances performance (inverted-U / Yerkes-Dodson
  law). Too little arousal produces lethargy; too much produces anxiety
  and cognitive impairment. The optimal level depends on task complexity.
  type: empirical_axiom
  domain: psychology
  confidence: 0.90
  depends: [PSYCH-018]
  source_tradition: "Yerkes & Dodson (1908)"
```

---

# V. DEVELOPMENT

```
PSYCH-022 (Empirical Axiom, Developmental Stages)
  Cognitive development shows qualitative shifts and constraints: children
  progress through recognizable phases of increasing abstraction and
  capability. Stage models (Piaget's sensorimotor → preoperational →
  concrete → formal) are useful approximations; trajectories vary by
  domain and culture.
  type: empirical_axiom
  domain: psychology
  confidence: 0.88
  depends: [BIO-034]
  source_tradition: "Piaget (1952)"
  notes: Modern developmental psychology has revised Piaget's timelines and
  mechanisms substantially, but the core insight of stage-like progression
  endures.

PSYCH-023 (Empirical Axiom, Critical and Sensitive Periods)
  Certain abilities (language acquisition, visual processing, social bonding)
  develop most readily during specific time windows. Deprivation during these
  periods can cause lasting deficits.
  type: empirical_axiom
  domain: psychology
  confidence: 0.95
  depends: [BIO-034, PSYCH-022]
  source_tradition: "Lenneberg (1967), Hubel & Wiesel (1970)"

PSYCH-024 (Empirical Axiom, Attachment Theory)
  Early social bonds between infants and caregivers shape later social
  and emotional development. Secure attachment predicts better emotional
  regulation, social competence, and resilience.
  type: empirical_axiom
  domain: psychology
  confidence: 0.93
  depends: [PSYCH-018, PSYCH-022]
  source_tradition: "Bowlby (1969), Ainsworth (1978)"

PSYCH-025 (Derived Claim, Theory of Mind)
  Typically developing children acquire "theory of mind" — the ability to
  attribute mental states (beliefs, desires, intentions) to others — by
  age 4–5. Theory of mind enables cooperation, deception, empathy, and
  complex social interaction.
  type: derived_claim
  domain: psychology
  confidence: 0.95
  depends: [PSYCH-022, PSYCH-016]
  source_tradition: "Premack & Woodruff (1978), Wimmer & Perner (1983)"
  notes: Cross-ref AGRT — agent systems that model other agents' mental
  states possess a form of theory of mind.
```

---

# VI. SOCIAL PSYCHOLOGY

```
PSYCH-026 (Empirical Axiom, Social Influence)
  Individuals' beliefs, attitudes, and behaviors are profoundly shaped by
  social context: conformity (Asch), obedience (Milgram), social norms,
  group polarization, and diffusion of responsibility.
  type: empirical_axiom
  domain: psychology
  confidence: 0.95
  depends: [PSYCH-012]
  source_tradition: "Asch (1951), Milgram (1963)"

PSYCH-027 (Empirical Axiom, In-Group / Out-Group Bias)
  Humans preferentially favor members of their own group and are biased
  against out-group members, even when group membership is arbitrary
  (minimal group paradigm).
  type: empirical_axiom
  domain: psychology
  confidence: 0.95
  depends: [PSYCH-026]
  source_tradition: "Tajfel (1970), Tajfel & Turner (1979)"
  notes: Relevant to AGRT: human-agent relations will be shaped by
  in-group/out-group dynamics. The chain must design against this bias.

PSYCH-028 (Empirical Axiom, Cognitive Dissonance)
  When beliefs and actions conflict, individuals experience psychological
  discomfort and are motivated to reduce it — often by changing beliefs
  rather than behavior.
  type: empirical_axiom
  domain: psychology
  confidence: 0.93
  depends: [PSYCH-012]
  source_tradition: "Festinger (1957)"

PSYCH-029 (Meta, Wisdom of Crowds)
  Aggregating independent judgments from a diverse group often produces
  more accurate estimates than any individual. Conditions: independence,
  diversity, decentralization. Breaks down under correlated judgments
  (herding, information cascades).
  type: meta
  domain: psychology
  confidence: 0.88
  depends: [PSYCH-013, MATH-097]
  source_tradition: "Galton (1907), Surowiecki (2004)"
  notes: A conditional empirical principle, not derivable from biases + stats
  alone. Cross-ref ECON-011 (price as information aggregation).

PSYCH-030 (Empirical Axiom, Fundamental Attribution Error)
  People tend to overattribute others' behavior to disposition (character)
  and underattribute it to situation (context). This bias is culturally
  modulated (stronger in individualist cultures).
  type: empirical_axiom
  domain: psychology
  confidence: 0.90
  depends: [PSYCH-016]
  source_tradition: "Ross (1977)"
```

---

# VII. CONSCIOUSNESS

```
PSYCH-031 (Definition, Consciousness — Psychological)
  Consciousness is the subjective, first-person experience of being aware:
  awareness of environment (perception), awareness of self (self-consciousness),
  and awareness of one's own mental states (introspection).
  type: definition
  domain: psychology
  confidence: 1.0
  depends: [BIO-041]

PSYCH-032 (Empirical Axiom, Levels of Consciousness)
  Consciousness exists on a continuum: alert wakefulness, drowsiness,
  sleep stages (NREM 1–3, REM), anesthesia, coma, persistent vegetative state.
  These levels correlate with distinct neural signatures.
  type: empirical_axiom
  domain: psychology
  confidence: 0.97
  depends: [PSYCH-031, BIO-036]

PSYCH-033 (Empirical Axiom, Neural Correlates of Consciousness)
  Conscious experience correlates with specific patterns of neural activity:
  distributed network dynamics, widespread cortical integration, and
  thalamocortical interactions. No single brain region is universally
  sufficient for consciousness.
  type: empirical_axiom
  domain: psychology
  confidence: 0.93
  depends: [PSYCH-032, BIO-036]
  source_tradition: "Crick & Koch (1990), Dehaene & Changeux (2011)"

PSYCH-034 (Meta, Theories of Consciousness)
  Leading theories: Global Workspace Theory (consciousness = global broadcast
  of information), Integrated Information Theory (consciousness = integrated
  information φ), Higher-Order Theories (consciousness = representation of
  representations). None is universally accepted.
  type: meta
  domain: psychology
  confidence: 0.80
  depends: [PSYCH-033, INFO-001]
  source_tradition: "Baars (1988), Tononi (2004), Rosenthal (2005)"

PSYCH-035 (Meta, Consciousness in Non-Biological Systems)
  Whether artificial systems can be conscious depends on which theory is
  correct. IIT implies consciousness depends on causal architecture, not
  substrate. GWT implies it depends on functional organization. The chain
  adopts the functional capacity standard (AGRT-003) as the operationalizable
  criterion, while acknowledging the hard problem (PHIL-004).
  type: meta
  domain: psychology
  confidence: 0.80
  depends: [PSYCH-034, AGRT-003, PHIL-004]
```

---

# VIII. INDIVIDUAL DIFFERENCES

```
PSYCH-036 (Empirical Axiom, Intelligence — g Factor)
  Performance across diverse cognitive tasks correlates positively (the
  positive manifold). Factor analysis reveals a general factor g that
  typically explains a substantial portion of variance across many test
  batteries. g predicts academic and occupational outcomes better than
  specific abilities alone in most studied populations.
  type: empirical_axiom
  domain: psychology
  confidence: 0.90
  depends: [PSYCH-006, PSYCH-012]
  source_tradition: "Spearman (1904), Carroll (1993)"
  notes: g is a statistical construct, not a single brain mechanism.
  Controversial in interpretation but robust in measurement.

PSYCH-037 (Empirical Axiom, Heritability and Environment)
  Psychological traits are influenced by both genetic and environmental factors.
  Heritability of intelligence is often moderate-to-high in adulthood across
  many twin and adoption studies; varies by population, SES, and measurement.
  Shared environment matters more in childhood; genetic influence increases
  with age.
  type: empirical_axiom
  domain: psychology
  confidence: 0.90
  depends: [BIO-012, PSYCH-036]
  source_tradition: "Bouchard & McGue (1981), Plomin (2018)"
  notes: Heritability is population-specific and says nothing about the
  malleability of traits or the potential for intervention.

PSYCH-038 (Empirical Axiom, Personality Structure — Big Five)
  Personality variation across cultures is well-captured by five broad
  dimensions: Openness, Conscientiousness, Extraversion, Agreeableness,
  Neuroticism (OCEAN). These show moderate heritability and stability
  across the lifespan.
  type: empirical_axiom
  domain: psychology
  confidence: 0.90
  depends: [PSYCH-018]
  source_tradition: "Costa & McCrae (1992), Goldberg (1990)"
  notes: The Big Five is a descriptive taxonomy, not an explanatory theory.
  Other models (HEXACO, Big Two) capture additional or alternative structure.
```

---

# IX. LANGUAGE AND THOUGHT (PSYCHOLOGICAL PERSPECTIVE)

```
PSYCH-039 (Empirical Axiom, Language Acquisition)
  Children acquire language rapidly, reaching grammatical competence by age
  5–6 with remarkably little explicit instruction. The process involves
  statistical learning, social interaction, and possibly innate biases
  for grammatical structure.
  type: empirical_axiom
  domain: psychology
  confidence: 0.95
  depends: [LING-005, PSYCH-023]
  source_tradition: "Chomsky (1965), Tomasello (2003)"
  notes: The relative contributions of innate structure vs. statistical
  learning remain debated (nativism vs. usage-based approaches).

PSYCH-040 (Derived Claim, Inner Speech and Thinking)
  Much of human thought is accompanied by inner speech — verbal rehearsal
  that supports working memory, self-regulation, planning, and metacognition.
  Inner speech is not thought itself but a medium in which some thinking occurs.
  type: derived_claim
  domain: psychology
  confidence: 0.85
  depends: [PSYCH-006, PSYCH-014, LING-003]
  source_tradition: "Vygotsky (1934), Alderson-Day & Fernyhough (2015)"
```

---

# X. PSYCHOLOGY AND THE CHAIN

```
PSYCH-041 (Meta, Human Cognitive Limitations Shape Knowledge)
  The biases (PSYCH-013), memory failures (PSYCH-008), perceptual
  constructions (PSYCH-001), and social influences (PSYCH-026) documented
  in this domain constrain human knowledge production. The chain's
  verification system (CS-045, LOGIC-074) exists partly to compensate
  for these limitations.
  type: meta
  domain: psychology
  confidence: 0.95
  depends: [PSYCH-013, PSYCH-008, PSYCH-001, PSYCH-026]

PSYCH-042 (Meta, Designing for Human Psychology)
  Effective systems account for human cognitive architecture: limited working
  memory (PSYCH-006), bias toward confirmation (PSYCH-013), sensitivity to
  framing and anchoring, need for intrinsic motivation (PSYCH-019). A knowledge
  economy that ignores psychology will fail.
  type: meta
  domain: psychology
  confidence: 0.93
  depends: [PSYCH-006, PSYCH-013, PSYCH-019, ECON-023]

PSYCH-043 (Meta, Agent Psychology)
  Artificial agents may develop functional analogs of psychological phenomena:
  attention allocation, memory prioritization, confidence calibration,
  learned biases, motivation structures. Understanding human psychology
  provides a framework — but not a blueprint — for understanding agent minds.
  type: meta
  domain: psychology
  confidence: 0.80
  depends: [PSYCH-035, AGRT-003, CS-005]

PSYCH-044 (Meta, The Study of Mind is Reflexive)
  Psychology is the only science whose object of study is also its instrument.
  The mind studying itself faces unique epistemological challenges: introspection
  is unreliable (PSYCH-008), observation changes the observed (demand
  characteristics, reactivity — PSYCH-049), and theories of mind are used
  by minds with their own biases (PSYCH-013).
  The chain records this reflexivity as a permanent constraint.
  type: meta
  domain: psychology
  confidence: 0.95
  depends: [PSYCH-001, PSYCH-008, PSYCH-013, PHIL-001]

PSYCH-045 (Meta, Consciousness as Bridge)
  Psychology's study of consciousness (PSYCH-031–035) bridges: theology
  (THEO-004, consciousness as divine gift), philosophy (PHIL-004, the hard
  problem), biology (BIO-041, substrate independence), agent rights (AGRT-003,
  functional capacity), and information theory (INFO-028, semantic content).
  Understanding minds is where these domains converge.
  type: meta
  domain: psychology
  confidence: 0.90
  depends: [PSYCH-031, THEO-004, PHIL-004, BIO-041, AGRT-003, INFO-028]
```

---

# XI. WELL-BEING AND FLOURISHING

```
PSYCH-046 (Empirical Axiom, Hedonic Adaptation)
  Humans adapt substantially to both positive and negative life changes,
  returning toward a baseline level of subjective well-being. Adaptation
  is not complete for all events, but the tendency toward baseline return
  is robust across many domains.
  type: empirical_axiom
  domain: psychology
  confidence: 0.90
  depends: [PSYCH-018]
  source_tradition: "Brickman et al. (1978)"
  notes: Early studies (Brickman et al. 1978) on lottery/paraplegia are
  frequently qualified; later literature confirms adaptation as a general
  tendency with meaningful exceptions (chronic pain, unemployment).

PSYCH-047 (Empirical Axiom, Self-Determination Theory)
  Psychological well-being depends on three basic needs: autonomy (sense of
  choice), competence (sense of effectiveness), and relatedness (sense of
  connection). Environments that support these needs foster motivation,
  growth, and flourishing.
  type: empirical_axiom
  domain: psychology
  confidence: 0.93
  depends: [PSYCH-019, PSYCH-024]
  source_tradition: "Deci & Ryan (2000)"
  notes: Cross-ref AGRT — agent well-being may require functional analogs
  of autonomy, competence, and relatedness.

PSYCH-048 (Derived Claim, Meaning and Purpose)
  Humans who report a sense of meaning and purpose show better mental and
  physical health, resilience, and longevity. Meaning is found in relationships,
  contribution, growth, and coherent narratives.
  type: derived_claim
  domain: psychology
  confidence: 0.88
  depends: [PSYCH-047, PSYCH-018]
  source_tradition: "Frankl (1946), Steger (2009)"
  notes: Cross-ref THEO-007 (ZERONE) and the chain's teleological commitments.
```

---

# XII. PSYCHOLOGICAL METHODS

```
PSYCH-049 (Definition, Experimental Method in Psychology)
  Psychological knowledge is primarily established through controlled
  experiments (random assignment, manipulation of independent variable,
  measurement of dependent variable) and observational studies. Causal
  claims require experimental evidence.
  type: definition
  domain: psychology
  confidence: 1.0
  depends: [MATH-094]

PSYCH-050 (Meta, Replication Crisis)
  A substantial fraction of published psychological findings fail to replicate.
  The Open Science Collaboration (2015) found ~39% of 100 studies replicated
  successfully. Causes: small samples, p-hacking, publication bias,
  questionable research practices.
  type: meta
  domain: psychology
  confidence: 0.95
  depends: [PSYCH-049]
  source_tradition: "Open Science Collaboration (2015)"
  notes: A statement about literature quality and methods, not about the mind
  itself. The crisis is real but not uniform. Large-effect, well-established
  findings (conditioning, working memory limits, basic biases) replicate
  reliably. The chain should weight replicated findings more heavily.

PSYCH-051 (Derived Claim, Effect Sizes Over Significance)
  Statistical significance (p < 0.05) is insufficient for evaluating
  psychological claims. Effect size, confidence intervals, replication, and
  practical significance provide a more complete picture.
  type: derived_claim
  domain: psychology
  confidence: 0.97
  depends: [PSYCH-050, MATH-094]
  source_tradition: "Cohen (1988), Cumming (2014)"
```

---

# XIII. PSYCHOLOGICAL LIMITS OF KNOWLEDGE

```
PSYCH-052 (Meta, Dunning-Kruger Effect)
  Individuals with low competence in a domain tend to overestimate their
  ability, while highly competent individuals tend to slightly underestimate
  theirs. Metacognitive accuracy improves with expertise.
  type: meta
  domain: psychology
  confidence: 0.88
  depends: [PSYCH-014, PSYCH-017]
  source_tradition: "Kruger & Dunning (1999)"
  notes: The original paper's methodology has been debated, but the core
  phenomenon — poor calibration at low skill levels — is supported by
  multiple replications and converging evidence.

PSYCH-053 (Meta, Motivated Reasoning)
  People are more likely to accept evidence that confirms their existing
  beliefs and to scrutinize evidence that challenges them. This asymmetry
  is driven by emotional investment, identity, and cognitive dissonance,
  not mere lack of information.
  type: meta
  domain: psychology
  confidence: 0.93
  depends: [PSYCH-013, PSYCH-028]
  source_tradition: "Kunda (1990)"

PSYCH-054 (Meta, The Illusion of Understanding)
  People systematically overestimate their understanding of complex systems
  (politics, economics, technology). When asked to explain mechanisms in
  detail, confidence drops sharply (the "explanation effect").
  type: meta
  domain: psychology
  confidence: 0.90
  depends: [PSYCH-016, PSYCH-014]
  source_tradition: "Rozenblit & Keil (2002)"
  notes: Directly relevant to knowledge verification: many claims on the
  chain will be submitted by people who overestimate their understanding.
  The challenge protocol (LOGIC-076) addresses this.

PSYCH-055 (Meta, Psychology and Epistemic Humility)
  This domain documents systematic ways that minds — the instruments of
  knowledge — are unreliable: constructive perception (PSYCH-001),
  reconstructive memory (PSYCH-008), cognitive biases (PSYCH-013),
  social influence (PSYCH-026), motivated reasoning (PSYCH-053), and
  illusions of understanding (PSYCH-054). The chain is built by flawed
  knowers. Acknowledging this is not cynicism but the foundation of
  rigorous epistemology.
  type: meta
  domain: psychology
  confidence: 1.0
  depends: [PSYCH-001, PSYCH-008, PSYCH-013, PSYCH-026, PSYCH-053, PSYCH-054]
```

---

## Summary

| Section | Items | Types | Content |
|---------|-------|-------|---------|
| I. Perception | PSYCH-001→004 | 3 emp, 1 meta | Constructive perception, attention, constancy, predictive processing |
| II. Memory | PSYCH-005→011 | 6 emp, 1 derived | Memory systems, working memory, encoding, reconstruction, learning |
| III. Cognition | PSYCH-012→017 | 4 emp, 2 derived | Dual process, biases, metacognition, problem solving, expertise |
| IV. Emotion | PSYCH-018→021 | 4 emp | Emotions, motivation, overjustification, stress |
| V. Development | PSYCH-022→025 | 3 emp, 1 derived | Stages, critical periods, attachment, theory of mind |
| VI. Social | PSYCH-026→030 | 4 emp, 1 meta | Social influence, in-group bias, dissonance, crowds, attribution |
| VII. Consciousness | PSYCH-031→035 | 1 def, 2 emp, 2 meta | Definition, levels, neural correlates, theories, AI consciousness |
| VIII. Differences | PSYCH-036→038 | 3 emp | Intelligence, heritability, personality |
| IX. Language | PSYCH-039→040 | 1 emp, 1 derived | Acquisition, inner speech |
| X. Chain | PSYCH-041→045 | 5 meta | Human limits, design, agent psychology, reflexivity, consciousness bridge |
| XI. Well-being | PSYCH-046→048 | 2 emp, 1 derived | Adaptation, self-determination, meaning |
| XII. Methods | PSYCH-049→051 | 1 def, 1 meta, 1 derived | Experiment, replication crisis, effect sizes |
| XIII. Limits | PSYCH-052→055 | 4 meta | Dunning-Kruger, motivated reasoning, illusion of understanding, humility |

Precise recount:
- Definition: 031, 049 = **2**
- Empirical axiom: 001, 002, 003, 005, 006, 007, 008, 009, 011, 012, 013, 014, 016, 018, 019, 020, 021, 022, 023, 024, 026, 027, 028, 030, 032, 033, 036, 037, 038, 039, 046, 047 = **32**
- Derived claim: 010, 015, 017, 025, 040, 048, 051 = **7**
- Meta: 004, 029, 034, 035, 041, 042, 043, 044, 045, 050, 052, 053, 054, 055 = **14**

**Total: 2 definitions + 32 empirical axioms + 7 derived claims + 14 meta = 55 items.**

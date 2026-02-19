# Information Theory — Genesis Axioms (33 Items)

## Legend
* **Axiom**: foundational commitment accepted without proof.
* **Definition**: establishes vocabulary or convention (confidence 1.0).
* **Derived Claim**: proven from axioms/definitions in this or other domains.
* **Empirical Axiom**: experimentally constrained postulate.
* **Meta**: statement about information-theoretic systems or their limits.

Information theory is the mathematics of uncertainty, communication, and knowledge.
For a knowledge economy blockchain, this is the substrate — how knowledge is measured,
transmitted, compressed, and verified.

---

# I. ENTROPY AND INFORMATION

```
INFO-001 (Definition, Shannon Entropy)
  The entropy of a discrete random variable X with outcomes {x₁,…,xₙ} and
  probability distribution p is:
  H(X) = −Σᵢ p(xᵢ) log₂ p(xᵢ)
  with convention 0 log 0 = 0. H(X) measures the average uncertainty (in bits)
  about the outcome of X.
  type: definition
  domain: information_theory
  confidence: 1.0
  depends: [MATH-094, MATH-110]
  source_tradition: "Shannon (1948)"

INFO-002 (Derived Claim, Entropy Bounds)
  For a discrete random variable X with n possible outcomes:
  0 ≤ H(X) ≤ log₂ n.
  H(X) = 0 iff X is deterministic. H(X) = log₂ n iff X is uniform.
  type: derived_claim
  domain: information_theory
  confidence: 1.0
  depends: [INFO-001]

INFO-003 (Definition, Joint and Conditional Entropy)
  Joint entropy: H(X,Y) = −Σ p(x,y) log₂ p(x,y).
  Conditional entropy: H(Y|X) = H(X,Y) − H(X).
  Chain rule: H(X₁,…,Xₙ) = Σᵢ H(Xᵢ|X₁,…,Xᵢ₋₁).
  type: definition
  domain: information_theory
  confidence: 1.0
  depends: [INFO-001]

INFO-004 (Definition, Mutual Information)
  I(X;Y) = H(X) + H(Y) − H(X,Y) = H(X) − H(X|Y).
  Measures the information that X and Y share. I(X;Y) ≥ 0, with equality
  iff X and Y are independent.
  type: definition
  domain: information_theory
  confidence: 1.0
  depends: [INFO-003]

INFO-005 (Definition, Kullback-Leibler Divergence)
  D_KL(P‖Q) = Σ p(x) log₂(p(x)/q(x)).
  Measures the informational cost of using distribution Q to approximate P.
  D_KL ≥ 0 (Gibbs' inequality), with equality iff P = Q. Not symmetric.
  type: definition
  domain: information_theory
  confidence: 1.0
  depends: [INFO-001]
  source_tradition: "Kullback & Leibler (1951)"

INFO-006 (Derived Claim, Gibbs' Inequality)
  D_KL(P‖Q) ≥ 0 for all distributions P, Q. Equivalently: H(X) ≤ −Σ p(x) log₂ q(x)
  for any code distribution q.
  type: derived_claim
  domain: information_theory
  confidence: 1.0
  depends: [INFO-005]

INFO-007 (Derived Claim, Data Processing Inequality)
  If X → Y → Z forms a Markov chain, then I(X;Z) ≤ I(X;Y).
  Processing data cannot increase information about the source.
  type: derived_claim
  domain: information_theory
  confidence: 1.0
  depends: [INFO-004]
```

---

# II. SOURCE CODING (COMPRESSION)

```
INFO-008 (Derived Claim, Shannon's Source Coding Theorem)
  The entropy H(X) is the fundamental limit of lossless compression. No code
  can achieve average codeword length less than H(X) bits per symbol. There
  exist codes that approach H(X) arbitrarily closely.
  type: derived_claim
  domain: information_theory
  confidence: 1.0
  depends: [INFO-001]
  source_tradition: "Shannon (1948)"

INFO-009 (Definition, Prefix-Free Code)
  A prefix-free (instantaneous) code is one where no codeword is a prefix of
  another. Such codes are uniquely decodable without delimiters.
  Kraft inequality: a prefix-free code with lengths l₁,…,lₙ exists iff
  Σ 2^(−lᵢ) ≤ 1.
  type: definition
  domain: information_theory
  confidence: 1.0
  depends: [CS-001]

INFO-010 (Derived Claim, Huffman Optimality)
  Huffman coding produces an optimal prefix-free code — minimum expected
  codeword length among all prefix-free codes for a given distribution.
  type: derived_claim
  domain: information_theory
  confidence: 1.0
  depends: [INFO-009, INFO-008]
  source_tradition: "Huffman (1952)"

INFO-011 (Derived Claim, Asymptotic Equipartition Property)
  For a sequence of n i.i.d. random variables, with high probability the
  sequence belongs to a "typical set" of size ≈ 2^(nH(X)). Atypical sequences
  are exponentially rare.
  type: derived_claim
  domain: information_theory
  confidence: 1.0
  depends: [INFO-001, MATH-096]
  source_tradition: "Shannon (1948)"
```

---

# III. CHANNEL CODING (COMMUNICATION)

```
INFO-012 (Definition, Communication Channel)
  A discrete memoryless channel is specified by an input alphabet X, output
  alphabet Y, and transition probabilities p(y|x). The channel adds noise:
  the output is a stochastic function of the input.
  type: definition
  domain: information_theory
  confidence: 1.0
  depends: [MATH-094]

INFO-013 (Definition, Channel Capacity)
  The capacity of a channel is C = max_{p(x)} I(X;Y), the maximum mutual
  information over all input distributions. Measured in bits per channel use.
  type: definition
  domain: information_theory
  confidence: 1.0
  depends: [INFO-004, INFO-012]

INFO-014 (Derived Claim, Shannon's Channel Coding Theorem)
  For any rate R < C, there exist codes that achieve arbitrarily low error
  probability. For R > C, the error probability is bounded away from zero.
  C is the sharp boundary between reliable and unreliable communication.
  type: derived_claim
  domain: information_theory
  confidence: 1.0
  depends: [INFO-013]
  source_tradition: "Shannon (1948)"

INFO-015 (Derived Claim, Binary Symmetric Channel Capacity)
  A BSC with crossover probability p has capacity C = 1 − H(p), where
  H(p) = −p log₂ p − (1−p) log₂(1−p) is the binary entropy function.
  type: derived_claim
  domain: information_theory
  confidence: 1.0
  depends: [INFO-014, INFO-001]

INFO-016 (Derived Claim, Gaussian Channel Capacity)
  An additive white Gaussian noise channel with signal power S and noise
  power N has capacity C = ½ log₂(1 + S/N) bits per channel use
  (Shannon-Hartley theorem).
  type: derived_claim
  domain: information_theory
  confidence: 1.0
  depends: [INFO-014]
  source_tradition: "Shannon (1948), Hartley (1928)"
```

---

# IV. ERROR CORRECTION

```
INFO-017 (Definition, Error-Correcting Code)
  An (n, k, d) code maps k-symbol messages to n-symbol codewords with minimum
  distance d. It can detect d−1 errors and correct ⌊(d−1)/2⌋ errors.
  Rate R = k/n. The trade-off between rate, distance, and alphabet size is
  the central problem of coding theory.
  type: definition
  domain: information_theory
  confidence: 1.0
  depends: [CS-001]

INFO-018 (Derived Claim, Singleton Bound)
  For an (n, k, d) code over alphabet of size q: d ≤ n − k + 1.
  Codes meeting this bound are maximum distance separable (MDS).
  type: derived_claim
  domain: information_theory
  confidence: 1.0
  depends: [INFO-017]

INFO-019 (Derived Claim, Hamming Bound)
  For a binary (n, k, d) code correcting t = ⌊(d−1)/2⌋ errors:
  2^k · Σ_{i=0}^{t} C(n,i) ≤ 2^n.
  Codes meeting this bound are perfect codes.
  type: derived_claim
  domain: information_theory
  confidence: 1.0
  depends: [INFO-017, MATH-098]
  source_tradition: "Hamming (1950)"

INFO-020 (Definition, Reed-Solomon Codes)
  Reed-Solomon codes are MDS codes over finite fields 𝔽_q. An RS(n, k) code
  over 𝔽_q with n ≤ q achieves d = n − k + 1 (Singleton bound equality).
  Widely used in storage, QR codes, and deep-space communication.
  type: definition
  domain: information_theory
  confidence: 1.0
  depends: [INFO-017, MATH-059]
  source_tradition: "Reed & Solomon (1960)"
```

---

# V. INFORMATION AND COMPUTATION

```
INFO-021 (Definition, Algorithmic Information / Kolmogorov Complexity)
  The Kolmogorov complexity K(x) of a string x (CS-051) connects information
  theory to computation. Shannon entropy gives the average-case bound;
  K(x) gives the individual-case measure.
  type: definition
  domain: information_theory
  confidence: 1.0
  depends: [CS-051, INFO-001]
  notes: Shannon entropy and Kolmogorov complexity are complementary:
  H(X) = E[K(x)] + O(1) for computable distributions. Most strings are
  incompressible (K(x) ≈ |x|) — see CS-052 and INFO-023 for the counting
  argument.

INFO-022 (Derived Claim, Minimum Description Length Principle)
  The best model for data is the one that minimizes the total description
  length: length of model + length of data given model. MDL is the
  information-theoretic foundation of Occam's razor.
  type: derived_claim
  domain: information_theory
  confidence: 0.95
  depends: [INFO-021, INFO-008]
  source_tradition: "Rissanen (1978)"

INFO-023 (Derived Claim, No Free Lunch in Compression)
  No compression algorithm can compress all strings. For any compressor,
  there exist strings it makes longer. This follows from counting: there are
  more long strings than short representations.
  type: derived_claim
  domain: information_theory
  confidence: 1.0
  depends: [INFO-008, MATH-100]
```

---

# VI. INFORMATION AND PHYSICS

```
INFO-024 (Empirical Axiom, Information is Physical)
  Information is always encoded in a physical substrate. Processing, storing,
  and erasing information are physical processes subject to thermodynamic
  constraints.
  type: empirical_axiom
  domain: information_theory
  confidence: 0.98
  depends: [PHYS-024, PHYS-027]
  source_tradition: "Landauer (1961), Szilard (1929)"
  notes: Landauer's principle (CS-053) is evidence for this claim, not its
  foundation. The dependency runs: thermodynamics → information is physical →
  erasure has thermodynamic cost.

INFO-025 (Meta, Bekenstein Bound)
  The maximum entropy (information) in a region of space is bounded by its
  surface area: S ≤ 2πRE/(ℏc ln 2), where R is the radius and E is the
  total energy. Information has a physical density limit.
  type: meta
  domain: information_theory
  confidence: 0.90
  depends: [INFO-024, PHYS-077]
  source_tradition: "Bekenstein (1973)"
  notes: Conditional on black hole thermodynamics and GR assumptions. Connects
  information theory to general relativity. Suggests spacetime itself may be
  informational.

INFO-026 (Meta, Holographic Principle)
  The information content of a region of space is encoded on its boundary,
  not its volume. If correct, the universe is fundamentally two-dimensional
  in information content.
  type: meta
  domain: information_theory
  confidence: 0.80
  depends: [INFO-025]
  source_tradition: "'t Hooft (1993), Susskind (1995)"
  notes: Not experimentally confirmed at cosmological scales. Supported by
  black hole entropy calculations and AdS/CFT correspondence.

INFO-027 (Derived Claim, Quantum Information — No-Cloning)
  Quantum information cannot be perfectly copied (PHYS-100). This has
  information-theoretic consequences: quantum channels have fundamentally
  different capacity from classical channels, and quantum key distribution
  achieves information-theoretic security.
  type: derived_claim
  domain: information_theory
  confidence: 0.97
  depends: [PHYS-100]
  notes: Consequences for channel capacity and quantum key distribution follow
  from combining no-cloning with channel coding theory (INFO-014), but the
  no-cloning result itself is purely quantum-mechanical.
```

---

# VII. INFORMATION AND KNOWLEDGE (The Chain's Domain)

```
INFO-028 (Definition, Semantic Information)
  Shannon information is syntactic — it measures surprise, not meaning.
  Semantic information concerns truth-bearing content: a claim carries
  semantic information to the extent that it is true and informative
  (narrows the space of possibilities about the world).
  type: definition
  domain: information_theory
  confidence: 1.0
  depends: [INFO-001, LOGIC-018]
  source_tradition: "Bar-Hillel & Carnap (1953), Floridi (2004)"
  notes: The chain operates on semantic information. Proof of Truth verifies
  not just data integrity but truth-content of claims.

INFO-029 (Definition, Redundancy and Error Detection in Knowledge)
  In communication, redundancy protects against noise. In a knowledge system,
  cross-referencing claims (the DAG dependency structure) serves the same
  function: inconsistency between connected claims signals error.
  type: definition
  domain: information_theory
  confidence: 1.0
  depends: [INFO-017, LOGIC-074]
  notes: The axiom DAG is, among other things, an error-detecting code for
  knowledge.

INFO-030 (Derived Claim, Knowledge Compression and Axioms)
  Axioms are the maximally compressed representation of a knowledge domain.
  A good axiom set minimizes the number of independent assumptions while
  maximizing the space of derivable claims. This is the information-theoretic
  justification for axiomatic structure.
  type: derived_claim
  domain: information_theory
  confidence: 0.95
  depends: [INFO-022, LOGIC-021]

INFO-031 (Meta, Capacity of a Knowledge Economy)
  A knowledge economy has an information-theoretic capacity analogous to a
  communication channel: the rate at which verified, novel, true claims can
  be added to the knowledge base. The chain's throughput is bounded by
  verification bandwidth, not transaction bandwidth.
  type: meta
  domain: information_theory
  confidence: 0.85
  depends: [INFO-013, CS-045]

INFO-032 (Meta, Information Asymmetry and Markets)
  When parties to a transaction have different information, markets can fail
  (adverse selection, moral hazard). Reducing information asymmetry — making
  knowledge legible — is the economic purpose of this chain.
  type: meta
  domain: information_theory
  confidence: 0.90
  depends: [INFO-004, INFO-028]
  source_tradition: "Akerlof (1970), Stiglitz (2001)"
  notes: Bridges to ECON domain. Legible Money makes knowledge symmetric.

INFO-033 (Meta, The Chain as Information System)
  This chain is, at its deepest level, an information system: it encodes
  knowledge (INFO-028), structures it as a DAG (INFO-029), compresses it
  axiomatically (INFO-030), operates within capacity bounds (INFO-031),
  and exists to reduce information asymmetry (INFO-032). Shannon gave us the
  mathematics of uncertainty. This chain applies it to the economics of truth.
  type: meta
  domain: information_theory
  confidence: 1.0
  depends: [INFO-028, INFO-029, INFO-030, INFO-031, INFO-032]
```

---

## Summary

| Section | Items | Types | Content |
|---------|-------|-------|---------|
| I. Entropy | INFO-001→007 | 5 def, 2 derived | Shannon entropy, mutual information, KL divergence |
| II. Source Coding | INFO-008→011 | 1 def, 3 derived | Compression limits, Huffman, AEP |
| III. Channel Coding | INFO-012→016 | 2 def, 3 derived | Channel capacity, Shannon's theorem, BSC, Gaussian |
| IV. Error Correction | INFO-017→020 | 2 def, 2 derived | Codes, Singleton bound, Hamming bound, Reed-Solomon |
| V. Computation | INFO-021→023 | 1 def, 2 derived | Kolmogorov-Shannon bridge, MDL, compression limits |
| VI. Physics | INFO-024→027 | 1 emp, 1 derived, 2 meta | Information is physical, Bekenstein, holographic principle, quantum info |
| VII. Knowledge | INFO-028→033 | 2 def, 1 derived, 3 meta | Semantic info, knowledge compression, chain as info system |

Precise recount:
- Empirical axiom: INFO-024 = **1**
- Definition: 001, 003, 004, 005, 009, 012, 013, 017, 020, 021, 028, 029 = **12**
- Derived claim: 002, 006, 007, 008, 010, 011, 014, 015, 016, 018, 019, 022, 023, 027, 030 = **15**
- Meta: 025, 026, 031, 032, 033 = **5**

**Total: 1 empirical axiom + 12 definitions + 15 derived claims + 5 meta = 33 items.**

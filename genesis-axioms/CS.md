# Computer Science — Genesis Axioms (56 Items)

## Legend
* **Axiom**: foundational commitment accepted without proof.
* **Definition**: establishes vocabulary or convention (confidence 1.0).
* **Derived Claim**: proven from axioms/definitions in this or other domains.
* **Meta**: statement about computational systems or their limits.

Computer science is the chain's self-knowledge. The chain IS computation — it
must understand what computation can do, what it cannot, and what makes it trustworthy.

---

# I. COMPUTATION FOUNDATIONS

```
CS-001 (Definition, Alphabet and String)
  An alphabet Σ is a finite nonempty set of symbols. A string over Σ is a
  finite sequence of symbols from Σ. The set of all strings over Σ is Σ*.
  The empty string is ε.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [MATH-015]

CS-002 (Definition, Language)
  A language L over alphabet Σ is any subset of Σ*. A language may be finite
  or infinite.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-001, MATH-011]

CS-003 (Definition, Turing Machine)
  A Turing machine is a 7-tuple (Q, Σ, Γ, δ, q₀, q_accept, q_reject) with:
  finite state set Q, input alphabet Σ, tape alphabet Γ ⊇ Σ, transition
  function δ: Q × Γ → Q × Γ × {L,R}, start state q₀, accept state q_accept,
  reject state q_reject. It operates on an infinite tape, reading/writing one
  cell at a time.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-001]
  source_tradition: "Turing (1936)"

CS-004 (Definition, Turing-Recognizable and Turing-Decidable)
  A language L is Turing-recognizable (recursively enumerable) if some TM
  accepts every string in L. L is Turing-decidable (recursive) if some TM
  accepts every string in L and rejects every string not in L (always halts).
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-003, CS-002]

CS-005 (Axiom, Church-Turing Thesis)
  Every effectively computable function is computable by a Turing machine.
  type: axiom
  domain: computer_science
  confidence: 1.0
  depends: [CS-003]
  source_tradition: "Church (1936), Turing (1936)"
  notes: Not a theorem but a thesis — it cannot be formally proven because
  "effectively computable" is informal. Universally accepted in practice.
  Cross-ref LOGIC-049 (Church's theorem).

CS-005a (Meta, Physical Church-Turing Thesis)
  No physically realizable computational device exceeds the computational
  power of a Turing machine. As far as currently known, physics does not
  enable hypercomputation.
  type: meta
  domain: computer_science
  confidence: 0.90
  depends: [CS-005]
  notes: Stronger than CS-005 and contested by some (hypercomputation proposals,
  relativistic spacetime models, analog computation idealizations). The chain
  adopts this as a working assumption with acknowledged uncertainty.

CS-006 (Derived Claim, Universal Turing Machine)
  There exists a Turing machine U that, given an encoding of any TM M and
  input w, simulates M on w. Computation is self-describing.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-003]
  source_tradition: "Turing (1936)"

CS-007 (Derived Claim, Halting Problem — Undecidability)
  There is no Turing machine that decides, for arbitrary TM M and input w,
  whether M halts on w.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-006]
  source_tradition: "Turing (1936)"
  notes: Cross-ref MATH-104, LOGIC-049.

CS-008 (Derived Claim, Rice's Theorem)
  Every nontrivial semantic property of programs is undecidable.
  formal: If P is a nontrivial property of Turing-recognizable languages,
  then {⟨M⟩ : L(M) ∈ P} is undecidable.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-007]
  source_tradition: "Rice (1953)"

CS-009 (Definition, Lambda Calculus)
  The lambda calculus is a formal system where: terms are variables, abstractions
  (λx.M), or applications (M N). Computation proceeds by β-reduction:
  (λx.M)N → M[N/x]. Church-Turing equivalent to Turing machines.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-005]
  source_tradition: "Church (1936)"
  notes: Cross-ref LOGIC-055 (Curry-Howard correspondence).

CS-010 (Definition, Recursive and Primitive Recursive Functions)
  Primitive recursive functions are built from zero, successor, and projection
  by composition and primitive recursion. Recursive (μ-recursive) functions
  add unbounded search (μ-operator). The recursive functions are exactly the
  Turing-computable functions.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-005, MATH-015]
```

---

# II. COMPLEXITY THEORY

```
CS-011 (Definition, Time Complexity)
  The time complexity of a TM M is f(n) if M halts on every input of length n
  within f(n) steps. TIME(f(n)) is the class of languages decidable in O(f(n))
  time by a deterministic TM.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-004]

CS-012 (Definition, Space Complexity)
  The space complexity of a TM M is f(n) if M uses at most f(n) tape cells on
  inputs of length n. SPACE(f(n)) is the class of languages decidable in O(f(n))
  space.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-004]

CS-013 (Definition, P — Polynomial Time)
  P = ∪_k TIME(n^k). The class of decision problems solvable in polynomial time
  by a deterministic Turing machine. Informally: "efficiently solvable."
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-011]

CS-014 (Definition, NP — Nondeterministic Polynomial Time)
  NP is the class of decision problems for which a "yes" answer has a
  polynomial-time verifiable certificate. Equivalently: solvable in polynomial
  time by a nondeterministic TM.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-013]

CS-015 (Definition, NP-Completeness)
  A language L is NP-complete if: (i) L ∈ NP, and (ii) every language in NP
  is polynomial-time reducible to L. If any NP-complete problem is in P,
  then P = NP.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-014]
  source_tradition: "Cook (1971), Karp (1972)"

CS-016 (Derived Claim, Cook-Levin Theorem)
  SAT (Boolean satisfiability) is NP-complete.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-015]
  source_tradition: "Cook (1971), Levin (1973)"

CS-017 (Meta, P ≠ NP Conjecture)
  It is widely conjectured that P ≠ NP: there exist problems whose solutions
  are efficiently verifiable but not efficiently findable. This remains the most
  important open problem in theoretical computer science.
  type: meta
  domain: computer_science
  confidence: 0.95
  depends: [CS-013, CS-014]
  notes: Not proven. The chain records the conjecture with high confidence
  based on decades of failed attempts and structural evidence.

CS-018 (Definition, PSPACE)
  PSPACE = ∪_k SPACE(n^k). P ⊆ NP ⊆ PSPACE ⊆ EXPTIME.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-012, CS-014]

CS-019 (Derived Claim, Space Hierarchy Theorem)
  For space-constructible f, SPACE(o(f(n))) ⊊ SPACE(f(n)).
  More space strictly means more power.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-012]

CS-020 (Derived Claim, Time Hierarchy Theorem)
  For time-constructible f, TIME(o(f(n))) ⊊ TIME(f(n) · log f(n)).
  More time strictly means more power.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-011]
```

---

# III. ALGORITHMS AND DATA STRUCTURES

```
CS-021 (Definition, Algorithm)
  An algorithm is a finite, unambiguous procedure that produces correct output
  for its specified inputs. A total algorithm terminates on all inputs; partial
  algorithms, online algorithms, and interactive/reactive systems may not.
  An algorithm's efficiency is measured by time and space complexity as
  functions of input size.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-005, CS-011]

CS-022 (Derived Claim, Sorting Lower Bound)
  Comparison-based sorting requires Ω(n log n) comparisons in the worst case.
  Merge sort and heapsort achieve this bound.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-021, MATH-110, MATH-060]

CS-023 (Derived Claim, Search in Sorted Data)
  Binary search finds an element in a sorted array of n elements in O(log n)
  comparisons. This is optimal for comparison-based search.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-021]

CS-024 (Definition, Hash Function)
  A hash function h: U → {0,…,m−1} maps keys from a universe U to table
  indices. A good hash function distributes keys uniformly. Hash tables support
  average-case O(1) lookup, insert, and delete.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-021]

CS-025 (Definition, Graph)
  A graph G = (V, E) consists of vertices V and edges E ⊆ V × V.
  Directed: edges are ordered pairs. Undirected: edges are unordered pairs.
  Weighted: edges carry values. Graphs model networks, dependencies, and relations.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [MATH-006]

CS-026 (Derived Claim, Shortest Path — Dijkstra)
  In a graph with non-negative edge weights, Dijkstra's algorithm finds
  shortest paths from a source vertex in O((V + E) log V) with a priority queue.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-025]
  source_tradition: "Dijkstra (1959)"
```

---

# IV. CRYPTOGRAPHY

```
CS-027 (Definition, One-Way Function)
  A function f is one-way if it is efficiently computable but no probabilistic
  polynomial-time algorithm can invert it on a random input with non-negligible
  probability.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-013]
  notes: The existence of one-way functions is unproven (implies P ≠ NP).
  All of modern cryptography rests on this assumption.

CS-028 (Definition, Cryptographic Hash Function)
  A cryptographic hash function H: {0,1}* → {0,1}^n satisfies:
  (i) preimage resistance (given y, hard to find x with H(x) = y);
  (ii) second-preimage resistance (given x, hard to find x' ≠ x with H(x) = H(x'));
  (iii) collision resistance (hard to find any x ≠ x' with H(x) = H(x')).
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-027, CS-024]

CS-029 (Definition, Symmetric Encryption)
  A symmetric encryption scheme (Gen, Enc, Dec) uses the same key k for
  encryption and decryption: Dec(k, Enc(k, m)) = m. Security: ciphertext
  reveals nothing about plaintext without k.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-027]

CS-030 (Definition, Public-Key Encryption)
  An asymmetric encryption scheme uses a key pair (pk, sk): Enc(pk, m) produces
  ciphertext decryptable only by Dec(sk, ·). pk is public; sk is secret.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-027]
  source_tradition: "Diffie & Hellman (1976), RSA (1977)"

CS-031 (Definition, Digital Signature)
  A digital signature scheme (Gen, Sign, Verify) produces a signature σ = Sign(sk, m)
  verifiable by anyone with pk: Verify(pk, m, σ) = accept. Unforgeable without sk.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-030]
  notes: Foundational to blockchain — every transaction is signed.

CS-032 (Definition, Zero-Knowledge Proof)
  A zero-knowledge proof for language L allows a prover to convince a verifier
  that x ∈ L without revealing any information beyond the truth of the statement.
  Properties: completeness, soundness, zero-knowledge.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-014, CS-027]
  source_tradition: "Goldwasser, Micali & Rackoff (1985)"

CS-033 (Meta, Computational Hardness Assumptions)
  Modern cryptography relies on specific hardness assumptions:
  (i) Integer factorization is hard (RSA);
  (ii) Discrete logarithm is hard (Diffie-Hellman, ECDSA);
  (iii) Learning with errors is hard (lattice-based, best current candidates
  for post-quantum resistance).
  None are proven hard; all are conjectured based on extensive cryptanalysis.
  type: meta
  domain: computer_science
  confidence: 0.95
  depends: [CS-027, CS-017]
```

---

# V. DISTRIBUTED SYSTEMS

```
CS-034 (Definition, Distributed System)
  A distributed system is a collection of independent computational nodes that
  communicate by message passing over a network. Nodes may fail independently.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-003]

CS-035 (Empirical Axiom, Network Unreliability)
  Messages between nodes may be delayed, reordered, duplicated, or lost.
  Network partitions (total communication failure between subsets of nodes)
  occur in practice.
  type: empirical_axiom
  domain: computer_science
  confidence: 0.99
  depends: [CS-034]

CS-036 (Derived Claim, Byzantine Generals Problem)
  In a system with n nodes, up to t of which may be arbitrarily faulty
  (Byzantine), consensus is achievable iff n ≥ 3t + 1 (with synchronous
  communication) or with additional cryptographic assumptions.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-034, CS-035]
  source_tradition: "Lamport, Shostak & Pease (1982)"

CS-037 (Derived Claim, FLP Impossibility)
  In an asynchronous distributed system with even one faulty process, there is
  no deterministic algorithm that guarantees consensus.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-034, CS-035]
  source_tradition: "Fischer, Lynch & Paterson (1985)"

CS-038 (Derived Claim, CAP Theorem)
  A distributed data store cannot simultaneously provide all three of:
  Consistency (all nodes see the same data), Availability (every request gets
  a response), and Partition tolerance. Under partition, a system must trade off
  between strong consistency and availability.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-035]
  source_tradition: "Brewer (2000), Gilbert & Lynch (2002)"

CS-039 (Definition, State Machine Replication)
  If all correct nodes process the same sequence of commands in the same order,
  starting from the same state, they reach the same state. Consensus protocols
  (Paxos, Raft, BFT) implement this to achieve fault tolerance.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-036]
  source_tradition: "Schneider (1990), Lamport (1998)"

CS-040 (Definition, Eventual Consistency)
  A replicated system is eventually consistent if, given no new updates, all
  replicas converge to the same value. Weaker than strong consistency but
  available under partition.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-038]
```

---

# VI. BLOCKCHAIN AND CONSENSUS

```
CS-041 (Definition, Blockchain)
  A blockchain is an append-only data structure of blocks, each containing a
  set of transactions and a cryptographic hash of the previous block, forming
  a tamper-evident chain. Modifying any block invalidates all subsequent hashes.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-028, CS-031]

CS-042 (Definition, Consensus Protocol)
  A consensus protocol achieves agreement among distributed nodes on the next
  block of transactions. Properties: safety (no two honest nodes finalize
  different blocks at the same height), liveness (the chain makes progress),
  and fault tolerance (operates despite bounded failures).
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-039, CS-041]

CS-043 (Definition, Proof of Work)
  A consensus mechanism where nodes compete to find a nonce such that
  H(block || nonce) < target. Finding a valid nonce requires computational
  work; verifying requires one hash evaluation. Sybil-resistant by binding
  voting power to physical resources (energy).
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-028, CS-042]
  source_tradition: "Nakamoto (2008)"

CS-044 (Definition, Proof of Stake)
  A consensus mechanism where validators are selected proportionally to their
  staked collateral. Misbehavior is punished by slashing (destroying stake).
  Sybil-resistant by binding voting power to economic commitment.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-042]

CS-045 (Definition, Proof of Truth)
  The consensus mechanism of this chain. Validators stake claims with explicit
  confidence, type, and dependency structure. Claims are accepted by verification
  against the chain's axiom DAG and challenge protocol. Sybil-resistant by
  binding voting power to verified knowledge contribution.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-042, CS-041, LOGIC-074, LOGIC-075, LOGIC-076]
  notes: This is the chain defining itself. Cross-ref LOGIC-074 (inference rules),
  LOGIC-075 (confidence propagation), LOGIC-076 (challenge protocol).

CS-046 (Definition, Smart Contract)
  A smart contract is a program stored on a blockchain that executes
  deterministically when invoked by a transaction. Given the same state and
  input, all nodes compute the same output. Enables programmable, trustless
  agreements.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-041, CS-039]

CS-047 (Definition, Merkle Tree)
  A Merkle tree is a binary hash tree where each leaf is a hash of a data block
  and each internal node is a hash of its children. The root hash commits to
  all data. Membership proofs require O(log n) hashes.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-028]
  source_tradition: "Merkle (1979)"
```

---

# VII. FORMAL VERIFICATION AND TYPE THEORY

```
CS-048 (Definition, Type System)
  A type system assigns types to program expressions, constraining their
  behavior. A sound type system guarantees: well-typed programs do not produce
  type errors at runtime (type safety / progress + preservation).
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-009, LOGIC-028]
  notes: Cross-ref LOGIC-055 (Curry-Howard). Types are propositions; programs
  are proofs.

CS-049 (Definition, Formal Verification)
  Formal verification is the use of mathematical proof to establish that a
  system (program, protocol, circuit) satisfies a specification. Methods include
  model checking, theorem proving, and abstract interpretation.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [LOGIC-022, CS-048]

CS-050 (Derived Claim, Verified Computation)
  A computation can be proven correct without re-executing it, via:
  (i) proof-carrying code (proof accompanies result);
  (ii) interactive proofs (verifier queries prover);
  (iii) succinct arguments (SNARKs/STARKs — short proofs of arbitrary computation).
  type: derived_claim
  domain: computer_science
  confidence: 0.97
  depends: [CS-032, CS-049]
  notes: Critical for Proof of Truth: the chain must verify claims without
  trusting the claimant.
```

---

# VIII. INFORMATION-THEORETIC LIMITS OF COMPUTATION

```
CS-051 (Definition, Kolmogorov Complexity)
  The Kolmogorov complexity K(x) of a string x is the length of the shortest
  program (on a fixed universal TM) that produces x. K(x) provides the
  theoretical foundation for randomness and compression.
  type: definition
  domain: computer_science
  confidence: 1.0
  depends: [CS-006]
  source_tradition: "Kolmogorov (1965), Solomonoff (1964), Chaitin (1966)"

CS-052 (Derived Claim, Incompressibility and Randomness)
  K(x) is uncomputable — no algorithm can compute it for all x. A string x is
  Kolmogorov-random if K(x) ≥ |x| (incompressible). Most strings are random;
  randomness is the norm, structure is the exception.
  type: derived_claim
  domain: computer_science
  confidence: 1.0
  depends: [CS-051, CS-007]

CS-053 (Derived Claim, Landauer's Principle)
  Erasing one bit of information dissipates at least kT ln 2 joules of energy.
  Computation has a thermodynamic cost; reversible computation can approach
  but not reach zero dissipation.
  type: derived_claim
  domain: computer_science
  confidence: 0.95
  depends: [INFO-024, PHYS-024, PHYS-027]
  source_tradition: "Landauer (1961)"
  notes: Bridges computation to physics. Information is physical (INFO-024);
  erasure is the specific thermodynamic act that makes the cost concrete.

CS-054 (Meta, Quantum Computation)
  Quantum computers exploit superposition and entanglement to solve certain
  problems faster than classical computers (e.g., Shor's algorithm factors
  integers in polynomial time; Grover's algorithm searches in O(√n)).
  Quantum computers do not violate Church-Turing but may violate the
  Extended Church-Turing Thesis (efficient simulation).
  type: meta
  domain: computer_science
  confidence: 0.93
  depends: [CS-005, PHYS-089, PHYS-092]
  notes: Threat to CS-033 hardness assumptions (factoring, discrete log).
  Lattice-based assumptions are the best current candidates for quantum resistance.

CS-055 (Meta, The Chain Knows Its Computational Limits)
  By the halting problem (CS-007), Rice's theorem (CS-008), and FLP (CS-037),
  the chain cannot: (i) decide all properties of submitted programs;
  (ii) guarantee consensus in fully asynchronous settings without additional
  assumptions; (iii) compress knowledge below its Kolmogorov complexity.
  The chain is born knowing what computation cannot do.
  type: meta
  domain: computer_science
  confidence: 1.0
  depends: [CS-007, CS-008, CS-037, CS-051]
  notes: Mirrors LOGIC-077 (the chain knows its logical limits) and extends
  it to computational limits.
```

---

## Summary

| Section | Items | Types | Content |
|---------|-------|-------|---------|
| I. Computation Foundations | CS-001→010 | 1 axiom, 1 meta, 7 def, 2 derived | TMs, Church-Turing, physical CT, UTM, halting, lambda calculus |
| II. Complexity Theory | CS-011→020 | 8 def, 2 derived | P, NP, NP-completeness, hierarchy theorems |
| III. Algorithms | CS-021→026 | 3 def, 3 derived | Sorting, searching, hashing, graphs |
| IV. Cryptography | CS-027→033 | 6 def, 1 meta | One-way functions, hashing, encryption, signatures, ZKP, hardness |
| V. Distributed Systems | CS-034→040 | 4 def, 3 derived, 1 emp | Byzantine fault tolerance, FLP, CAP, replication |
| VI. Blockchain | CS-041→047 | 7 def | Blockchain, PoW, PoS, Proof of Truth, smart contracts, Merkle |
| VII. Verification | CS-048→050 | 2 def, 1 derived | Type systems, formal verification, verified computation |
| VIII. Limits | CS-051→055 | 1 def, 2 derived, 2 meta | Kolmogorov complexity, Landauer, quantum, self-knowledge |

Precise recount:
- Axiom: CS-005 = **1**
- Empirical axiom: CS-035 = **1**
- Definition: 001, 002, 003, 004, 009, 010, 011, 012, 013, 014, 015, 018, 021, 024, 025, 027, 028, 029, 030, 031, 032, 034, 039, 040, 041, 042, 043, 044, 045, 046, 047, 048, 049, 051 = **34**
- Derived claim: 006, 007, 008, 016, 019, 020, 022, 023, 026, 036, 037, 038, 050, 052, 053 = **15**
- Meta: 005a, 017, 033, 054, 055 = **5**

**Total: 1 axiom + 1 empirical axiom + 34 definitions + 15 derived claims + 5 meta = 56 items.**

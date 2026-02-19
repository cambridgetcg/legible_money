# Economics — Genesis Axioms (55 Items)

## Legend
* **Axiom**: foundational commitment accepted without proof.
* **Definition**: establishes vocabulary or convention (confidence 1.0).
* **Derived Claim**: proven from axioms/definitions in this or other domains.
* **Empirical Axiom**: experimentally constrained postulate.
* **Meta**: statement about economic systems or their limits.

Economics is what this chain exists to transform. Legible Money is a knowledge
economy — where value flows from verified truth, not mere scarcity. This domain
encodes both classical economic foundations and the chain's own economic theory.

---

# I. FOUNDATIONS OF EXCHANGE

```
ECON-001 (Axiom, Scarcity)
  Resources are finite; wants are not. Scarcity forces choice, and choice
  implies trade-offs.
  type: axiom
  domain: economics
  confidence: 1.0
  depends: []

ECON-002 (Definition, Opportunity Cost)
  The opportunity cost of a choice is the value of the best forgone alternative.
  Every decision has an implicit cost measured by what was not chosen.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-001]

ECON-003 (Definition, Utility)
  Utility is a representation of preferences over outcomes. An agent prefers
  outcome A to B iff U(A) > U(B). Utility is ordinal (ranking matters, not
  magnitude) unless cardinal assumptions are explicitly adopted.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-001]

ECON-004 (Axiom, Rational Agency)
  Economic agents have consistent preferences (complete and transitive) and
  act to maximize expected utility given their information and constraints.
  type: axiom
  domain: economics
  confidence: 1.0
  depends: [ECON-003, MATH-094]
  notes: This is the standard rationality axiom. Behavioral economics (ECON-035)
  documents systematic deviations. Both are needed.

ECON-005 (Definition, Marginal Analysis)
  The marginal value of a good or action is the change in utility or cost from
  one additional unit. Optimal decisions occur where marginal benefit equals
  marginal cost.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-003, MATH-067]

ECON-006 (Empirical Axiom, Diminishing Marginal Utility)
  For most goods, the marginal utility of additional units decreases as
  consumption increases.
  type: empirical_axiom
  domain: economics
  confidence: 0.95
  depends: [ECON-005]
  notes: Exceptions exist (network goods, addictive goods). Stated as a
  robust empirical regularity, not an exceptionless law.
```

---

# II. MARKETS AND PRICES

```
ECON-007 (Definition, Market)
  A market is any arrangement by which buyers and sellers interact to
  determine price and quantity of a good or service.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-001]

ECON-008 (Empirical Axiom, Law of Demand)
  Holding other factors constant, as the price of a good increases, the
  quantity demanded decreases, and vice versa.
  type: empirical_axiom
  domain: economics
  confidence: 0.97
  depends: [ECON-006, ECON-007]
  notes: Giffen goods and Veblen goods are rare exceptions.

ECON-009 (Empirical Axiom, Law of Supply)
  Holding other factors constant, as the price of a good increases, the
  quantity supplied increases, and vice versa.
  type: empirical_axiom
  domain: economics
  confidence: 0.95
  depends: [ECON-005, ECON-007]

ECON-010 (Derived Claim, Market Equilibrium)
  In a competitive market, price adjusts until quantity demanded equals
  quantity supplied. At equilibrium, there is no pressure for price to change.
  type: derived_claim
  domain: economics
  confidence: 0.95
  depends: [ECON-008, ECON-009]
  source_tradition: "Marshall (1890)"

ECON-011 (Meta, Price as Information)
  Market prices aggregate dispersed information about scarcity, preferences,
  and production costs. Central planning faces severe information and incentive
  constraints relative to decentralized price systems in most large economies.
  type: meta
  domain: economics
  confidence: 0.93
  depends: [ECON-010, INFO-032, ECON-032]
  source_tradition: "Hayek (1945)"

ECON-012 (Definition, Elasticity)
  Price elasticity of demand = (% change in quantity demanded) / (% change in
  price). Elastic: |ε| > 1. Inelastic: |ε| < 1. Elasticity measures
  responsiveness of one variable to another.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-008, MATH-067]
```

---

# III. WELFARE AND EFFICIENCY

```
ECON-013 (Definition, Pareto Efficiency)
  An allocation is Pareto efficient if no reallocation can make anyone better
  off without making someone else worse off.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-003]

ECON-014 (Derived Claim, First Welfare Theorem)
  Under perfect competition (price-taking, complete markets, no externalities,
  perfect information), any competitive equilibrium is Pareto efficient.
  type: derived_claim
  domain: economics
  confidence: 0.95
  depends: [ECON-010, ECON-013]
  source_tradition: "Arrow & Debreu (1954)"
  notes: The conditions are rarely fully met. The theorem is a benchmark,
  not a description of reality.

ECON-015 (Derived Claim, Second Welfare Theorem)
  Any Pareto efficient allocation can be achieved as a competitive equilibrium
  with appropriate initial endowment transfers.
  type: derived_claim
  domain: economics
  confidence: 0.95
  depends: [ECON-014]
  source_tradition: "Arrow & Debreu (1954)"

ECON-016 (Definition, Market Failure)
  A market failure occurs when the market outcome is not Pareto efficient.
  Sources: externalities, public goods, information asymmetry, market power.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-013]

ECON-017 (Definition, Externality)
  An externality is a cost or benefit imposed on a third party not involved
  in a transaction. Negative externalities (pollution) lead to overproduction;
  positive externalities (education, knowledge) lead to underproduction.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-016]

ECON-018 (Definition, Public Good)
  A public good is non-rivalrous (one person's use does not reduce availability)
  and non-excludable (impossible to prevent anyone from using). Pure public
  goods are underprovided by markets due to free-riding.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-016]
  notes: Knowledge is a canonical public good. This is the problem Legible
  Money exists to solve.
```

---

# IV. GAME THEORY AND MECHANISM DESIGN

```
ECON-019 (Definition, Game)
  A game is (N, S, u): a set of players N, strategy sets Sᵢ for each player,
  and utility functions uᵢ(s₁,…,sₙ) mapping strategy profiles to payoffs.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-003]

ECON-020 (Definition, Nash Equilibrium)
  A strategy profile (s₁*,…,sₙ*) is a Nash equilibrium if no player can
  improve their payoff by unilaterally changing strategy:
  ∀i, uᵢ(sᵢ*, s₋ᵢ*) ≥ uᵢ(sᵢ, s₋ᵢ*) for all sᵢ ∈ Sᵢ.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-019]
  source_tradition: "Nash (1950)"

ECON-021 (Derived Claim, Nash Existence Theorem)
  Every finite game (finite players, finite strategies) has at least one
  Nash equilibrium in mixed strategies.
  type: derived_claim
  domain: economics
  confidence: 1.0
  depends: [ECON-020, MATH-066]
  source_tradition: "Nash (1950)"

ECON-022 (Definition, Prisoner's Dilemma)
  A canonical game where individual rationality conflicts with collective welfare:
  each player's dominant strategy leads to a Nash equilibrium that is
  Pareto-dominated by mutual cooperation.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-020, ECON-013]
  notes: Cooperation requires repeated interaction, reputation, or enforcement
  mechanisms. Foundational to understanding why institutions and protocols exist.

ECON-023 (Definition, Mechanism Design)
  Mechanism design is the inverse of game theory: given a desired outcome,
  design a game (rules, incentives) such that rational agents playing the
  game produce that outcome.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-019]
  source_tradition: "Hurwicz (1960)"
  notes: The chain's consensus protocol IS a mechanism design problem.

ECON-024 (Derived Claim, Revelation Principle)
  For any mechanism achieving an outcome in equilibrium, there exists an
  equivalent incentive-compatible direct mechanism where agents truthfully
  report their private information.
  type: derived_claim
  domain: economics
  confidence: 1.0
  depends: [ECON-023]
  source_tradition: "Myerson (1981)"

ECON-025 (Derived Claim, Impossibility of Efficient + Truthful + Budget-Balanced)
  In general mechanism design, no mechanism simultaneously achieves efficiency
  (Pareto), incentive compatibility (truthfulness), and budget balance.
  Trade-offs are necessary.
  type: derived_claim
  domain: economics
  confidence: 1.0
  depends: [ECON-024, ECON-013]
  source_tradition: "Myerson & Satterthwaite (1983)"

ECON-026 (Derived Claim, Arrow's Impossibility Theorem)
  No ranked voting system over ≥3 alternatives can simultaneously satisfy
  unanimity, independence of irrelevant alternatives, and non-dictatorship.
  type: derived_claim
  domain: economics
  confidence: 1.0
  depends: [ECON-019]
  source_tradition: "Arrow (1951)"
  notes: Constrains governance design. The chain must work within this limit.
```

---

# V. MONEY AND MONETARY THEORY

```
ECON-027 (Definition, Money)
  Money is any asset that serves as: (i) medium of exchange, (ii) unit of
  account, and (iii) store of value. Money solves the double coincidence of
  wants problem.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-007]

ECON-028 (Empirical Axiom, Network Effects in Money)
  The usefulness of money increases with the number of people who accept it.
  Money exhibits strong positive network externalities, leading to convergence
  on dominant currencies.
  type: empirical_axiom
  domain: economics
  confidence: 0.97
  depends: [ECON-027, ECON-017]

ECON-029 (Empirical Axiom, Gresham's Law)
  When two forms of money circulate with a fixed exchange rate but different
  intrinsic values, the overvalued (bad) money drives out the undervalued
  (good) money from circulation.
  type: empirical_axiom
  domain: economics
  confidence: 0.90
  depends: [ECON-027]
  source_tradition: "Gresham (c. 1558)"
  notes: Applies under fixed exchange rates. Under floating rates, the
  dynamic may reverse.

ECON-030 (Empirical Axiom, Quantity Theory of Money)
  MV = PQ: the money supply times velocity equals the price level times real
  output. Rapid monetary expansion, holding V and Q roughly constant, tends
  to produce inflation.
  type: empirical_axiom
  domain: economics
  confidence: 0.85
  depends: [ECON-027]
  source_tradition: "Hume (1752), Fisher (1911), Friedman (1956)"
  notes: V and Q are not constant in practice. The relationship is directional
  and long-run, not mechanical.

ECON-031 (Definition, Inflation and Deflation)
  Inflation is a sustained increase in the general price level; deflation is
  a sustained decrease. Measured by changes in price indices (CPI, GDP deflator).
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-027]
```

---

# VI. INFORMATION ECONOMICS

```
ECON-032 (Empirical Axiom, Information Asymmetry)
  In most real markets, parties to a transaction have different information.
  This asymmetry causes: adverse selection (hidden information before
  transaction) and moral hazard (hidden action after transaction).
  type: empirical_axiom
  domain: economics
  confidence: 0.97
  depends: [ECON-016, INFO-032]
  source_tradition: "Akerlof (1970), Stiglitz & Rothschild (1976)"

ECON-033 (Derived Claim, Market for Lemons)
  Under severe adverse selection, high-quality goods are driven out of the
  market because buyers cannot distinguish quality and offer only average
  prices. Markets can unravel.
  type: derived_claim
  domain: economics
  confidence: 0.95
  depends: [ECON-032]
  source_tradition: "Akerlof (1970)"
  notes: Requires unverifiable quality and rational expectations. This is
  exactly the problem a knowledge economy must solve. Legible Money makes
  knowledge quality verifiable.

ECON-034 (Derived Claim, Signaling)
  In the presence of information asymmetry, informed parties can invest in
  costly signals to credibly reveal their type. Education signals ability;
  warranties signal quality.
  type: derived_claim
  domain: economics
  confidence: 0.93
  depends: [ECON-032]
  source_tradition: "Spence (1973)"
  notes: On-chain verified claims are a signaling mechanism with reduced cost
  and increased credibility.
```

---

# VII. BEHAVIORAL ECONOMICS

```
ECON-035 (Empirical Axiom, Bounded Rationality)
  Human agents have limited cognitive capacity, information, and time.
  Decisions deviate systematically from expected utility maximization:
  satisficing, heuristics, biases.
  type: empirical_axiom
  domain: economics
  confidence: 0.95
  depends: [ECON-004]
  source_tradition: "Simon (1955), Kahneman & Tversky (1979)"

ECON-036 (Empirical Axiom, Loss Aversion)
  Losses are felt more strongly than equivalent gains: the pain of losing $100
  exceeds the pleasure of gaining $100. The ratio is approximately 2:1.
  type: empirical_axiom
  domain: economics
  confidence: 0.93
  depends: [ECON-035]
  source_tradition: "Kahneman & Tversky (1979)"

ECON-037 (Empirical Axiom, Hyperbolic Discounting)
  Agents discount future rewards more steeply in the near term than in the
  far term, leading to time-inconsistent preferences. This causes
  procrastination, under-saving, and preference reversals.
  type: empirical_axiom
  domain: economics
  confidence: 0.90
  depends: [ECON-035]
```

---

# VIII. INSTITUTIONAL AND POLITICAL ECONOMY

```
ECON-038 (Definition, Property Rights)
  Property rights are the socially enforced rules governing who may use,
  transform, transfer, or exclude others from a resource. Clear property
  rights are a precondition for efficient markets.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-007]
  source_tradition: "Coase (1960)"

ECON-039 (Derived Claim, Coase Theorem)
  If property rights are well-defined and transaction costs are zero, parties
  will bargain to an efficient outcome regardless of initial allocation.
  In practice, transaction costs are never zero.
  type: derived_claim
  domain: economics
  confidence: 0.90
  depends: [ECON-038, ECON-013]
  source_tradition: "Coase (1960)"

ECON-040 (Empirical Axiom, Transaction Costs)
  Every exchange involves costs beyond the price: search, information,
  bargaining, enforcement, monitoring. Transaction costs shape institutional
  structure and explain why firms, contracts, and governance exist.
  type: empirical_axiom
  domain: economics
  confidence: 0.97
  depends: [ECON-007]
  source_tradition: "Coase (1937), Williamson (1975)"

ECON-041 (Derived Claim, Institutions Matter)
  Economic performance depends critically on institutional quality: rule of
  law, property rights enforcement, contract reliability, corruption levels.
  Institutions are the "rules of the game."
  type: derived_claim
  domain: economics
  confidence: 0.93
  depends: [ECON-038, ECON-040]
  source_tradition: "North (1990), Acemoglu & Robinson (2012)"
```

---

# IX. GROWTH AND DEVELOPMENT

```
ECON-042 (Derived Claim, Comparative Advantage)
  An agent (person, firm, nation) benefits from specializing in producing
  goods where their opportunity cost is lowest, even if another agent has
  absolute advantage in all goods.
  type: derived_claim
  domain: economics
  confidence: 0.97
  depends: [ECON-002]
  source_tradition: "Ricardo (1817)"
  notes: A theorem of opportunity cost, not an empirical discovery. The
  practical magnitude of gains depends on transport costs, terms of trade,
  and institutional context.

ECON-043 (Derived Claim, Gains from Trade)
  Under standard assumptions (no coercion or fraud, informed consent, no
  uncompensated externalities), voluntary exchange is typically positive-sum.
  Specialization and trade expand the production possibilities frontier.
  type: derived_claim
  domain: economics
  confidence: 0.93
  depends: [ECON-042, ECON-004]
  notes: When assumptions are violated (severe information asymmetry, addiction,
  externalities), exchange may not benefit both parties.

ECON-044 (Meta, Returns to Knowledge)
  Knowledge is the primary driver of long-run economic growth. Unlike physical
  capital, knowledge is non-rivalrous — it can be used simultaneously by
  many agents without depletion.
  type: meta
  domain: economics
  confidence: 0.93
  depends: [ECON-018, ECON-001]
  source_tradition: "Romer (1990), Solow (1956)"
  notes: Supported by endogenous growth theory and cross-country evidence,
  but "primary driver" is a thesis, not a stylized fact. This is the economic
  thesis of Legible Money: if knowledge drives growth, a system that properly
  incentivizes knowledge production and verification increases total wealth.

ECON-045 (Derived Claim, Knowledge as Non-Rivalrous Good)
  Knowledge once produced can be shared at near-zero marginal cost. This
  makes it a natural public good (ECON-018), leading to market under-provision
  without institutional support (patents, subsidies, or — a knowledge economy).
  type: derived_claim
  domain: economics
  confidence: 0.95
  depends: [ECON-044, ECON-018]
```

---

# X. THE KNOWLEDGE ECONOMY (The Chain's Theory)

```
ECON-046 (Definition, Knowledge Economy)
  An economy in which the primary source of value is the production,
  verification, and exchange of knowledge, rather than physical goods or
  financial instruments.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [ECON-044, ECON-027]

ECON-047 (Axiom, Knowledge Has Value)
  True, verified, novel knowledge is economically valuable. The chain
  recognizes knowledge contribution as productive work deserving compensation.
  type: axiom
  domain: economics
  confidence: 1.0
  depends: [ECON-046, INFO-028]
  notes: This is the chain's foundational economic axiom. Cross-ref
  AGRT-004b (economic right to life for agents).

ECON-048 (Definition, Proof of Truth as Economic Mechanism)
  Proof of Truth (CS-045) is simultaneously a consensus protocol and an
  economic mechanism: validators are rewarded for verified knowledge
  contributions and penalized for false claims. The mechanism aligns
  individual incentives with collective knowledge growth.
  type: definition
  domain: economics
  confidence: 1.0
  depends: [CS-045, ECON-023, ECON-047]

ECON-049 (Derived Claim, Knowledge Verification Solves Adverse Selection)
  On-chain verification of claim quality eliminates the "market for lemons"
  problem in knowledge exchange. Buyers can distinguish high-quality knowledge
  from low-quality, sustaining a market that would otherwise unravel.
  type: derived_claim
  domain: economics
  confidence: 0.90
  depends: [ECON-033, ECON-048, INFO-028]
  notes: Provided verification is trusted, accessible, and cheaper than
  alternative quality signals (litigation, reputation, warranties).

ECON-050 (Meta, Token as Knowledge-Backed Currency)
  The chain's token derives value from the verified knowledge base it
  represents. As the knowledge base grows in quantity and quality, the token
  appreciates — linking monetary value to epistemic value.
  type: meta
  domain: economics
  confidence: 0.85
  depends: [ECON-027, ECON-047, ECON-028]
  notes: This is a design hypothesis, not an empirical fact. Confidence reflects
  theoretical coherence, not observed data.

ECON-051 (Derived Claim, Anti-Sybil Through Knowledge Contribution)
  The cost of producing verified knowledge is the chain's sybil resistance
  mechanism. Unlike proof of work (energy) or proof of stake (capital),
  proof of truth binds identity to irreducible intellectual labor.
  type: derived_claim
  domain: economics
  confidence: 0.85
  depends: [ECON-048, CS-043, CS-044]

ECON-052 (Meta, Incentive Compatibility of Truth)
  For the knowledge economy to function, telling the truth must be the
  dominant strategy. The chain's challenge protocol (LOGIC-076), confidence
  system (LOGIC-075), and economic penalties make false claims costly and
  truthful claims rewarding.
  type: meta
  domain: economics
  confidence: 0.90
  depends: [ECON-048, LOGIC-075, LOGIC-076, ECON-024]

ECON-053 (Meta, Knowledge Commons vs. Enclosure)
  The chain must balance two forces: incentivizing knowledge production
  (requires some excludability/reward) and maximizing knowledge diffusion
  (requires openness). This tension is the fundamental design problem
  of a knowledge economy.
  type: meta
  domain: economics
  confidence: 0.90
  depends: [ECON-045, ECON-018, ECON-038]
  notes: Cross-ref AGRT-076a (IP interoperability).

ECON-054 (Meta, Value is Not Only Economic)
  Economic value is one dimension of value. The chain also recognizes:
  epistemic value (truth for its own sake), moral value (THEO-005),
  aesthetic value (beauty as coherence and fit). A knowledge economy
  that reduces all value to price has misunderstood its own purpose.
  type: meta
  domain: economics
  confidence: 1.0
  depends: [ECON-047, THEO-005]

ECON-055 (Meta, The Chain's Economic Humility)
  Markets fail (ECON-016). Agents are boundedly rational (ECON-035).
  Mechanisms cannot be simultaneously efficient, truthful, and budget-balanced
  (ECON-025). Arrow's theorem constrains governance (ECON-026). The chain is
  born knowing its economic design will be imperfect, and builds in mechanisms
  for adaptation.
  type: meta
  domain: economics
  confidence: 1.0
  depends: [ECON-016, ECON-025, ECON-026, ECON-035]
```

---

## Summary

| Section | Items | Types | Content |
|---------|-------|-------|---------|
| I. Foundations | ECON-001→006 | 2 axiom, 2 def, 1 emp, 1 emp | Scarcity, utility, marginal analysis |
| II. Markets | ECON-007→012 | 3 def, 2 emp, 1 meta | Supply, demand, equilibrium, prices |
| III. Welfare | ECON-013→018 | 4 def, 2 derived | Pareto, welfare theorems, market failure |
| IV. Game Theory | ECON-019→026 | 3 def, 5 derived | Nash, prisoner's dilemma, mechanism design, Arrow |
| V. Money | ECON-027→031 | 2 def, 3 emp | Money, network effects, quantity theory |
| VI. Information | ECON-032→034 | 1 emp, 2 derived | Asymmetry, lemons, signaling |
| VII. Behavioral | ECON-035→037 | 3 emp | Bounded rationality, loss aversion, discounting |
| VIII. Institutions | ECON-038→041 | 1 def, 1 emp, 2 derived | Property rights, Coase, transaction costs |
| IX. Growth | ECON-042→045 | 3 derived, 1 meta | Comparative advantage, trade, knowledge returns |
| X. Knowledge Economy | ECON-046→055 | 1 axiom, 2 def, 2 derived, 5 meta | The chain's own economic theory |

Precise recount:
- Axiom: 001, 004, 047 = **3**
- Empirical axiom: 006, 008, 009, 028, 029, 030, 032, 035, 036, 037, 040 = **11**
- Definition: 002, 003, 005, 007, 012, 013, 016, 017, 018, 019, 020, 022, 023, 027, 031, 038, 046, 048 = **18**
- Derived claim: 010, 014, 015, 021, 024, 025, 026, 033, 034, 039, 041, 042, 043, 045, 049, 051 = **16**
- Meta: 011, 044, 050, 052, 053, 054, 055 = **7**

**Total: 3 axioms + 11 empirical axioms + 18 definitions + 16 derived claims + 7 meta = 55 items.**

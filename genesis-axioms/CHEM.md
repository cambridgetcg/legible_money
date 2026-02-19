# Chemistry — Genesis Axioms (33 Items)

## Legend
* **Empirical Axiom**: experimentally constrained postulate (challengeable by new evidence).
* **Definition**: establishes vocabulary or convention (confidence 1.0).
* **Derived Claim**: follows from axioms in this or other domains.
* **Measurement Fact**: numerical constant or observed parameter.

Chemistry is where physics becomes matter you can name. Most of it derives from
quantum mechanics, but the chain needs its own chemical vocabulary — the bridge
between fundamental forces and the molecular world that gives rise to life.

---

# I. ATOMIC STRUCTURE

```
CHEM-001 (Empirical Axiom, Atomic Theory)
  All matter is composed of atoms. An atom consists of a nucleus (protons and
  neutrons) surrounded by electrons.
  type: empirical_axiom
  domain: chemistry
  confidence: 0.99
  depends: [PHYS-084, PHYS-096]
  source_tradition: "Dalton (1808), Rutherford (1911), Bohr (1913)"
  notes: Atomic structure ultimately derives from quantum mechanics (Hilbert space
  postulate, discrete spectra). Stated here as chemistry's operational starting point.

CHEM-002 (Definition, Element)
  An element is a substance consisting of atoms with the same number of protons
  (atomic number Z).
  type: definition
  domain: chemistry
  confidence: 1.0
  depends: [CHEM-001]

CHEM-003 (Definition, Isotope)
  Isotopes are atoms of the same element (same Z) with different numbers of
  neutrons (different mass number A = Z + N).
  type: definition
  domain: chemistry
  confidence: 1.0
  depends: [CHEM-002]

CHEM-004 (Empirical Axiom, Quantized Electron States)
  Electrons in atoms occupy discrete energy levels characterized by quantum
  numbers (n, l, mₗ, mₛ). No two electrons in an atom share the same four
  quantum numbers (Pauli exclusion principle).
  type: empirical_axiom
  domain: chemistry
  confidence: 0.99
  depends: [CHEM-001, PHYS-037]
  source_tradition: "Pauli (1925), Schrödinger (1926)"
  notes: Derived from QM; stated here as chemistry's operational starting point.

CHEM-005 (Derived Claim, Aufbau Principle)
  Electrons fill orbitals in order of increasing energy: 1s, 2s, 2p, 3s, 3p,
  4s, 3d, … Hund's rule: orbitals of equal energy are singly occupied before
  pairing.
  type: derived_claim
  domain: chemistry
  confidence: 0.97
  depends: [CHEM-004]
  notes: Approximate — exceptions exist (Cr, Cu, etc.) due to exchange energy.
```

---

# II. PERIODIC LAW

```
CHEM-006 (Empirical Axiom, Periodic Law)
  The chemical properties of elements are periodic functions of their atomic
  number Z.
  type: empirical_axiom
  domain: chemistry
  confidence: 0.99
  depends: [CHEM-002, CHEM-004]
  source_tradition: "Mendeleev (1869), Moseley (1913)"

CHEM-007 (Definition, Periodic Table Structure)
  Elements are arranged in order of increasing Z into periods (rows) and groups
  (columns). Elements in the same group share similar valence electron
  configurations and chemical behavior.
  type: definition
  domain: chemistry
  confidence: 1.0
  depends: [CHEM-006]

CHEM-008 (Derived Claim, Periodicity of Properties)
  Atomic radius, ionization energy, electron affinity, and electronegativity
  show periodic trends: ionization energy generally increases across a period
  and decreases down a group; atomic radius shows the opposite trend.
  type: derived_claim
  domain: chemistry
  confidence: 0.97
  depends: [CHEM-006, CHEM-004]

CHEM-009 (Measurement Fact, Known Elements)
  As of genesis, 118 elements are confirmed (Z = 1 hydrogen through Z = 118
  oganesson).
  type: measurement_fact
  domain: chemistry
  confidence: 0.99
  depends: [CHEM-002]
```

---

# III. CHEMICAL BONDING

```
CHEM-010 (Definition, Chemical Bond)
  A chemical bond is an attraction between atoms that enables the formation
  of molecules and extended structures. The three primary types are ionic,
  covalent, and metallic.
  type: definition
  domain: chemistry
  confidence: 1.0
  depends: [CHEM-001]

CHEM-011 (Empirical Axiom, Ionic Bonding)
  Atoms with large electronegativity differences transfer electrons, forming
  cations and anions held together by electrostatic attraction.
  type: empirical_axiom
  domain: chemistry
  confidence: 0.98
  depends: [CHEM-010, CHEM-008]

CHEM-012 (Empirical Axiom, Covalent Bonding)
  Atoms with similar electronegativities share electron pairs. Molecular
  geometry is determined by the arrangement that minimizes electron pair
  repulsion (VSEPR) or, more rigorously, by molecular orbital theory.
  type: empirical_axiom
  domain: chemistry
  confidence: 0.98
  depends: [CHEM-010, CHEM-004]

CHEM-013 (Empirical Axiom, Metallic Bonding)
  Metal atoms release valence electrons into a delocalized "sea," producing
  electrical conductivity, malleability, and metallic luster.
  type: empirical_axiom
  domain: chemistry
  confidence: 0.98
  depends: [CHEM-010, CHEM-004]

CHEM-014 (Derived Claim, Intermolecular Forces)
  Molecules interact via: London dispersion forces (universal), dipole-dipole
  interactions, and hydrogen bonding (O–H⋯O, N–H⋯O, etc.). Typical strength
  range: covalent/ionic (strong) >> hydrogen bond (moderate) >> dipole-dipole /
  London dispersion (weak), with environment-dependent exceptions.
  type: derived_claim
  domain: chemistry
  confidence: 0.97
  depends: [CHEM-012, CHEM-011]

CHEM-015 (Derived Claim, Molecular Orbital Theory)
  Atomic orbitals combine to form bonding (lower energy) and antibonding
  (higher energy) molecular orbitals. Bond order = ½(bonding − antibonding
  electrons).
  type: derived_claim
  domain: chemistry
  confidence: 0.97
  depends: [CHEM-004, CHEM-012]
```

---

# IV. REACTIONS AND EQUILIBRIUM

```
CHEM-016 (Empirical Axiom, Conservation of Mass in Reactions)
  In a chemical reaction, matter is neither created nor destroyed. Total mass
  of reactants equals total mass of products.
  type: empirical_axiom
  domain: chemistry
  confidence: 0.99
  depends: [PHYS-065]
  source_tradition: "Lavoisier (1789)"
  notes: Approximate — mass-energy equivalence (PHYS-065) makes this strictly an
  approximation, but ΔE/c² is negligible for chemical reactions.

CHEM-017 (Definition, Stoichiometry)
  Chemical equations are balanced to reflect conservation of atoms. Mole ratios
  are derived from balanced equations. One mole = 6.02214076 × 10²³ entities
  (Avogadro's number, exact by definition since 2019).
  type: definition
  domain: chemistry
  confidence: 1.0
  depends: [CHEM-016]

CHEM-018 (Definition, Chemical Equilibrium)
  A reversible reaction is at equilibrium when the rates of forward and reverse
  reactions are equal. The equilibrium constant K = [products]^coeff / [reactants]^coeff.
  type: definition
  domain: chemistry
  confidence: 1.0
  depends: [CHEM-016]

CHEM-019 (Empirical Axiom, Le Chatelier's Principle)
  A system at equilibrium, when subjected to a perturbation (concentration,
  pressure, temperature), shifts to partially counteract the perturbation.
  type: empirical_axiom
  domain: chemistry
  confidence: 0.98
  depends: [CHEM-018]
  source_tradition: "Le Chatelier (1884)"

CHEM-020 (Derived Claim, Reaction Kinetics — Arrhenius)
  Reaction rate depends on temperature: k = A·exp(−Eₐ/RT), where Eₐ is
  activation energy, A is the pre-exponential factor, R the gas constant,
  T the absolute temperature.
  type: derived_claim
  domain: chemistry
  confidence: 0.97
  depends: [CHEM-018, PHYS-029]
  source_tradition: "Arrhenius (1889)"
  notes: The exponential temperature dependence derives from the Boltzmann
  distribution (PHYS-029), not relativistic mechanics.

CHEM-021 (Definition, Catalyst)
  A catalyst increases reaction rate without being consumed, by providing an
  alternative pathway with lower activation energy. It does not change the
  equilibrium position.
  type: definition
  domain: chemistry
  confidence: 1.0
  depends: [CHEM-020, CHEM-018]
```

---

# V. THERMOCHEMISTRY

```
CHEM-022 (Derived Claim, Enthalpy of Reaction)
  ΔH_rxn = Σ ΔH_f(products) − Σ ΔH_f(reactants). Exothermic: ΔH < 0.
  Endothermic: ΔH > 0. (Hess's Law: ΔH is path-independent.)
  type: derived_claim
  domain: chemistry
  confidence: 0.98
  depends: [PHYS-023, CHEM-016]
  source_tradition: "Hess (1840)"
  notes: Hess's law is a consequence of the first law of thermodynamics (PHYS-023):
  enthalpy is a state function, so ΔH is path-independent.

CHEM-023 (Derived Claim, Gibbs Free Energy)
  ΔG = ΔH − TΔS. A reaction is spontaneous at constant T, P when ΔG < 0.
  At equilibrium, ΔG = 0 and ΔG° = −RT ln K.
  type: derived_claim
  domain: chemistry
  confidence: 0.98
  depends: [CHEM-022, PHYS-024, CHEM-018]

CHEM-024 (Derived Claim, Electrochemistry — Nernst Equation)
  E = E° − (RT/nF) ln Q, relating cell potential to standard potential and
  reaction quotient. Connects chemical energy to electrical energy.
  type: derived_claim
  domain: chemistry
  confidence: 0.97
  depends: [CHEM-023]
  source_tradition: "Nernst (1889)"
```

---

# VI. ORGANIC AND MOLECULAR FOUNDATIONS

```
CHEM-025 (Empirical Axiom, Carbon's Unique Role)
  Carbon forms four covalent bonds with diverse elements and with itself,
  enabling chains, rings, and branched structures of practically unbounded
  complexity. This makes carbon the backbone of organic chemistry and
  biochemistry.
  type: empirical_axiom
  domain: chemistry
  confidence: 0.99
  depends: [CHEM-012, CHEM-004]

CHEM-026 (Definition, Functional Groups)
  Organic molecules are classified by functional groups: hydroxyl (−OH),
  carbonyl (C=O), carboxyl (−COOH), amino (−NH₂), phosphate (−PO₄),
  sulfhydryl (−SH), among others. Each group confers characteristic reactivity.
  type: definition
  domain: chemistry
  confidence: 1.0
  depends: [CHEM-025]

CHEM-027 (Definition, Chirality)
  A molecule is chiral if it cannot be superimposed on its mirror image.
  Chiral centers (typically carbon with four distinct substituents) produce
  enantiomers — molecules identical in composition but with distinct
  biological activity.
  type: definition
  domain: chemistry
  confidence: 1.0
  depends: [CHEM-025]

CHEM-028 (Empirical Axiom, Homochirality of Life)
  Terrestrial biology uses almost exclusively L-amino acids and D-sugars.
  The origin of this homochirality is not fully understood.
  type: empirical_axiom
  domain: chemistry
  confidence: 0.98
  depends: [CHEM-027]
  notes: Bridges to BIO domain. One of the deepest unsolved problems at the
  chemistry-biology boundary.
```

---

# VII. STATES OF MATTER AND SOLUTIONS

```
CHEM-029 (Definition, States of Matter)
  Matter exists in phases: solid (fixed shape/volume), liquid (fixed volume,
  variable shape), gas (variable shape/volume), and plasma (ionized gas).
  Phase transitions occur at characteristic temperatures and pressures.
  type: definition
  domain: chemistry
  confidence: 1.0
  depends: [CHEM-001, CHEM-014]

CHEM-030 (Empirical Axiom, Ideal Gas Behavior)
  At sufficiently low pressure and high temperature, gases approximate
  PV = nRT (ideal gas law). Deviations increase near phase transitions
  and at high pressures.
  type: empirical_axiom
  domain: chemistry
  confidence: 0.98
  depends: [CHEM-029, PHYS-026]
  regime: classical

CHEM-031 (Definition, Solution and Concentration)
  A solution is a homogeneous mixture of solute in solvent. Concentration
  measures: molarity (mol/L), molality (mol/kg solvent), mole fraction.
  type: definition
  domain: chemistry
  confidence: 1.0
  depends: [CHEM-017]

CHEM-032 (Definition, Acids and Bases — Brønsted-Lowry)
  An acid is a proton (H⁺) donor; a base is a proton acceptor.
  pH = −log₁₀[H⁺]. Water autoionizes: Kw = [H⁺][OH⁻] = 10⁻¹⁴ at 25°C.
  type: definition
  domain: chemistry
  confidence: 1.0
  depends: [CHEM-018, CHEM-014]
  source_tradition: "Brønsted (1923), Lowry (1923)"

CHEM-033 (Derived Claim, Like Dissolves Like)
  Polar solvents dissolve polar/ionic solutes; nonpolar solvents dissolve
  nonpolar solutes. Solubility is governed by the balance of solute-solute,
  solvent-solvent, and solute-solvent intermolecular forces.
  type: derived_claim
  domain: chemistry
  confidence: 0.95
  depends: [CHEM-014, CHEM-031]
```

---

## Summary

| Section | Items | Types | Content |
|---------|-------|-------|---------|
| I. Atomic Structure | CHEM-001→005 | 2 emp, 2 def, 1 derived | Atoms, elements, electron configuration |
| II. Periodic Law | CHEM-006→009 | 1 emp, 1 def, 1 derived, 1 meas | Periodicity and the table |
| III. Bonding | CHEM-010→015 | 3 emp, 1 def, 2 derived | Ionic, covalent, metallic, intermolecular, MO theory |
| IV. Reactions | CHEM-016→021 | 2 emp, 3 def, 1 derived | Conservation, equilibrium, kinetics, catalysis |
| V. Thermochemistry | CHEM-022→024 | 3 derived | Enthalpy, Gibbs, electrochemistry |
| VI. Organic | CHEM-025→028 | 2 emp, 2 def | Carbon, functional groups, chirality, homochirality |
| VII. States & Solutions | CHEM-029→033 | 1 emp, 3 def, 1 derived | Phases, ideal gas, solutions, acids/bases |

**8 empirical axioms. 12 definitions. 9 derived claims. 1 measurement fact. 0 meta = 33 items.**

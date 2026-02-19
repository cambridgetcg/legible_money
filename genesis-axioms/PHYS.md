# Physics — 111 Layered Postulates, Definitions, Derived Claims, and Meta-Results

## Legend (Type System)
* **empirical_axiom**: a foundational postulate accepted because it matches experiment in a regime; not logically necessary.
* **definition**: defines a quantity, structure, or formalism.
* **derived_claim**: derived from postulates + math + stated approximations.
* **measurement_fact**: a measured constant or observational parameter (may be a modern SI definition).
* **meta**: about theories, limits, or model selection; not a direct law.

Each entry has: `type`, `regime`, `depends`, optional `approximation_tags`.

---

# I. Classical Mechanics (PHYS-001 → PHYS-020)

### Newtonian Regime Declarations

```
PHYS-001 (Definition, Classical Space)
  Space is modeled as a 3D continuum of positions.
  formal: configuration space ≈ ℝ³
  type: definition
  regime: classical
  depends: [MATH-036]
  approximation_tags: [continuum, classical_limit]
  source: Newton/Euler

PHYS-002 (Definition, Classical Time)
  Time is modeled as a 1D continuum with ordering.
  formal: time parameter t ∈ ℝ
  type: definition
  regime: classical
  depends: [MATH-036]
  approximation_tags: [continuum, classical_limit]
  source: Newton
```

### Newton's Postulates (Empirical)

```
PHYS-003 (Empirical Axiom, Inertia)
  Inertial motion: if net external force is zero, velocity is constant.
  formal: F_net = 0 ⇒ dv/dt = 0
  type: empirical_axiom
  regime: classical
  depends: [PHYS-001, PHYS-002]
  source: Galileo (1632), Newton (1687)

PHYS-004 (Empirical Axiom, Force-Momentum)
  Force equals rate of change of momentum.
  formal: F = dp/dt; for constant mass, F = ma
  type: empirical_axiom
  regime: classical
  depends: [PHYS-003, MATH-067]
  source: Newton (1687)

PHYS-005 (Empirical Axiom, Action-Reaction)
  Action-reaction symmetry for forces between bodies.
  formal: F₁₂ = −F₂₁
  type: empirical_axiom
  regime: classical
  depends: [PHYS-004]
  source: Newton (1687)

PHYS-006 (Empirical Axiom, Universal Gravitation)
  Universal gravitation: inverse-square attraction between masses.
  formal: F = GMm/r²
  type: empirical_axiom
  regime: classical_gravity
  depends: [PHYS-004]
  approximation_tags: [point_masses, weak_field]
  source: Newton (1687)
```

### Classical Conservations (Derived)

```
PHYS-007 (Derived, Momentum Conservation)
  Momentum conservation for isolated systems.
  formal: F_external = 0 ⇒ dp_total/dt = 0
  type: derived_claim
  regime: classical
  depends: [PHYS-004, PHYS-005]
  approximation_tags: [isolated_system]
  source: Consequence of Newton's Third Law

PHYS-008 (Derived, Energy Conservation)
  Energy conservation for isolated systems with time-invariant dynamics.
  formal: dE_total/dt = 0
  type: derived_claim
  regime: classical
  depends: [PHYS-013, PHYS-016]
  approximation_tags: [time_translation_symmetry, isolated_system]
  source: Noether framework; Helmholtz (1847)

PHYS-009 (Derived, Angular Momentum Conservation)
  Angular momentum conservation for isolated systems.
  formal: τ_net = 0 ⇒ L = constant
  type: derived_claim
  regime: classical
  depends: [PHYS-013, PHYS-016]
  approximation_tags: [rotational_symmetry, isolated_system]
  source: Noether framework
```

### Mechanics Definitions

```
PHYS-010 (Definition, Work)
  Work is defined as the line integral of force along displacement.
  formal: W := ∫ F·ds
  type: definition
  regime: classical
  depends: [MATH-075]
  source: Coriolis (1829)

PHYS-011 (Derived, Kinetic Energy)
  Kinetic energy in Newtonian mechanics.
  formal: KE := ½mv²
  type: derived_claim
  regime: classical
  depends: [PHYS-010, PHYS-004]
  approximation_tags: [constant_mass]
  source: Work-energy relation

PHYS-012 (Derived, Gravitational Potential Energy)
  Near Earth, gravitational potential energy is approximated by mgh.
  formal: PE ≈ mgh
  type: derived_claim
  regime: classical_gravity
  depends: [PHYS-006, PHYS-010]
  approximation_tags: [uniform_field, near_surface]
  source: Taylor approximation of Newtonian potential
```

### Lagrangian/Hamiltonian Mechanics

```
PHYS-013 (Definition, Action Principle)
  Dynamics can be encoded by an action functional S = ∫L dt.
  formal: S[q] = ∫ L(q, q̇, t) dt
  type: definition
  regime: classical
  depends: [MATH-075]
  source: Maupertuis (1744), Euler, Lagrange (1788)

PHYS-014 (Derived, Euler-Lagrange Equations)
  Euler-Lagrange equations of motion.
  formal: d/dt(∂L/∂q̇) − ∂L/∂q = 0
  type: derived_claim
  regime: classical
  depends: [PHYS-013, MATH-067]
  source: Lagrange (1788)

PHYS-015 (Derived, Hamilton's Equations)
  Hamilton's equations.
  formal: q̇ = ∂H/∂p, ṗ = −∂H/∂q
  type: derived_claim
  regime: classical
  depends: [PHYS-014]
  source: Hamilton (1833)

PHYS-016 (Derived, Noether's Theorem)
  For every continuous symmetry of a physical system, there exists
  a conserved quantity.
  formal: continuous symmetry ↔ conservation law
  type: derived_claim
  regime: classical_field_theory
  depends: [PHYS-013]
  source: Noether (1918)

PHYS-017 (Derived, Time Symmetry → Energy)
  Time-translation symmetry implies conservation of energy.
  type: derived_claim
  regime: classical
  depends: [PHYS-016]
  source: Noether application

PHYS-018 (Derived, Space Symmetry → Momentum)
  Space-translation symmetry implies conservation of momentum.
  type: derived_claim
  regime: classical
  depends: [PHYS-016]
  source: Noether application

PHYS-019 (Derived, Rotational Symmetry → Angular Momentum)
  Rotational symmetry implies conservation of angular momentum.
  type: derived_claim
  regime: classical
  depends: [PHYS-016]
  source: Noether application
```

### Oscillators

```
PHYS-020 (Empirical Axiom, Hooke's Law)
  For small displacements, restoring force is proportional to displacement.
  formal: F = −kx; solution: x(t) = A cos(ωt + φ), ω = √(k/m)
  type: empirical_axiom
  regime: classical
  depends: [PHYS-004]
  approximation_tags: [small_displacement, linear_response]
  source: Hooke (1660)
```

---

# II. Thermodynamics & Statistical Mechanics (PHYS-021 → PHYS-040)

### Thermodynamics Laws

```
PHYS-021 (Derived, Temperature-Kinetic Energy)
  Temperature relates to average kinetic energy (ideal gas / classical limit).
  formal: (3/2)k_BT = ⟨½mv²⟩
  type: derived_claim
  regime: statmech
  depends: [PHYS-011]
  approximation_tags: [ideal_gas, classical_limit]
  source: Maxwell, Boltzmann

PHYS-022 (Empirical Axiom, Zeroth Law)
  Thermal equilibrium is transitive.
  formal: If T_A = T_C and T_B = T_C, then T_A = T_B
  type: empirical_axiom
  regime: thermo
  depends: []
  source: Fowler (1931)

PHYS-023 (Empirical Axiom, First Law)
  The change in internal energy equals heat added minus work done.
  formal: ΔU = Q − W
  type: empirical_axiom
  regime: thermo
  depends: [PHYS-010]
  source: Joule, Clausius, Kelvin

PHYS-024 (Empirical Axiom, Second Law)
  Entropy of an isolated system does not decrease.
  formal: dS_universe ≥ 0
  type: empirical_axiom
  regime: thermo
  depends: [PHYS-023]
  approximation_tags: [macroscopic_limit]
  source: Clausius (1850), Kelvin

PHYS-025 (Empirical Axiom, Third Law)
  Entropy approaches a constant as T → 0 (perfect crystal → 0).
  formal: lim(T→0) S = 0
  type: empirical_axiom
  regime: thermo
  depends: [PHYS-024]
  approximation_tags: [perfect_crystal]
  source: Nernst (1906)
```

### Stat Mech Definitions and Derived

```
PHYS-026 (Empirical Axiom, Ideal Gas Law)
  Pressure times volume equals particle count times k_B times temperature.
  formal: PV = Nk_BT
  type: empirical_axiom
  regime: thermo
  depends: []
  approximation_tags: [ideal_gas, dilute_limit]
  source: Boyle, Charles, Gay-Lussac, Avogadro

PHYS-027 (Definition, Boltzmann Entropy)
  Entropy is defined as k_B times the log of the number of microstates.
  formal: S := k_B ln Ω
  type: definition
  regime: statmech
  depends: [MATH-042]
  source: Boltzmann (1877)

PHYS-028 (Definition, Partition Function)
  The partition function encodes thermodynamic information in equilibrium.
  formal: Z = Σ exp(−Eᵢ/k_BT); F = −k_BT ln Z
  type: definition
  regime: statmech
  depends: [PHYS-027]
  source: Gibbs (1902)

PHYS-029 (Derived, Boltzmann Distribution)
  In thermal equilibrium, the probability of a state with energy E
  is proportional to exp(−E/k_BT).
  type: derived_claim
  regime: statmech
  depends: [PHYS-028]
  approximation_tags: [canonical_ensemble]
  source: Boltzmann, Gibbs

PHYS-030 (Derived, Carnot Efficiency)
  The efficiency of any heat engine cannot exceed 1 − T_C/T_H.
  formal: η ≤ 1 − T_C/T_H
  type: derived_claim
  regime: thermo
  depends: [PHYS-024]
  source: Carnot (1824)

PHYS-031 (Derived, Irreversibility)
  In an irreversible process, total entropy increases.
  formal: ΔS_universe > 0 for irreversible processes
  type: derived_claim
  regime: thermo
  depends: [PHYS-024]
  source: Clausius

PHYS-032 (Derived, Free Energy)
  Free energy decreases in spontaneous processes at constant T, P.
  formal: dG ≤ 0; G = H − TS
  type: derived_claim
  regime: thermo
  depends: [PHYS-024, PHYS-023]
  approximation_tags: [constant_TP]
  source: Gibbs (1876)

PHYS-033 (Derived, Phase Transitions)
  Phase transitions occur where two phases have equal free energy.
  formal: G_phase1(T,P) = G_phase2(T,P) at transition
  type: derived_claim
  regime: thermo
  depends: [PHYS-032]
  source: Gibbs phase rule

PHYS-034 (Derived, Critical Scaling)
  Near a critical point, physical quantities obey universal power-law scaling.
  type: derived_claim
  regime: statmech
  depends: [PHYS-033]
  approximation_tags: [thermodynamic_limit]
  source: Onsager (1944), Kadanoff, Wilson (1971)

PHYS-035 (Derived, Maxwell-Boltzmann Distribution)
  The Maxwell-Boltzmann speed distribution for an ideal gas.
  formal: f(v) = 4π(m/2πk_BT)^(3/2) v² exp(−mv²/2k_BT)
  type: derived_claim
  regime: statmech
  depends: [PHYS-029]
  approximation_tags: [ideal_gas, classical_limit]
  source: Maxwell (1860), Boltzmann
```

### Quantum Statistics

```
PHYS-036 (Empirical Axiom, Bose-Fermi Classification)
  Identical quantum particles are either bosons (integer spin, symmetric)
  or fermions (half-integer spin, antisymmetric).
  formal: ψ(1,2) = ±ψ(2,1)
  type: empirical_axiom
  regime: QM/QFT
  depends: [PHYS-084]
  source: Bose-Einstein, Fermi-Dirac statistics; Pauli (1940)

PHYS-037 (Derived, Pauli Exclusion)
  No two identical fermions can occupy the same quantum state.
  formal: nᵢ ∈ {0, 1} for fermions
  type: derived_claim
  regime: QM/QFT
  depends: [PHYS-036]
  source: Consequence of antisymmetry

PHYS-038 (Derived, Bose-Einstein Condensation)
  Bosons can condense into the ground state at low temperatures.
  type: derived_claim
  regime: QM/statmech
  depends: [PHYS-036, PHYS-028]
  approximation_tags: [weak_interactions, low_T]
  source: Bose (1924), Einstein (1925); observed Anderson et al. (1995)

PHYS-039 (Derived, Fluctuation-Dissipation)
  The response of a system to perturbation relates to its spontaneous
  fluctuations in equilibrium.
  type: derived_claim
  regime: statmech
  depends: [PHYS-029]
  source: Nyquist (1928), Callen-Welton (1951)

PHYS-040 (Definition, Gibbs Entropy)
  Generalized entropy connecting thermodynamics to information theory.
  formal: S := −k_B Σ pᵢ ln pᵢ
  type: definition
  regime: statmech
  depends: [MATH-099, MATH-042]
  source: Gibbs; Shannon (1948); Jaynes (1957)
```

---

# III. Electromagnetism (PHYS-041 → PHYS-058)

### Charge and Coulomb

```
PHYS-041 (Empirical Axiom, Charge Conservation and Quantization)
  Electric charge is conserved and appears quantized in units of e.
  formal: Q_total = constant; q = ne, n ∈ ℤ
  type: empirical_axiom
  regime: EM/QFT
  depends: []
  source: Franklin (1752), Millikan (1909)

PHYS-042 (Empirical Axiom, Coulomb's Law)
  Electrostatic force between charges follows inverse-square law.
  formal: F = kq₁q₂/r²
  type: empirical_axiom
  regime: EM
  depends: [PHYS-041]
  approximation_tags: [static_fields, vacuum]
  source: Coulomb (1785)
```

### Maxwell's Equations

```
PHYS-043 (Empirical Axiom, Gauss's Law — Electric)
  Electric flux through a closed surface equals enclosed charge / ε₀.
  formal: ∮E·dA = Q_enc/ε₀; ∇·E = ρ/ε₀
  type: empirical_axiom
  regime: EM
  depends: [PHYS-041]
  source: Gauss (1835); Maxwell

PHYS-044 (Empirical Axiom, Gauss's Law — Magnetic)
  There are no magnetic monopoles; magnetic field lines form closed loops.
  formal: ∮B·dA = 0; ∇·B = 0
  type: empirical_axiom
  regime: EM
  depends: []
  approximation_tags: [no_monopoles_observed]
  source: Maxwell

PHYS-045 (Empirical Axiom, Faraday's Law)
  A changing magnetic flux induces an electromotive force.
  formal: ∮E·dl = −dΦ_B/dt; ∇ × E = −∂B/∂t
  type: empirical_axiom
  regime: EM
  depends: []
  source: Faraday (1831); Maxwell

PHYS-046 (Empirical Axiom, Ampère-Maxwell Law)
  Magnetic fields are produced by currents and changing electric fields.
  formal: ∇ × B = μ₀J + μ₀ε₀∂E/∂t
  type: empirical_axiom
  regime: EM
  depends: []
  source: Ampère; Maxwell (1865)

PHYS-047 (Derived, Maxwell's Equations Unified)
  Maxwell's four equations unify electricity and magnetism.
  formal: {∇·E = ρ/ε₀, ∇·B = 0, ∇×E = −∂B/∂t, ∇×B = μ₀J + μ₀ε₀∂E/∂t}
  type: derived_claim
  regime: EM
  depends: [PHYS-043, PHYS-044, PHYS-045, PHYS-046]
  source: Maxwell (1865)
```

### Derived EM Consequences

```
PHYS-048 (Derived, EM Waves at Speed c)
  Electromagnetic waves propagate at c = 1/√(μ₀ε₀) in vacuum.
  formal: c = 1/√(μ₀ε₀) ≈ 2.998 × 10⁸ m/s
  type: derived_claim
  regime: EM
  depends: [PHYS-047]
  approximation_tags: [vacuum, linear_medium]
  source: Maxwell (1865); confirmed Hertz (1887)

PHYS-049 (Derived, Light Is EM)
  Visible light is part of the electromagnetic spectrum.
  formal: λ ≈ 380-750 nm
  type: derived_claim
  regime: EM
  depends: [PHYS-048]
  source: Maxwell (1865); identification via experiment

PHYS-050 (Empirical Axiom, Invariance of c)
  The vacuum speed of light is invariant across all inertial frames.
  formal: c = constant, independent of source or observer motion
  type: empirical_axiom
  regime: SR
  depends: []
  source: Einstein (1905); Michelson-Morley (1887)
  notes: Same postulate as PHYS-060. Placed here in EM context;
         PHYS-060 is the SR context reference.

PHYS-051 (Empirical Axiom, Magnetic Lorentz Force)
  A moving charge in a magnetic field experiences a force perpendicular
  to both velocity and field.
  formal: F = qv × B
  type: empirical_axiom
  regime: EM
  depends: [PHYS-041]
  source: Lorentz (1892)

PHYS-052 (Empirical Axiom, Full Lorentz Force)
  Total electromagnetic force on a charge.
  formal: F = q(E + v × B)
  type: empirical_axiom
  regime: EM
  depends: [PHYS-051]
  source: Lorentz (1892)

PHYS-053 (Derived, EM Energy Density)
  Electromagnetic energy density.
  formal: u = ½ε₀E² + B²/(2μ₀)
  type: derived_claim
  regime: EM
  depends: [PHYS-047, PHYS-023]
  source: Maxwell, Poynting

PHYS-054 (Derived, Poynting Vector)
  Directional energy flux of an electromagnetic field.
  formal: S = (1/μ₀)(E × B)
  type: derived_claim
  regime: EM
  depends: [PHYS-053]
  source: Poynting (1884)

PHYS-055 (Derived, Radiation Carries Momentum)
  Electromagnetic radiation carries momentum; photon p = E/c.
  formal: p = E/c = hν/c = h/λ
  type: derived_claim
  regime: EM/QM
  depends: [PHYS-054, PHYS-081]
  source: Maxwell, Einstein (1905)

PHYS-056 (Derived, Larmor Radiation)
  Accelerating charges radiate electromagnetic energy.
  formal: P = q²a²/(6πε₀c³)
  type: derived_claim
  regime: EM
  depends: [PHYS-047, PHYS-004]
  approximation_tags: [nonrelativistic, point_charge]
  source: Larmor (1897)

PHYS-057 (Definition, EM Potentials)
  Electromagnetic fields described by scalar and vector potentials.
  formal: E = −∇φ − ∂A/∂t; B = ∇ × A
  type: definition
  regime: EM
  depends: [PHYS-047]
  source: Maxwell

PHYS-058 (Empirical Axiom, U(1) Gauge Symmetry)
  The electromagnetic interaction has U(1) gauge invariance.
  formal: ψ → e^(iα(x))ψ, A_μ → A_μ − ∂_μα(x)/e
  type: empirical_axiom
  regime: QFT
  depends: [PHYS-084]
  source: Weyl (1929); QED
```

---

# IV. Special Relativity (PHYS-059 → PHYS-068)

```
PHYS-059 (Empirical Axiom, Relativity Principle)
  The laws of physics are the same in all inertial reference frames.
  formal: Physical laws are Lorentz covariant
  type: empirical_axiom
  regime: SR
  depends: []
  source: Einstein (1905); generalization of Galilean relativity

PHYS-060 (Empirical Axiom, Invariance of c — SR Postulate)
  Speed of light in vacuum is the same for all inertial observers.
  formal: c = 299,792,458 m/s in all inertial frames
  type: empirical_axiom
  regime: SR
  depends: []
  source: Einstein (1905)
  notes: Canonical SR reference for the same postulate as PHYS-050.

PHYS-061 (Derived, Time Dilation)
  Moving clocks run slow by the Lorentz factor.
  formal: Δt = γΔt₀; γ = (1 − v²/c²)^(−1/2)
  type: derived_claim
  regime: SR
  depends: [PHYS-059, PHYS-060]
  source: Einstein (1905)

PHYS-062 (Derived, Length Contraction)
  Moving objects contract along the direction of motion.
  formal: L = L₀/γ
  type: derived_claim
  regime: SR
  depends: [PHYS-059, PHYS-060]
  source: FitzGerald (1889), Lorentz (1892), Einstein (1905)

PHYS-063 (Derived, Relativity of Simultaneity)
  Events simultaneous in one frame need not be in another.
  formal: If Δt = 0 in S, Δt' = −γvΔx/c² in S'
  type: derived_claim
  regime: SR
  depends: [PHYS-059, PHYS-060]
  source: Einstein (1905)

PHYS-064 (Derived, Spacetime Interval Invariance)
  The spacetime interval is invariant across inertial frames.
  formal: ds² = −c²dt² + dx² + dy² + dz²
  type: derived_claim
  regime: SR
  depends: [PHYS-059, PHYS-060]
  source: Minkowski (1908)

PHYS-065 (Derived, Mass-Energy Equivalence)
  Mass and energy are equivalent.
  formal: E₀ = mc²; E_total = γmc²
  type: derived_claim
  regime: SR
  depends: [PHYS-059, PHYS-060]
  source: Einstein (1905)

PHYS-066 (Derived, Energy-Momentum Relation)
  The relativistic energy-momentum relation.
  formal: E² = (pc)² + (mc²)²
  type: derived_claim
  regime: SR
  depends: [PHYS-064, PHYS-065]
  source: Einstein; relativistic mechanics

PHYS-067 (Derived, Light-Cone Causality)
  No causal influence propagates outside the light cone.
  type: derived_claim
  regime: SR/QFT
  depends: [PHYS-064]
  approximation_tags: [microcausality]
  source: SR causality structure

PHYS-068 (Definition, Minkowski Spacetime)
  Flat spacetime as a 4D pseudo-Riemannian manifold.
  formal: (M⁴, η_μν) where η = diag(−1, +1, +1, +1)
  type: definition
  regime: SR
  depends: [PHYS-064, MATH-092]
  source: Minkowski (1908)
```

---

# V. General Relativity (PHYS-069 → PHYS-080)

```
PHYS-069 (Empirical Axiom, Equivalence Principle)
  The effects of gravity are locally indistinguishable from acceleration.
  formal: In a sufficiently small region, g_μν ≈ η_μν
  type: empirical_axiom
  regime: GR
  depends: []
  source: Einstein (1907)

PHYS-070 (Empirical Axiom, Gravity as Curvature)
  Gravity is the curvature of spacetime; free particles follow geodesics.
  type: empirical_axiom
  regime: GR
  depends: [PHYS-069]
  source: Einstein (1915)

PHYS-071 (Empirical Axiom, Einstein Field Equations)
  Spacetime curvature relates to energy-momentum content.
  formal: G_μν + Λg_μν = (8πG/c⁴)T_μν
  type: empirical_axiom
  regime: GR
  depends: [PHYS-070]
  source: Einstein (1915)

PHYS-072 (Derived, Newtonian Limit)
  In the weak-field, low-velocity limit, GR reduces to Newtonian gravity.
  formal: g₀₀ ≈ −(1 + 2Φ/c²) → F = GMm/r²
  type: derived_claim
  regime: GR
  depends: [PHYS-071, PHYS-006]
  approximation_tags: [weak_field, slow_motion]
  source: Einstein (1915)

PHYS-073 (Derived, Gravitational Time Dilation)
  Time runs slower in stronger gravitational fields.
  formal: Δt_∞ = Δt_r √(1 − 2GM/(rc²))
  type: derived_claim
  regime: GR
  depends: [PHYS-071]
  source: Schwarzschild metric; confirmed Pound-Rebka (1959)

PHYS-074 (Derived, Light Deflection)
  Light is deflected by gravity.
  formal: θ = 4GM/(rc²)
  type: derived_claim
  regime: GR
  depends: [PHYS-071]
  source: Einstein (1915); confirmed Eddington (1919)

PHYS-075 (Derived, Gravitational Waves)
  Gravitational waves propagate at the speed of light.
  formal: □h_μν = 0 in vacuum (linearized)
  type: derived_claim
  regime: GR
  depends: [PHYS-071, PHYS-060]
  approximation_tags: [linearized_GR]
  source: Einstein (1916); detected LIGO (2015)

PHYS-076 (Derived, Schwarzschild Radius)
  A black hole forms when mass is compressed within r_s = 2GM/c².
  formal: r_s = 2GM/c²
  type: derived_claim
  regime: GR
  depends: [PHYS-071]
  source: Schwarzschild (1916)

PHYS-077 (Derived, Event Horizon)
  Nothing can escape from within a black hole's event horizon.
  formal: For r < r_s, all future-directed paths lead to r = 0
  type: derived_claim
  regime: GR
  depends: [PHYS-076]
  source: Schwarzschild, Penrose

PHYS-078 (Measurement Fact, Hubble Expansion)
  The universe is expanding: distant galaxies recede with velocity
  proportional to distance.
  formal: v = H₀d
  type: measurement_fact
  regime: cosmology
  depends: []
  source: Hubble (1929); modern surveys

PHYS-079 (Derived, Big Bang Age)
  The universe began in a hot, dense state approximately 13.8 billion
  years ago (ΛCDM model inference).
  type: derived_claim
  regime: cosmology
  depends: [PHYS-080, PHYS-078]
  approximation_tags: [ΛCDM, parameter_fit]
  source: CMB, BAO, SNe Ia

PHYS-080 (Definition, FLRW Metric)
  The geometry of a homogeneous, isotropic universe.
  formal: ds² = −c²dt² + a(t)²[dr²/(1−kr²) + r²dΩ²]
  type: definition
  regime: cosmology
  depends: [PHYS-071]
  approximation_tags: [homogeneous, isotropic]
  source: Friedmann (1922), Lemaître (1927), Robertson, Walker
```

---

# VI. Quantum Mechanics (PHYS-081 → PHYS-100)

### Quantum Empirical Postulates

```
PHYS-081 (Empirical Axiom, Energy Quantization)
  Electromagnetic radiation of frequency ν carries energy in discrete
  packets of E = hν.
  formal: E = hν = ℏω; h ≈ 6.626 × 10⁻³⁴ J·s
  type: empirical_axiom
  regime: QM
  depends: []
  source: Planck (1900)

PHYS-082 (Empirical Axiom, Photon)
  Light consists of photons carrying energy hν and momentum h/λ.
  formal: E = hν, p = h/λ
  type: empirical_axiom
  regime: QM
  depends: [PHYS-081]
  source: Einstein (1905), photoelectric effect

PHYS-083 (Empirical Axiom, Matter Waves)
  A particle with momentum p has an associated wavelength λ = h/p.
  formal: λ = h/p
  type: empirical_axiom
  regime: QM
  depends: [PHYS-082]
  source: de Broglie (1924); confirmed Davisson-Germer (1927)

PHYS-084 (Empirical Axiom, Hilbert Space Postulate)
  The state of a quantum system is described by a vector in a
  complex Hilbert space.
  formal: |ψ⟩ ∈ H; ⟨ψ|ψ⟩ = 1
  type: empirical_axiom
  regime: QM
  depends: [MATH-050, MATH-044]
  source: Dirac (1930), von Neumann (1932)

PHYS-085 (Empirical Axiom, Observables as Operators)
  Observable quantities correspond to Hermitian operators.
  formal: A = A†; eigenvalues real; measurement yields eigenvalue
  type: empirical_axiom
  regime: QM
  depends: [PHYS-084]
  source: Dirac, von Neumann

PHYS-086 (Empirical Axiom, Schrödinger Equation)
  The time evolution of a quantum state.
  formal: iℏ ∂|ψ⟩/∂t = Ĥ|ψ⟩
  type: empirical_axiom
  regime: QM
  depends: [PHYS-084, PHYS-085]
  source: Schrödinger (1926)

PHYS-087 (Empirical Axiom, Born Rule)
  The probability of measuring eigenvalue aₙ is |⟨aₙ|ψ⟩|².
  formal: P(aₙ) = |⟨aₙ|ψ⟩|²
  type: empirical_axiom
  regime: QM
  depends: [PHYS-084]
  source: Born (1926)

PHYS-088 (Derived, Uncertainty Principle)
  Certain pairs of observables cannot be simultaneously known with
  arbitrary precision.
  formal: ΔxΔp ≥ ℏ/2
  type: derived_claim
  regime: QM
  depends: [PHYS-085, PHYS-084]
  source: Heisenberg (1927); from operator commutators

PHYS-089 (Derived, Superposition)
  A linear combination of valid states is also a valid state.
  formal: |ψ⟩ = Σᵢ cᵢ|aᵢ⟩, Σ|cᵢ|² = 1
  type: derived_claim
  regime: QM
  depends: [PHYS-084]
  source: Linear structure of Hilbert space

PHYS-090 (Empirical Axiom, Measurement Postulate)
  Upon measurement, a quantum system transitions to an eigenstate
  of the measured observable.
  formal: measurement of A on |ψ⟩ → |aₙ⟩ with probability |⟨aₙ|ψ⟩|²
  type: empirical_axiom
  regime: QM
  depends: [PHYS-087, PHYS-085]
  approximation_tags: [Copenhagen_measurement]
  source: Von Neumann (1932)
```

### Derived Quantum Phenomena

```
PHYS-091 (Derived, Quantum Tunneling)
  Quantum particles can penetrate potential barriers classically forbidden.
  formal: P_tunnel ∝ exp(−2κL)
  type: derived_claim
  regime: QM
  depends: [PHYS-086]
  source: Gamow (1928); Schrödinger equation solutions

PHYS-092 (Derived, Entanglement)
  Two or more quantum particles can have joint states that cannot be
  factored into individual states.
  formal: |ψ⟩₁₂ ≠ |φ⟩₁ ⊗ |χ⟩₂ for entangled states
  type: derived_claim
  regime: QM
  depends: [PHYS-084]
  source: Einstein-Podolsky-Rosen (1935); tensor product structure

PHYS-093 (Meta, Bell's Theorem)
  No local hidden variable theory can reproduce all predictions
  of quantum mechanics.
  formal: Bell inequality |S| ≤ 2 is violated by QM: S_QM = 2√2
  type: meta
  regime: QM
  depends: [PHYS-092, PHYS-087]
  source: Bell (1964); Aspect experiment (1982)

PHYS-094 (Empirical Axiom, Spin)
  Spin is an intrinsic angular momentum with quantized values.
  formal: S² = s(s+1)ℏ²; S_z = m_sℏ
  type: empirical_axiom
  regime: QM/QFT
  depends: [PHYS-084]
  source: Stern-Gerlach (1922), Dirac (1928)

PHYS-095 (Derived, Spin-Statistics Connection)
  Half-integer spin ↔ fermions; integer spin ↔ bosons.
  type: derived_claim
  regime: QFT
  depends: [PHYS-094, PHYS-036]
  source: QFT theorem (Pauli 1940; requires locality + Lorentz invariance)

PHYS-096 (Derived, Discrete Atomic Spectra)
  Atomic energy levels are discrete, producing characteristic spectral lines.
  formal: Eₙ = −13.6 eV/n² (hydrogen)
  type: derived_claim
  regime: QM
  depends: [PHYS-086]
  source: Bohr (1913); Schrödinger (1926)

PHYS-097 (Empirical Axiom, Particle Indistinguishability)
  Identical quantum particles are fundamentally indistinguishable.
  formal: P(1,2) = P(2,1); exchange symmetry is exact
  type: empirical_axiom
  regime: QM
  depends: [PHYS-084]
  source: Quantum statistics; Gibbs paradox resolution

PHYS-098 (Derived, Vacuum Fluctuations)
  Quantum fields have zero-point fluctuations even in vacuum.
  formal: ⟨0|Ĥ|0⟩ = ½ℏω per mode
  type: derived_claim
  regime: QFT
  depends: [PHYS-084, PHYS-086]
  source: Casimir effect (1948); QFT

PHYS-099 (Empirical Axiom, Unitarity)
  Quantum time evolution preserves inner products.
  formal: U(t) = exp(−iĤt/ℏ); U†U = I
  type: empirical_axiom
  regime: QM
  depends: [PHYS-086]
  source: Von Neumann; unitarity of quantum mechanics

PHYS-100 (Derived, No-Cloning Theorem)
  An arbitrary unknown quantum state cannot be perfectly copied.
  formal: ∄ U: |ψ⟩|0⟩ → |ψ⟩|ψ⟩ for all |ψ⟩
  type: derived_claim
  regime: QM
  depends: [PHYS-099, PHYS-084]
  source: Wootters-Zurek (1982), Dieks (1982)
```

---

# VII. Fundamental Forces & Particles (PHYS-101 → PHYS-111)

```
PHYS-101 (Derived, Four Fundamental Interactions)
  There are four fundamental interactions in current effective theory:
  gravity, electromagnetic, strong, and weak.
  type: derived_claim
  regime: modern_physics
  depends: [PHYS-071, PHYS-047, PHYS-102, PHYS-103]
  approximation_tags: [effective_field_theory]
  source: Standard Model + GR

PHYS-102 (Empirical Axiom, Strong Interaction — QCD)
  The strong interaction is described by SU(3) gauge theory with confinement.
  type: empirical_axiom
  regime: QFT
  depends: []
  source: Gell-Mann (1964), Gross-Wilczek-Politzer (1973)

PHYS-103 (Empirical Axiom, Weak Interaction)
  The weak interaction is described by SU(2)_L with W±, Z⁰ bosons.
  type: empirical_axiom
  regime: QFT
  depends: []
  source: Fermi (1934); Glashow-Weinberg-Salam (1967-68)

PHYS-104 (Derived, Electroweak Unification)
  EM and weak forces are unified at high energies.
  formal: SU(2)_L × U(1)_Y → U(1)_EM via spontaneous symmetry breaking
  type: derived_claim
  regime: QFT
  depends: [PHYS-103, PHYS-058]
  source: Glashow (1961), Weinberg (1967), Salam (1968)

PHYS-105 (Empirical Axiom, Higgs Mechanism)
  The Higgs mechanism gives mass to W±, Z bosons and fermions.
  formal: ⟨φ⟩ ≠ 0; m = yv/√2
  type: empirical_axiom
  regime: QFT
  depends: [PHYS-104]
  source: Higgs, Englert, Brout (1964); confirmed CERN (2012)

PHYS-106 (Measurement Fact, Matter Content)
  Matter is composed of quarks and leptons in three generations.
  formal: Quarks: (u,d), (c,s), (t,b); Leptons: (e,νₑ), (μ,ν_μ), (τ,ν_τ)
  type: measurement_fact
  regime: particle_physics
  depends: []
  source: Collider experiments

PHYS-107 (Derived, Antiparticles)
  For every particle there exists an antiparticle with opposite
  quantum numbers.
  formal: ∀ particle P, ∃ P̄: m_P̄ = m_P, Q_P̄ = −Q_P
  type: derived_claim
  regime: QFT
  depends: [PHYS-084, PHYS-060]
  source: Dirac (1928); positron discovered Anderson (1932)

PHYS-108 (Derived, CPT Symmetry)
  CPT symmetry holds for local Lorentz-invariant QFT.
  type: derived_claim
  regime: QFT
  depends: [PHYS-059, PHYS-084]
  approximation_tags: [locality, Lorentz_invariance, unitarity]
  source: Lüders-Pauli theorem (1954)

PHYS-109 (Measurement Fact, Baryon/Lepton Conservation)
  Baryon and lepton number are approximately conserved at accessible energies.
  type: measurement_fact
  regime: particle_physics
  depends: []
  approximation_tags: [SM_perturbative, sphalerons_violate_BplusL]
  source: Experiments + SM structure

PHYS-110 (Derived, Running Couplings)
  Gauge coupling constants run with energy scale; approximate
  unification is possible in extensions.
  type: derived_claim
  regime: QFT
  depends: [PHYS-102, PHYS-104]
  approximation_tags: [renormalization_group, model_dependent]
  source: Georgi-Glashow (1974); RG

PHYS-111 (Measurement Fact + Derived, Matter-Antimatter Asymmetry)
  The observable universe contains far more matter than antimatter.
  Baryogenesis requires the Sakharov conditions.
  formal: η = (n_B − n_B̄)/n_γ ≈ 6 × 10⁻¹⁰
  type: measurement_fact (asymmetry) + derived_claim (conditions)
  regime: cosmology
  depends: [PHYS-079, PHYS-107]
  source: Sakharov (1967); CMB measurements
```

---

## Summary

| Section | Range | Count | Types |
|---------|-------|-------|-------|
| I. Classical Mechanics | PHYS-001→020 | 20 | 2 def, 5 empirical, 11 derived, 2 empirical |
| II. Thermodynamics | PHYS-021→040 | 20 | 3 def, 6 empirical, 11 derived |
| III. Electromagnetism | PHYS-041→058 | 18 | 1 def, 10 empirical, 7 derived |
| IV. Special Relativity | PHYS-059→068 | 10 | 1 def, 2 empirical, 7 derived |
| V. General Relativity | PHYS-069→080 | 12 | 1 def, 3 empirical, 6 derived, 1 measurement, 1 def |
| VI. Quantum Mechanics | PHYS-081→100 | 20 | 11 empirical, 8 derived, 1 meta |
| VII. Fundamental Forces | PHYS-101→111 | 11 | 4 empirical, 4 derived, 3 measurement |

### By type (machine-verified)
| Type | Count |
|------|-------|
| empirical_axiom | 40 |
| derived_claim | 56 |
| definition | 10 |
| measurement_fact | 3 |
| measurement_fact + derived_claim (hybrid) | 1 |
| meta | 1 |
| **Total** | **111** (PHYS-001→111, no gaps) |

### Key corrections from original draft
1. PHYS-001/002: definitions (regime declarations), not axioms
2. Energy/momentum/angular momentum conservation: derived via Noether, not independent postulates
3. Entropy formulas (Boltzmann, Gibbs): definitions, not laws
4. PHYS-050/060: same postulate in two contexts (noted, both retained)
5. PHYS-079: ΛCDM inference, not axiom
6. PHYS-109: baryon/lepton conservation is approximate, not absolute
7. CPT: theorem under assumptions, not raw axiom

**The chain is born knowing 40 empirical postulates about the physical world, 56 consequences
derived from them, 10 definitions that frame the language, 4 measured facts about our universe,
and 1 meta-truth about the limits of local hidden variables.**

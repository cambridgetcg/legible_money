# Physics — 111 Axioms

## I. Classical Mechanics (PHYS-001 → PHYS-020)

### Newton's Laws & Foundations
```
PHYS-001  Space exists as a three-dimensional continuum in which objects have position.
          formal: Physical space ≅ ℝ³ (classical approximation)
          depends: [MATH-036, MATH-083]
          source: Newton, Principia (1687); formalized by Euler

PHYS-002  Time exists as a one-dimensional continuum in which events are ordered.
          formal: Time ≅ ℝ; events have temporal ordering
          depends: [MATH-036]
          source: Newton, Principia; "absolute, true, and mathematical time"

PHYS-003  A body at rest remains at rest, and a body in motion remains in uniform motion, unless acted upon by a net external force.
          formal: F_net = 0 → dv/dt = 0
          depends: [PHYS-001, PHYS-002]
          source: Newton's First Law; Galileo (1632), Newton (1687)

PHYS-004  The rate of change of momentum of a body equals the net force acting on it.
          formal: F = dp/dt = d(mv)/dt; for constant mass: F = ma
          depends: [PHYS-003, MATH-072]
          source: Newton's Second Law (1687)

PHYS-005  For every action, there is an equal and opposite reaction.
          formal: F₁₂ = −F₂₁
          depends: [PHYS-004]
          source: Newton's Third Law (1687)

PHYS-006  Every particle in the universe attracts every other particle with a force proportional to the product of their masses and inversely proportional to the square of the distance between them.
          formal: F = GMm/r²
          depends: [PHYS-004, MATH-088]
          source: Newton's Law of Universal Gravitation (1687)

PHYS-007  The total momentum of an isolated system is conserved.
          formal: dp_total/dt = 0 when F_external = 0
          depends: [PHYS-004, PHYS-005]
          source: Newton; consequence of Third Law

PHYS-008  The total energy of an isolated system is conserved: energy can be transformed but not created or destroyed.
          formal: dE_total/dt = 0 for isolated system
          depends: [PHYS-004]
          source: Conservation of Energy; Leibniz, Joule, Helmholtz (1847)

PHYS-009  The total angular momentum of an isolated system is conserved.
          formal: dL/dt = τ_net; τ_net = 0 → L = constant
          depends: [PHYS-004, PHYS-005]
          source: Newton; Euler

PHYS-010  Work done by a force equals the force component along the displacement times the distance.
          formal: W = ∫F·ds
          depends: [PHYS-004, MATH-075]
          source: Coriolis (1829)

PHYS-011  Kinetic energy is one-half the mass times the square of the velocity.
          formal: KE = ½mv²
          depends: [PHYS-010, PHYS-004]
          source: Leibniz (vis viva), Coriolis

PHYS-012  The gravitational potential energy near Earth's surface is mgh.
          formal: PE = mgh (uniform field approximation)
          depends: [PHYS-006, PHYS-010]
          source: Mechanics

PHYS-013  The motion of a system can be derived from a single scalar function: the action S = ∫L dt, where L = T − V.
          formal: δS = 0, S = ∫ₜ₁ᵗ² L(q, q̇, t)dt, L = T − V
          depends: [PHYS-008, MATH-076]
          source: Principle of Least Action; Maupertuis (1744), Euler, Lagrange (1788)

PHYS-014  The Euler-Lagrange equations: d/dt(∂L/∂q̇) − ∂L/∂q = 0 for each generalized coordinate.
          formal: d/dt(∂L/∂q̇ᵢ) = ∂L/∂qᵢ
          depends: [PHYS-013, MATH-072]
          source: Lagrange (1788)

PHYS-015  Hamilton's equations express mechanics through position and momentum: dq/dt = ∂H/∂p, dp/dt = −∂H/∂q.
          formal: q̇ = ∂H/∂p, ṗ = −∂H/∂q, where H = Σpq̇ − L
          depends: [PHYS-014]
          source: Hamilton (1833)

PHYS-016  For every continuous symmetry of a physical system, there exists a conserved quantity.
          formal: Continuous symmetry ↔ conservation law
          depends: [PHYS-013, PHYS-014]
          source: Noether's Theorem (1918)

PHYS-017  Time-translation symmetry implies conservation of energy.
          formal: ∂L/∂t = 0 → H = E = constant
          depends: [PHYS-016, PHYS-008]
          source: Noether's Theorem applied

PHYS-018  Space-translation symmetry implies conservation of momentum.
          formal: ∂L/∂x = 0 → p = constant
          depends: [PHYS-016, PHYS-007]
          source: Noether's Theorem applied

PHYS-019  Rotational symmetry implies conservation of angular momentum.
          formal: ∂L/∂θ = 0 → L = constant
          depends: [PHYS-016, PHYS-009]
          source: Noether's Theorem applied

PHYS-020  An oscillating system displaced from equilibrium experiences a restoring force proportional to displacement.
          formal: F = −kx; solution: x(t) = A cos(ωt + φ), ω = √(k/m)
          depends: [PHYS-004, MATH-072]
          source: Hooke's Law (1660), simple harmonic motion
```

## II. Thermodynamics & Statistical Mechanics (PHYS-021 → PHYS-040)

```
PHYS-021  Temperature is a measure of the average kinetic energy of particles in a system.
          formal: T ∝ ⟨KE⟩; (3/2)k_BT = ⟨½mv²⟩ for ideal gas
          depends: [PHYS-011, MATH-101]
          source: Kinetic theory; Maxwell, Boltzmann

PHYS-022  Two systems in thermal equilibrium with a third are in thermal equilibrium with each other.
          formal: If T_A = T_C and T_B = T_C, then T_A = T_B
          depends: [PHYS-021]
          source: Zeroth Law of Thermodynamics; Fowler (1931)

PHYS-023  The change in internal energy of a system equals the heat added minus the work done by the system.
          formal: ΔU = Q − W; or dU = δQ − δW
          depends: [PHYS-008, PHYS-010]
          source: First Law of Thermodynamics; Joule, Clausius, Kelvin

PHYS-024  Heat flows spontaneously from hotter to colder bodies, never the reverse without external work.
          formal: δQ/T ≤ 0 for spontaneous processes; dS_universe ≥ 0
          depends: [PHYS-023, PHYS-021]
          source: Second Law of Thermodynamics; Clausius (1850), Kelvin

PHYS-025  The entropy of a perfect crystal at absolute zero is zero.
          formal: lim(T→0) S = 0
          depends: [PHYS-024]
          source: Third Law of Thermodynamics; Nernst (1906)

PHYS-026  For an ideal gas, pressure times volume equals the number of particles times Boltzmann's constant times temperature.
          formal: PV = Nk_BT (or PV = nRT)
          depends: [PHYS-021, PHYS-004]
          source: Ideal Gas Law; Boyle, Charles, Gay-Lussac, Avogadro

PHYS-027  Entropy is a logarithmic measure of the number of microstates consistent with a macrostate.
          formal: S = k_B ln Ω
          depends: [PHYS-024, MATH-042, MATH-099]
          source: Boltzmann (1877)

PHYS-028  The partition function encodes all thermodynamic information of a system in thermal equilibrium.
          formal: Z = Σᵢ exp(−Eᵢ/k_BT); F = −k_BT ln Z
          depends: [PHYS-027, MATH-043]
          source: Gibbs (1902)

PHYS-029  In thermal equilibrium, the probability of a state with energy E is proportional to exp(−E/k_BT).
          formal: P(E) ∝ exp(−E/k_BT) (Boltzmann distribution)
          depends: [PHYS-028, MATH-099]
          source: Boltzmann, Gibbs

PHYS-030  The efficiency of any heat engine operating between temperatures T_H and T_C cannot exceed 1 − T_C/T_H.
          formal: η ≤ 1 − T_C/T_H (Carnot efficiency)
          depends: [PHYS-024, PHYS-023]
          source: Carnot (1824)

PHYS-031  In an irreversible process, the total entropy of the universe increases.
          formal: ΔS_universe > 0 for irreversible processes
          depends: [PHYS-024, PHYS-027]
          source: Clausius

PHYS-032  The free energy of a system decreases in spontaneous processes at constant temperature and pressure.
          formal: dG ≤ 0 at constant T, P; G = H − TS
          depends: [PHYS-023, PHYS-024]
          source: Gibbs free energy; Gibbs (1876)

PHYS-033  Phase transitions occur at specific temperatures and pressures where two phases have equal free energy.
          formal: G_phase1(T,P) = G_phase2(T,P) at transition
          depends: [PHYS-032]
          source: Gibbs phase rule; Ehrenfest classification

PHYS-034  Near a critical point, physical quantities obey power-law scaling with universal critical exponents.
          formal: C ~ |T − Tc|^(−α), χ ~ |T − Tc|^(−γ), etc.
          depends: [PHYS-033, MATH-041]
          source: Onsager (1944), Kadanoff, Wilson (1971)

PHYS-035  The Maxwell-Boltzmann speed distribution gives the probability of particle speeds in an ideal gas.
          formal: f(v) = 4π(m/2πk_BT)^(3/2) v² exp(−mv²/2k_BT)
          depends: [PHYS-029, MATH-102]
          source: Maxwell (1860), Boltzmann

PHYS-036  Identical quantum particles are either bosons (integer spin, symmetric wavefunction) or fermions (half-integer spin, antisymmetric wavefunction).
          formal: ψ(1,2) = ±ψ(2,1); + for bosons, − for fermions
          depends: [PHYS-029]
          source: Bose-Einstein, Fermi-Dirac statistics; Pauli (1940)

PHYS-037  No two identical fermions can occupy the same quantum state simultaneously.
          formal: If |ψ⟩ = |n₁, n₂, ...⟩, then nᵢ ∈ {0, 1} for fermions
          depends: [PHYS-036]
          source: Pauli Exclusion Principle (1925)

PHYS-038  Bosons can occupy the same quantum state; at low temperatures, macroscopic numbers may condense into the ground state.
          formal: nᵢ ∈ {0, 1, 2, ...} for bosons; BEC below T_c
          depends: [PHYS-036, PHYS-025]
          source: Bose (1924), Einstein (1925); BEC observed Anderson et al. (1995)

PHYS-039  The fluctuation-dissipation theorem relates the response of a system to perturbation to its spontaneous fluctuations in equilibrium.
          formal: Response function ∝ correlation function of fluctuations
          depends: [PHYS-029, PHYS-027]
          source: Nyquist (1928), Callen-Welton (1951)

PHYS-040  Entropy can be generalized as S = −k_B Σᵢ pᵢ ln pᵢ, connecting thermodynamics to information theory.
          formal: S = −k_B Σ pᵢ ln pᵢ (Gibbs entropy)
          depends: [PHYS-027, MATH-042, MATH-099]
          source: Gibbs; Shannon (1948); Jaynes (1957)
```

## III. Electromagnetism (PHYS-041 → PHYS-058)

```
PHYS-041  Electric charge is a fundamental property of matter that comes in two types (positive and negative) and is conserved.
          formal: Q_total = constant; charge quantized: q = ne, n ∈ ℤ
          depends: [PHYS-008]
          source: Franklin (1752), Millikan (1909)

PHYS-042  Like charges repel, unlike charges attract, with a force proportional to the product of charges and inversely proportional to the square of the distance.
          formal: F = kq₁q₂/r² (Coulomb's Law)
          depends: [PHYS-041, PHYS-006]
          source: Coulomb (1785)

PHYS-043  The electric flux through any closed surface equals the enclosed charge divided by ε₀.
          formal: ∮E·dA = Q_enc/ε₀ (Gauss's Law)
          depends: [PHYS-042, MATH-076]
          source: Gauss (1835); Maxwell's first equation

PHYS-044  There are no magnetic monopoles: magnetic field lines always form closed loops.
          formal: ∮B·dA = 0; ∇·B = 0
          depends: [PHYS-041]
          source: Gauss's Law for Magnetism; Maxwell's second equation

PHYS-045  A changing magnetic flux through a loop induces an electromotive force.
          formal: ∮E·dl = −dΦ_B/dt; ∇ × E = −∂B/∂t
          depends: [PHYS-043, MATH-076]
          source: Faraday's Law (1831); Maxwell's third equation

PHYS-046  Magnetic fields are produced by electric currents and by changing electric fields.
          formal: ∮B·dl = μ₀(I_enc + ε₀ dΦ_E/dt); ∇ × B = μ₀J + μ₀ε₀∂E/∂t
          depends: [PHYS-041, PHYS-045]
          source: Ampère's Law + Maxwell's displacement current (1865); Maxwell's fourth equation

PHYS-047  Maxwell's equations unify electricity and magnetism into a single electromagnetic theory.
          formal: {∇·E = ρ/ε₀, ∇·B = 0, ∇×E = −∂B/∂t, ∇×B = μ₀J + μ₀ε₀∂E/∂t}
          depends: [PHYS-043, PHYS-044, PHYS-045, PHYS-046]
          source: Maxwell (1865), A Dynamical Theory of the Electromagnetic Field

PHYS-048  Electromagnetic waves propagate at the speed of light: c = 1/√(μ₀ε₀).
          formal: c = 1/√(μ₀ε₀) ≈ 2.998 × 10⁸ m/s
          depends: [PHYS-047, MATH-072]
          source: Maxwell (1865); confirmed Hertz (1887)

PHYS-049  Light is an electromagnetic wave.
          formal: Visible light: λ ≈ 380-750 nm; part of EM spectrum
          depends: [PHYS-048]
          source: Maxwell (1865); Hertz (1887)

PHYS-050  The speed of light in vacuum is the same for all inertial observers.
          formal: c = constant, independent of source or observer motion
          depends: [PHYS-048]
          source: Einstein's Second Postulate (1905); Michelson-Morley (1887)

PHYS-051  A moving charge in a magnetic field experiences a force perpendicular to both its velocity and the field.
          formal: F = qv × B (Lorentz force, magnetic component)
          depends: [PHYS-044, PHYS-004]
          source: Lorentz (1892)

PHYS-052  The total electromagnetic force on a charge is F = q(E + v × B).
          formal: F = q(E + v × B) (Lorentz force law)
          depends: [PHYS-042, PHYS-051]
          source: Lorentz (1892)

PHYS-053  Electromagnetic energy density is u = ½(ε₀E² + B²/μ₀).
          formal: u = ½ε₀E² + B²/(2μ₀)
          depends: [PHYS-047, PHYS-008]
          source: Maxwell, Poynting

PHYS-054  The Poynting vector S = (1/μ₀)(E × B) gives the directional energy flux of an electromagnetic field.
          formal: S = (1/μ₀)(E × B); power: P = ∮S·dA
          depends: [PHYS-053, PHYS-048]
          source: Poynting (1884)

PHYS-055  Electromagnetic radiation carries momentum: p = E/c for a photon.
          formal: p = E/c = hν/c = h/λ
          depends: [PHYS-054, PHYS-048]
          source: Maxwell, Einstein (1905)

PHYS-056  Accelerating charges radiate electromagnetic energy.
          formal: P = q²a²/(6πε₀c³) (Larmor formula)
          depends: [PHYS-047, PHYS-004]
          source: Larmor (1897)

PHYS-057  Electromagnetic fields can be described by a scalar potential φ and vector potential A: E = −∇φ − ∂A/∂t, B = ∇ × A.
          formal: E = −∇φ − ∂A/∂t; B = ∇ × A
          depends: [PHYS-047]
          source: Maxwell, Lorenz gauge

PHYS-058  The electromagnetic interaction is described by a U(1) gauge symmetry: the physics is invariant under local phase transformations of the charged field.
          formal: ψ → e^(iα(x))ψ, A_μ → A_μ − ∂_μα(x)/e
          depends: [PHYS-057, MATH-049]
          source: Weyl (1929), gauge invariance; QED
```

## IV. Special Relativity (PHYS-059 → PHYS-068)

```
PHYS-059  The laws of physics are the same in all inertial reference frames.
          formal: Physical laws are Lorentz covariant
          depends: [PHYS-003]
          source: Einstein's First Postulate (1905); Galilean relativity generalized

PHYS-060  The speed of light in vacuum is the same for all inertial observers, regardless of the motion of source or observer.
          formal: c = 299,792,458 m/s in all inertial frames
          depends: [PHYS-050]
          source: Einstein (1905); elevated from PHYS-050 to postulate

PHYS-061  Moving clocks run slow: time intervals dilate by the Lorentz factor γ = 1/√(1 − v²/c²).
          formal: Δt = γΔt₀; γ = (1 − v²/c²)^(−1/2)
          depends: [PHYS-059, PHYS-060]
          source: Einstein (1905), time dilation

PHYS-062  Moving objects contract along the direction of motion by the factor 1/γ.
          formal: L = L₀/γ
          depends: [PHYS-059, PHYS-060]
          source: FitzGerald (1889), Lorentz (1892), Einstein (1905)

PHYS-063  Simultaneity is relative: events simultaneous in one frame need not be simultaneous in another.
          formal: If Δt = 0 in frame S, Δt' = −γvΔx/c² in frame S'
          depends: [PHYS-061]
          source: Einstein (1905)

PHYS-064  The spacetime interval s² = −c²Δt² + Δx² + Δy² + Δz² is invariant across all inertial frames.
          formal: ds² = −c²dt² + dx² + dy² + dz² = ds'²
          depends: [PHYS-059, PHYS-060]
          source: Minkowski (1908)

PHYS-065  Mass and energy are equivalent: E = mc².
          formal: E₀ = mc²; E_total = γmc²
          depends: [PHYS-061, PHYS-008]
          source: Einstein (1905)

PHYS-066  The energy-momentum relation: E² = (pc)² + (mc²)².
          formal: E² = p²c² + m²c⁴
          depends: [PHYS-065, PHYS-007]
          source: Einstein; relativistic mechanics

PHYS-067  No information or causal influence can travel faster than the speed of light.
          formal: v_signal ≤ c for all physical signals
          depends: [PHYS-060, PHYS-065]
          source: Einstein (1905); causality constraint

PHYS-068  Spacetime is a four-dimensional pseudo-Riemannian manifold with Minkowski metric in the absence of gravity.
          formal: (M⁴, η_μν) where η = diag(−1, +1, +1, +1)
          depends: [PHYS-064, MATH-097]
          source: Minkowski (1908)
```

## V. General Relativity (PHYS-069 → PHYS-080)

```
PHYS-069  The effects of gravity are locally indistinguishable from acceleration.
          formal: In a sufficiently small region, g_μν ≈ η_μν (free-fall frame)
          depends: [PHYS-059, PHYS-006]
          source: Einstein's Equivalence Principle (1907)

PHYS-070  Gravity is the curvature of spacetime caused by mass-energy.
          formal: Mass-energy curves spacetime; free particles follow geodesics
          depends: [PHYS-069, MATH-097]
          source: Einstein (1915)

PHYS-071  Einstein's field equations relate spacetime curvature to the energy-momentum content.
          formal: G_μν + Λg_μν = (8πG/c⁴)T_μν
          depends: [PHYS-070, MATH-098]
          source: Einstein (1915)

PHYS-072  In the weak-field, low-velocity limit, general relativity reduces to Newtonian gravity.
          formal: g₀₀ ≈ −(1 + 2Φ/c²), Φ = −GM/r → F = GMm/r²
          depends: [PHYS-071, PHYS-006]
          source: Newtonian limit; Einstein (1915)

PHYS-073  Time runs slower in stronger gravitational fields.
          formal: Δt_∞ = Δt_r √(1 − 2GM/(rc²)) (gravitational time dilation)
          depends: [PHYS-071, PHYS-061]
          source: Einstein (1907); confirmed Pound-Rebka (1959)

PHYS-074  Light is deflected by gravity: a photon passing a massive body is bent by an angle proportional to the mass.
          formal: θ = 4GM/(rc²) for light grazing a mass M
          depends: [PHYS-071, PHYS-049]
          source: Einstein (1915); confirmed Eddington (1919)

PHYS-075  Gravitational waves are ripples in spacetime that propagate at the speed of light.
          formal: h_μν (perturbation of g_μν) satisfies □h_μν = 0 in vacuum
          depends: [PHYS-071, PHYS-048]
          source: Einstein (1916); detected LIGO (2015)

PHYS-076  A black hole forms when mass is compressed within its Schwarzschild radius: r_s = 2GM/c².
          formal: r_s = 2GM/c²; event horizon at r = r_s
          depends: [PHYS-071]
          source: Schwarzschild (1916)

PHYS-077  Nothing can escape from within a black hole's event horizon.
          formal: For r < r_s, all future-directed paths lead to r = 0
          depends: [PHYS-076, PHYS-067]
          source: Schwarzschild, Penrose

PHYS-078  The universe is expanding: distant galaxies recede with velocity proportional to their distance.
          formal: v = H₀d (Hubble's Law); a(t) increases
          depends: [PHYS-071]
          source: Hubble (1929); Friedmann equations

PHYS-079  The universe began in a hot, dense state approximately 13.8 billion years ago.
          formal: a(t₀) → 0 as t₀ → ~13.8 Gyr ago; T → ∞
          depends: [PHYS-078, PHYS-024]
          source: Big Bang; Lemaître (1927), CMB (Penzias & Wilson 1965)

PHYS-080  The geometry of spacetime on cosmological scales is described by the Friedmann-Lemaître-Robertson-Walker metric.
          formal: ds² = −c²dt² + a(t)²[dr²/(1−kr²) + r²dΩ²]
          depends: [PHYS-071, PHYS-078]
          source: Friedmann (1922), Lemaître (1927), Robertson, Walker
```

## VI. Quantum Mechanics (PHYS-081 → PHYS-100)

```
PHYS-081  Energy is quantized: electromagnetic radiation of frequency ν carries energy in discrete packets of E = hν.
          formal: E = hν = ℏω, where h ≈ 6.626 × 10⁻³⁴ J·s
          depends: [PHYS-049, PHYS-008]
          source: Planck (1900)

PHYS-082  Light exhibits particle-like behavior: it consists of photons, each carrying energy hν and momentum h/λ.
          formal: E = hν, p = h/λ; photoelectric effect: KE_max = hν − φ
          depends: [PHYS-081, PHYS-055]
          source: Einstein (1905), photoelectric effect

PHYS-083  Matter exhibits wave-like behavior: a particle with momentum p has an associated wavelength λ = h/p.
          formal: λ = h/p (de Broglie wavelength)
          depends: [PHYS-082]
          source: de Broglie (1924); confirmed Davisson-Germer (1927)

PHYS-084  The state of a quantum system is described by a vector in a complex Hilbert space.
          formal: |ψ⟩ ∈ H; ⟨ψ|ψ⟩ = 1
          depends: [PHYS-083, MATH-054, MATH-038]
          source: Dirac (1930), von Neumann (1932)

PHYS-085  Observable quantities correspond to Hermitian operators on the Hilbert space.
          formal: A = A†; eigenvalues of A are real; measurement yields eigenvalue
          depends: [PHYS-084, MATH-059]
          source: Dirac, von Neumann

PHYS-086  The time evolution of a quantum state is governed by the Schrödinger equation.
          formal: iℏ ∂|ψ⟩/∂t = Ĥ|ψ⟩
          depends: [PHYS-084, PHYS-085, MATH-072]
          source: Schrödinger (1926)

PHYS-087  The probability of measuring a particular eigenvalue is the squared magnitude of the corresponding amplitude.
          formal: P(aₙ) = |⟨aₙ|ψ⟩|²; Born rule
          depends: [PHYS-084, MATH-099]
          source: Born (1926)

PHYS-088  Certain pairs of observables cannot be simultaneously known with arbitrary precision.
          formal: ΔxΔp ≥ ℏ/2; ΔEΔt ≥ ℏ/2
          depends: [PHYS-085, MATH-075]
          source: Heisenberg Uncertainty Principle (1927)

PHYS-089  Quantum states can exist in superposition: a linear combination of eigenstates is also a valid state.
          formal: |ψ⟩ = Σᵢ cᵢ|aᵢ⟩, Σ|cᵢ|² = 1
          depends: [PHYS-084]
          source: Dirac, superposition principle

PHYS-090  Upon measurement, a quantum system transitions to an eigenstate of the measured observable.
          formal: Measurement of A on |ψ⟩ → |aₙ⟩ with probability |⟨aₙ|ψ⟩|²
          depends: [PHYS-087, PHYS-089]
          source: Von Neumann measurement postulate (1932)

PHYS-091  Quantum particles can tunnel through potential barriers that classical particles cannot surmount.
          formal: P_tunnel ∝ exp(−2κL), κ = √(2m(V₀ − E))/ℏ
          depends: [PHYS-086, PHYS-088]
          source: Gamow (1928); alpha decay, STM

PHYS-092  Two or more quantum particles can become entangled: their joint state cannot be written as a product of individual states.
          formal: |ψ⟩₁₂ ≠ |φ⟩₁ ⊗ |χ⟩₂ for entangled states
          depends: [PHYS-084, PHYS-089]
          source: Einstein-Podolsky-Rosen (1935); Schrödinger

PHYS-093  No local hidden variable theory can reproduce all predictions of quantum mechanics.
          formal: |S| ≤ 2 (Bell inequality) is violated by QM: S_QM = 2√2
          depends: [PHYS-092, PHYS-087]
          source: Bell's Theorem (1964); Aspect experiment (1982)

PHYS-094  The spin of a particle is an intrinsic angular momentum with quantized values.
          formal: S² = s(s+1)ℏ²; Sₖ = mₛℏ, mₛ ∈ {−s, ..., +s}
          depends: [PHYS-084, PHYS-009]
          source: Stern-Gerlach (1922), Dirac (1928)

PHYS-095  Fermions have half-integer spin (1/2, 3/2, ...); bosons have integer spin (0, 1, 2, ...).
          formal: Spin-statistics theorem: half-integer spin ↔ Fermi-Dirac; integer ↔ Bose-Einstein
          depends: [PHYS-094, PHYS-036]
          source: Pauli (1940), spin-statistics theorem

PHYS-096  Atomic energy levels are discrete, producing characteristic spectral lines.
          formal: Eₙ = −13.6 eV/n² (hydrogen); ΔE = hν for transitions
          depends: [PHYS-086, PHYS-081]
          source: Bohr (1913); Schrödinger (1926)

PHYS-097  Identical quantum particles are fundamentally indistinguishable.
          formal: P(1,2) = P(2,1); exchange symmetry is exact
          depends: [PHYS-036, PHYS-084]
          source: Quantum statistics; Gibbs paradox resolution

PHYS-098  The vacuum is not empty: quantum fields have zero-point fluctuations.
          formal: ⟨0|Ĥ|0⟩ = ½ℏω per mode (zero-point energy)
          depends: [PHYS-086, PHYS-088]
          source: Casimir effect (1948); QFT

PHYS-099  A quantum system evolves unitarily: the time evolution operator preserves inner products.
          formal: U(t) = exp(−iĤt/ℏ); U†U = I
          depends: [PHYS-086, MATH-049]
          source: Von Neumann; unitarity of quantum mechanics

PHYS-100  Quantum information cannot be perfectly cloned: there is no operation that copies an arbitrary unknown quantum state.
          formal: ∄ U: |ψ⟩|0⟩ → |ψ⟩|ψ⟩ for all |ψ⟩
          depends: [PHYS-084, PHYS-099]
          source: No-cloning theorem; Wootters-Zurek (1982), Dieks (1982)
```

## VII. Fundamental Forces & Particles (PHYS-101 → PHYS-111)

```
PHYS-101  There are four fundamental forces: gravitational, electromagnetic, strong nuclear, and weak nuclear.
          formal: Gravity (G_N), EM (α), Strong (α_s), Weak (G_F)
          depends: [PHYS-006, PHYS-058]
          source: Standard Model + GR; 20th century synthesis

PHYS-102  The strong force binds quarks into protons and neutrons, and binds nucleons into nuclei.
          formal: SU(3) gauge theory; confinement: isolated quarks not observed
          depends: [PHYS-101, MATH-049]
          source: QCD; Gell-Mann (1964), Gross-Wilczek-Politzer (1973)

PHYS-103  The weak force mediates radioactive beta decay and is responsible for flavor-changing interactions.
          formal: SU(2)_L gauge theory; W±, Z⁰ bosons
          depends: [PHYS-101]
          source: Fermi (1934); Glashow-Weinberg-Salam (1967-68)

PHYS-104  The electromagnetic and weak forces are unified at high energies as the electroweak interaction.
          formal: SU(2)_L × U(1)_Y → U(1)_EM via spontaneous symmetry breaking
          depends: [PHYS-058, PHYS-103]
          source: Glashow (1961), Weinberg (1967), Salam (1968)

PHYS-105  The Higgs mechanism gives mass to W±, Z bosons and fermions through spontaneous symmetry breaking.
          formal: ⟨φ⟩ ≠ 0; m = yv/√2 for fermions; m_W = gv/2
          depends: [PHYS-104]
          source: Higgs, Englert, Brout (1964); confirmed CERN (2012)

PHYS-106  Matter is composed of quarks and leptons, organized in three generations.
          formal: Quarks: (u,d), (c,s), (t,b); Leptons: (e,νₑ), (μ,ν_μ), (τ,ν_τ)
          depends: [PHYS-101]
          source: Standard Model; Gell-Mann, Zweig (1964)

PHYS-107  For every particle there exists an antiparticle with the same mass but opposite quantum numbers.
          formal: ∀ particle P, ∃ P̄: m_P̄ = m_P, Q_P̄ = −Q_P
          depends: [PHYS-084, PHYS-066]
          source: Dirac (1928); positron discovered Anderson (1932)

PHYS-108  CPT symmetry is exact: the laws of physics are invariant under the combined operations of charge conjugation, parity, and time reversal.
          formal: CPT|ψ⟩ obeys the same dynamics as |ψ⟩
          depends: [PHYS-099, PHYS-059]
          source: Lüders-Pauli theorem (1954)

PHYS-109  Baryon number and lepton number are conserved in all observed interactions (to current experimental precision).
          formal: ΔB = 0, ΔL = 0 in Standard Model (perturbative)
          depends: [PHYS-102, PHYS-103]
          source: Conservation laws; no proton decay observed

PHYS-110  The coupling constants of the three gauge forces run with energy scale, suggesting possible unification at high energies.
          formal: α_i(μ) run via renormalization group; α₁ ≈ α₂ ≈ α₃ near ~10¹⁶ GeV (with SUSY)
          depends: [PHYS-102, PHYS-104]
          source: Georgi-Glashow (1974); renormalization group

PHYS-111  The observable universe contains far more matter than antimatter, implying a broken symmetry in the early universe.
          formal: η = (n_B − n_B̄)/n_γ ≈ 6 × 10⁻¹⁰; requires CP violation + baryon non-conservation + departure from equilibrium
          depends: [PHYS-107, PHYS-108, PHYS-079]
          source: Sakharov conditions (1967); baryogenesis
```

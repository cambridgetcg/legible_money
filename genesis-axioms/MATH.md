# Mathematics — 111 Axioms

## I. Foundations of Existence (MATH-001 → MATH-012)

### Set Theory & Logic Foundations
```
MATH-001  Something exists.
          formal: ∃x(x = x)
          depends: []
          source: Parmenides; Zermelo (1908)

MATH-002  A thing is identical to itself.
          formal: ∀x(x = x)
          depends: [MATH-001]
          source: Aristotle, Law of Identity

MATH-003  Nothing can both be and not be in the same respect at the same time.
          formal: ¬(P ∧ ¬P)
          depends: [MATH-002]
          source: Aristotle, Law of Non-Contradiction

MATH-004  Everything either is or is not.
          formal: P ∨ ¬P
          depends: [MATH-003]
          source: Aristotle, Law of Excluded Middle

MATH-005  The empty collection exists.
          formal: ∃S(∀x(x ∉ S))
          depends: [MATH-001]
          source: Zermelo, Axiom of Empty Set

MATH-006  Two collections with the same members are the same collection.
          formal: ∀A∀B(∀x(x ∈ A ↔ x ∈ B) → A = B)
          depends: [MATH-002, MATH-005]
          source: Zermelo, Axiom of Extensionality

MATH-007  For any two things, there exists a collection containing exactly those two things.
          formal: ∀a∀b∃S(∀x(x ∈ S ↔ (x = a ∨ x = b)))
          depends: [MATH-001, MATH-006]
          source: Zermelo, Axiom of Pairing

MATH-008  For any collection of collections, there exists a collection containing all their members.
          formal: ∀F∃U(∀x(x ∈ U ↔ ∃S(S ∈ F ∧ x ∈ S)))
          depends: [MATH-007]
          source: Zermelo, Axiom of Union

MATH-009  For any collection, there exists a collection of all its sub-collections.
          formal: ∀S∃P(∀T(T ∈ P ↔ T ⊆ S))
          depends: [MATH-006]
          source: Zermelo, Axiom of Power Set

MATH-010  Every non-empty collection contains a member that shares no member with it.
          formal: ∀S(S ≠ ∅ → ∃x(x ∈ S ∧ x ∩ S = ∅))
          depends: [MATH-005, MATH-006]
          source: Zermelo, Axiom of Foundation/Regularity

MATH-011  For any collection and any definite property, there exists a sub-collection of all members satisfying that property.
          formal: ∀S∃T(∀x(x ∈ T ↔ (x ∈ S ∧ φ(x))))
          depends: [MATH-006]
          source: Zermelo, Axiom Schema of Separation

MATH-012  There exists an infinite collection: a collection that contains the empty set and, for every member, also contains its successor.
          formal: ∃I(∅ ∈ I ∧ ∀x(x ∈ I → x ∪ {x} ∈ I))
          depends: [MATH-005, MATH-008]
          source: Zermelo, Axiom of Infinity
```

## II. Natural Numbers & Arithmetic (MATH-013 → MATH-030)

### Peano Axioms & Basic Arithmetic
```
MATH-013  Zero is a natural number.
          formal: 0 ∈ ℕ
          depends: [MATH-005, MATH-012]
          source: Peano (1889)

MATH-014  Every natural number has a unique successor that is also a natural number.
          formal: ∀n ∈ ℕ, ∃!S(n) ∈ ℕ
          depends: [MATH-013]
          source: Peano

MATH-015  Zero is not the successor of any natural number.
          formal: ¬∃n ∈ ℕ(S(n) = 0)
          depends: [MATH-013, MATH-014]
          source: Peano

MATH-016  If two natural numbers have the same successor, they are equal.
          formal: ∀m,n ∈ ℕ(S(m) = S(n) → m = n)
          depends: [MATH-014]
          source: Peano

MATH-017  If a property holds for zero and is preserved by succession, it holds for all natural numbers.
          formal: (φ(0) ∧ ∀n(φ(n) → φ(S(n)))) → ∀n(φ(n))
          depends: [MATH-013, MATH-014]
          source: Peano, Axiom of Induction

MATH-018  Addition is defined: n + 0 = n, and n + S(m) = S(n + m).
          formal: ∀n(n + 0 = n) ∧ ∀n,m(n + S(m) = S(n + m))
          depends: [MATH-013, MATH-014]
          source: Peano/Dedekind

MATH-019  Multiplication is defined: n × 0 = 0, and n × S(m) = (n × m) + n.
          formal: ∀n(n × 0 = 0) ∧ ∀n,m(n × S(m) = n × m + n)
          depends: [MATH-018]
          source: Peano/Dedekind

MATH-020  Addition is commutative.
          formal: ∀a,b ∈ ℕ(a + b = b + a)
          depends: [MATH-018, MATH-017]
          source: Proved from Peano axioms

MATH-021  Addition is associative.
          formal: ∀a,b,c ∈ ℕ((a + b) + c = a + (b + c))
          depends: [MATH-018, MATH-017]
          source: Proved from Peano axioms

MATH-022  Multiplication is commutative.
          formal: ∀a,b ∈ ℕ(a × b = b × a)
          depends: [MATH-019, MATH-017]
          source: Proved from Peano axioms

MATH-023  Multiplication is associative.
          formal: ∀a,b,c ∈ ℕ((a × b) × c = a × (b × c))
          depends: [MATH-019, MATH-017]
          source: Proved from Peano axioms

MATH-024  Multiplication distributes over addition.
          formal: ∀a,b,c ∈ ℕ(a × (b + c) = a × b + a × c)
          depends: [MATH-018, MATH-019, MATH-017]
          source: Proved from Peano axioms

MATH-025  The natural numbers are well-ordered: every non-empty subset has a least element.
          formal: ∀S ⊆ ℕ(S ≠ ∅ → ∃m ∈ S(∀n ∈ S(m ≤ n)))
          depends: [MATH-013, MATH-017]
          source: Well-ordering principle

MATH-026  For any natural numbers a and b with b ≠ 0, there exist unique q and r such that a = bq + r and 0 ≤ r < b.
          formal: ∀a ∈ ℕ, ∀b ∈ ℕ⁺, ∃!q,r ∈ ℕ(a = bq + r ∧ r < b)
          depends: [MATH-018, MATH-019, MATH-025]
          source: Division algorithm

MATH-027  A natural number greater than 1 is prime if its only divisors are 1 and itself.
          formal: prime(p) ↔ (p > 1 ∧ ∀d(d | p → (d = 1 ∨ d = p)))
          depends: [MATH-026]
          source: Euclid, Elements VII

MATH-028  Every natural number greater than 1 is either prime or a product of primes.
          formal: ∀n > 1, ∃p₁,...,pₖ prime: n = p₁ × ... × pₖ
          depends: [MATH-027, MATH-025]
          source: Fundamental Theorem of Arithmetic (existence)

MATH-029  The prime factorization of any natural number greater than 1 is unique up to ordering.
          formal: If n = p₁...pₖ = q₁...qₘ (primes), then k = m and pᵢ = qσ(i) for some permutation σ
          depends: [MATH-028, MATH-026]
          source: Fundamental Theorem of Arithmetic (uniqueness); Euclid/Gauss

MATH-030  There are infinitely many primes.
          formal: ∀n ∈ ℕ, ∃p > n: prime(p)
          depends: [MATH-027, MATH-012]
          source: Euclid, Elements IX.20
```

## III. Integers, Rationals, Reals (MATH-031 → MATH-048)

### Extending the Number Systems
```
MATH-031  The integers extend the natural numbers with additive inverses: for every n, there exists −n such that n + (−n) = 0.
          formal: (ℤ, +) is an abelian group extending (ℕ, +)
          depends: [MATH-020, MATH-021]
          source: Algebraic completion

MATH-032  The integers form an integral domain: if ab = 0 then a = 0 or b = 0.
          formal: ∀a,b ∈ ℤ(ab = 0 → a = 0 ∨ b = 0)
          depends: [MATH-031, MATH-019]
          source: Ring theory

MATH-033  The rational numbers are the field of fractions of the integers.
          formal: ℚ = {a/b : a,b ∈ ℤ, b ≠ 0}, with a/b = c/d iff ad = bc
          depends: [MATH-032]
          source: Field construction

MATH-034  The rationals are dense: between any two distinct rationals lies another.
          formal: ∀p,q ∈ ℚ(p < q → ∃r ∈ ℚ(p < r < q))
          depends: [MATH-033]
          source: Density of ℚ

MATH-035  The rationals are not complete: there exist bounded sequences of rationals with no rational limit.
          formal: ∃S ⊂ ℚ(S bounded above ∧ sup(S) ∉ ℚ) (e.g., {x ∈ ℚ : x² < 2})
          depends: [MATH-034]
          source: Pythagoreans; irrationality of √2

MATH-036  The real numbers form a complete ordered field: every non-empty set bounded above has a least upper bound.
          formal: ∀S ⊆ ℝ(S ≠ ∅ ∧ S bounded above → ∃sup(S) ∈ ℝ)
          depends: [MATH-033, MATH-035]
          source: Dedekind (1872), Completeness Axiom

MATH-037  The real numbers are uncountable: there is no bijection between ℕ and ℝ.
          formal: |ℝ| > |ℕ| (ℵ₀ < 𝔠)
          depends: [MATH-036, MATH-012]
          source: Cantor (1874), diagonal argument

MATH-038  The complex numbers extend the reals with a square root of −1.
          formal: ℂ = {a + bi : a,b ∈ ℝ}, where i² = −1
          depends: [MATH-036]
          source: Cardano (1545), Euler, Gauss

MATH-039  The complex numbers are algebraically closed: every non-constant polynomial has a root.
          formal: ∀p(x) ∈ ℂ[x], deg(p) ≥ 1 → ∃z ∈ ℂ(p(z) = 0)
          depends: [MATH-038]
          source: Fundamental Theorem of Algebra; d'Alembert/Gauss

MATH-040  For any real number x, the absolute value |x| satisfies: |x| ≥ 0, |x| = 0 iff x = 0, |xy| = |x||y|, and |x + y| ≤ |x| + |y|.
          formal: Triangle inequality: |x + y| ≤ |x| + |y|
          depends: [MATH-036]
          source: Metric space foundation

MATH-041  Exponentiation: for any real a > 0 and natural n, aⁿ is defined recursively as a⁰ = 1 and aⁿ⁺¹ = aⁿ × a.
          formal: a⁰ = 1, aⁿ⁺¹ = aⁿ · a
          depends: [MATH-036, MATH-014]
          source: Recursive definition

MATH-042  Logarithms: for any real a > 0, a ≠ 1, and x > 0, there exists a unique y such that aʸ = x, written y = log_a(x).
          formal: aʸ = x ↔ y = logₐ(x)
          depends: [MATH-041, MATH-036]
          source: Napier (1614)

MATH-043  The number e is the unique real number such that the derivative of eˣ equals eˣ.
          formal: e = lim(n→∞)(1 + 1/n)ⁿ ≈ 2.71828...
          depends: [MATH-036, MATH-042]
          source: Euler (1748)

MATH-044  π is the ratio of a circle's circumference to its diameter, and π is transcendental.
          formal: π = C/d; π ∉ algebraic numbers
          depends: [MATH-036]
          source: Archimedes; Lindemann (1882)

MATH-045  Euler's identity: e^(iπ) + 1 = 0.
          formal: e^(iπ) = −1
          depends: [MATH-043, MATH-044, MATH-038]
          source: Euler (1748)

MATH-046  The Archimedean property: for any real x > 0, there exists a natural number n such that n > x.
          formal: ∀x ∈ ℝ, x > 0 → ∃n ∈ ℕ(n > x)
          depends: [MATH-036, MATH-013]
          source: Archimedes

MATH-047  Between any two distinct real numbers lies a rational number.
          formal: ∀a,b ∈ ℝ(a < b → ∃q ∈ ℚ(a < q < b))
          depends: [MATH-036, MATH-034, MATH-046]
          source: Density of ℚ in ℝ

MATH-048  The real numbers can be constructed as equivalence classes of Cauchy sequences of rationals.
          formal: ℝ ≅ {Cauchy sequences in ℚ} / ~
          depends: [MATH-033, MATH-036]
          source: Cantor (1872), Méray (1869)
```

## IV. Algebra (MATH-049 → MATH-066)

### Groups, Rings, Fields, Vector Spaces
```
MATH-049  A group is a set with an associative binary operation, an identity element, and inverses for every element.
          formal: (G, ·): ∀a,b,c((ab)c = a(bc)) ∧ ∃e(ae = ea = a) ∧ ∀a∃a⁻¹(aa⁻¹ = e)
          depends: [MATH-002, MATH-006]
          source: Galois (1832), Cayley (1854)

MATH-050  The order of any element divides the order of a finite group.
          formal: |G| < ∞ → ∀g ∈ G, ord(g) | |G|
          depends: [MATH-049, MATH-026]
          source: Lagrange's Theorem

MATH-051  Every group of prime order is cyclic.
          formal: |G| = p (prime) → G ≅ ℤ/pℤ
          depends: [MATH-050, MATH-027]
          source: Corollary of Lagrange

MATH-052  A ring is a set with two operations (addition and multiplication) where addition forms an abelian group, multiplication is associative, and multiplication distributes over addition.
          formal: (R, +, ×): (R,+) abelian group, × associative, × distributes over +
          depends: [MATH-049, MATH-024]
          source: Fraenkel (1914), Noether (1921)

MATH-053  A field is a commutative ring where every non-zero element has a multiplicative inverse.
          formal: (F, +, ×): commutative ring, ∀a ≠ 0, ∃a⁻¹(a·a⁻¹ = 1)
          depends: [MATH-052]
          source: Dedekind, Weber (1882)

MATH-054  A vector space over a field F is a set V with vector addition and scalar multiplication satisfying linearity axioms.
          formal: (V, F, +, ·) satisfying 8 axioms of linearity
          depends: [MATH-049, MATH-053]
          source: Grassmann (1844), Peano (1888)

MATH-055  Every vector space has a basis, and all bases have the same cardinality (dimension).
          formal: ∀V over F, ∃ basis B; |B₁| = |B₂| for any two bases
          depends: [MATH-054]
          source: Steinitz (1913); requires Axiom of Choice for infinite case

MATH-056  A linear map between vector spaces preserves addition and scalar multiplication.
          formal: T: V → W linear iff T(αu + βv) = αT(u) + βT(v)
          depends: [MATH-054]
          source: Linear algebra foundations

MATH-057  For a linear map T: V → W, dim(V) = dim(ker T) + dim(im T).
          formal: dim(V) = nullity(T) + rank(T)
          depends: [MATH-055, MATH-056]
          source: Rank-nullity theorem

MATH-058  Every square matrix satisfies its own characteristic polynomial.
          formal: p_A(A) = 0 where p_A(λ) = det(A − λI)
          depends: [MATH-056]
          source: Cayley-Hamilton theorem

MATH-059  A real symmetric matrix is diagonalizable with real eigenvalues.
          formal: A = Aᵀ → A = QΛQᵀ, Q orthogonal, Λ diagonal real
          depends: [MATH-058, MATH-036]
          source: Spectral theorem (finite-dimensional)

MATH-060  An ideal I in a commutative ring R is maximal if and only if R/I is a field.
          formal: I maximal ↔ R/I is a field
          depends: [MATH-052, MATH-053]
          source: Ring theory; Krull

MATH-061  Every commutative ring with unity has a maximal ideal.
          formal: R commutative with 1 → ∃ maximal ideal M ⊂ R
          depends: [MATH-060]
          source: Krull's theorem (requires Zorn's Lemma)

MATH-062  If a polynomial of degree n over a field has more than n roots, it is the zero polynomial.
          formal: deg(p) = n ∧ |{x : p(x) = 0}| > n → p = 0
          depends: [MATH-053, MATH-026]
          source: Factor theorem consequence

MATH-063  A finite field has p^n elements for some prime p and positive integer n.
          formal: |F| < ∞ → |F| = pⁿ for some prime p, n ≥ 1
          depends: [MATH-053, MATH-051]
          source: Galois field theory; Wedderburn

MATH-064  The group of permutations on n elements (Sₙ) has order n!.
          formal: |Sₙ| = n!
          depends: [MATH-049, MATH-019]
          source: Cauchy, Cayley

MATH-065  Every finite group is isomorphic to a subgroup of some permutation group.
          formal: ∀G finite, ∃n: G ↪ Sₙ
          depends: [MATH-049, MATH-064]
          source: Cayley's theorem

MATH-066  The Axiom of Choice: for any collection of non-empty sets, there exists a function choosing one element from each.
          formal: ∀F(∅ ∉ F → ∃f: F → ∪F, ∀S ∈ F(f(S) ∈ S))
          depends: [MATH-006, MATH-009]
          source: Zermelo (1904); independent of ZF (Gödel 1938, Cohen 1963)
```

## V. Analysis (MATH-067 → MATH-082)

### Calculus, Limits, Continuity, Integration
```
MATH-067  A sequence of real numbers converges to L if for every ε > 0 there exists N such that |aₙ − L| < ε for all n > N.
          formal: lim(n→∞) aₙ = L ↔ ∀ε > 0, ∃N, ∀n > N: |aₙ − L| < ε
          depends: [MATH-036, MATH-040]
          source: Weierstrass, ε-δ formalization

MATH-068  Every bounded monotone sequence of real numbers converges.
          formal: (aₙ) bounded + monotone → (aₙ) converges
          depends: [MATH-067, MATH-036]
          source: Monotone convergence theorem

MATH-069  A function f is continuous at a if lim(x→a) f(x) = f(a).
          formal: ∀ε > 0, ∃δ > 0, |x − a| < δ → |f(x) − f(a)| < ε
          depends: [MATH-067]
          source: Weierstrass

MATH-070  A continuous function on a closed bounded interval attains its maximum and minimum.
          formal: f continuous on [a,b] → ∃c,d ∈ [a,b]: f(c) ≤ f(x) ≤ f(d) ∀x ∈ [a,b]
          depends: [MATH-069, MATH-036]
          source: Extreme Value Theorem; Weierstrass

MATH-071  If a continuous function on [a,b] takes values f(a) and f(b), it takes every value between them.
          formal: f continuous, f(a) < v < f(b) → ∃c ∈ (a,b): f(c) = v
          depends: [MATH-069, MATH-036]
          source: Intermediate Value Theorem; Bolzano

MATH-072  The derivative of f at a is the limit of the difference quotient: f'(a) = lim(h→0)(f(a+h) − f(a))/h.
          formal: f'(a) = lim(h→0) [f(a+h) − f(a)] / h
          depends: [MATH-067, MATH-069]
          source: Newton (1666), Leibniz (1684)

MATH-073  If f is differentiable on (a,b) and continuous on [a,b], there exists c in (a,b) where f'(c) = (f(b)−f(a))/(b−a).
          formal: ∃c ∈ (a,b): f'(c) = (f(b) − f(a))/(b − a)
          depends: [MATH-072, MATH-070]
          source: Mean Value Theorem; Lagrange

MATH-074  The chain rule: if g is differentiable at a and f is differentiable at g(a), then (f∘g)'(a) = f'(g(a))·g'(a).
          formal: (f ∘ g)'(a) = f'(g(a)) · g'(a)
          depends: [MATH-072]
          source: Leibniz

MATH-075  The Riemann integral of a bounded function on [a,b] is defined as the limit of Riemann sums when it exists.
          formal: ∫ₐᵇ f(x)dx = lim(|P|→0) Σf(cᵢ)Δxᵢ
          depends: [MATH-067, MATH-069]
          source: Riemann (1854)

MATH-076  The Fundamental Theorem of Calculus: differentiation and integration are inverse operations.
          formal: If f continuous on [a,b], then d/dx ∫ₐˣ f(t)dt = f(x); and ∫ₐᵇ f'(x)dx = f(b) − f(a)
          depends: [MATH-072, MATH-075]
          source: Newton, Leibniz; rigorous: Cauchy

MATH-077  Taylor's theorem: a sufficiently smooth function equals its Taylor polynomial plus a remainder term.
          formal: f(x) = Σₖ₌₀ⁿ f⁽ᵏ⁾(a)/k! · (x−a)ᵏ + Rₙ(x)
          depends: [MATH-072, MATH-076]
          source: Taylor (1715), Lagrange remainder

MATH-078  A power series converges absolutely inside its radius of convergence and diverges outside it.
          formal: Σaₙxⁿ has radius R: converges for |x| < R, diverges for |x| > R
          depends: [MATH-067, MATH-077]
          source: Cauchy, Hadamard

MATH-079  The Lebesgue integral extends the Riemann integral to a larger class of functions, with superior convergence properties.
          formal: L¹ ⊃ R-integrable; dominated convergence holds
          depends: [MATH-075, MATH-009]
          source: Lebesgue (1902)

MATH-080  Every continuous function on [a,b] can be uniformly approximated by polynomials.
          formal: ∀ε > 0, ∃p polynomial: sup|f(x) − p(x)| < ε on [a,b]
          depends: [MATH-069, MATH-077]
          source: Weierstrass Approximation Theorem (1885)

MATH-081  A metric space is compact if and only if every sequence has a convergent subsequence.
          formal: X compact ↔ every sequence in X has a convergent subsequence (sequential compactness)
          depends: [MATH-067, MATH-040]
          source: Bolzano-Weierstrass; Heine-Borel

MATH-082  In ℝⁿ, a set is compact if and only if it is closed and bounded.
          formal: K ⊆ ℝⁿ compact ↔ K closed and bounded
          depends: [MATH-081, MATH-036]
          source: Heine-Borel Theorem
```

## VI. Geometry & Topology (MATH-083 → MATH-098)

### Euclidean, Non-Euclidean, Topological
```
MATH-083  A point has no part. A line has length but no breadth.
          formal: Primitive notions of geometry
          depends: [MATH-001]
          source: Euclid, Elements I, Definitions 1-2

MATH-084  Between any two points, a unique straight line can be drawn.
          formal: ∀P,Q ∃! line L: P,Q ∈ L
          depends: [MATH-083]
          source: Euclid, Postulate 1

MATH-085  A finite straight line can be extended continuously in a straight line.
          formal: Any line segment can be extended indefinitely
          depends: [MATH-084]
          source: Euclid, Postulate 2

MATH-086  A circle can be drawn with any center and any radius.
          formal: ∀P, ∀r > 0, ∃ circle C(P, r)
          depends: [MATH-083, MATH-036]
          source: Euclid, Postulate 3

MATH-087  All right angles are equal to one another.
          formal: If ∠A and ∠B are right angles, then ∠A = ∠B
          depends: [MATH-084]
          source: Euclid, Postulate 4

MATH-088  In a right triangle, the square of the hypotenuse equals the sum of the squares of the other two sides.
          formal: a² + b² = c²
          depends: [MATH-084, MATH-087, MATH-018]
          source: Pythagorean Theorem; Euclid I.47

MATH-089  The angles of a triangle in Euclidean geometry sum to π radians.
          formal: α + β + γ = π (in Euclidean plane)
          depends: [MATH-084, MATH-087, MATH-044]
          source: Euclid I.32

MATH-090  Consistent geometries exist in which the parallel postulate does not hold.
          formal: ∃ models of geometry where through a point not on a line, zero or multiple parallels exist
          depends: [MATH-084, MATH-085]
          source: Lobachevsky (1829), Bolyai (1832), Riemann (1854)

MATH-091  A topological space is a set X with a collection of open sets closed under arbitrary unions and finite intersections, containing ∅ and X.
          formal: (X, τ): ∅,X ∈ τ; arbitrary ∪ ∈ τ; finite ∩ ∈ τ
          depends: [MATH-006, MATH-008]
          source: Hausdorff (1914)

MATH-092  A function between topological spaces is continuous if the preimage of every open set is open.
          formal: f: X → Y continuous ↔ ∀U ∈ τ_Y, f⁻¹(U) ∈ τ_X
          depends: [MATH-091, MATH-069]
          source: Topological generalization of Weierstrass

MATH-093  Two spaces are homeomorphic if there exists a continuous bijection with continuous inverse.
          formal: X ≅ Y ↔ ∃f: X → Y, f bijective, f and f⁻¹ continuous
          depends: [MATH-092]
          source: Topological equivalence

MATH-094  The Euler characteristic of a convex polyhedron satisfies V − E + F = 2.
          formal: V − E + F = 2
          depends: [MATH-083, MATH-013]
          source: Euler (1758)

MATH-095  A closed orientable surface is topologically classified by its genus.
          formal: Σ_g ≅ Σ_h ↔ g = h
          depends: [MATH-093, MATH-094]
          source: Classification of surfaces

MATH-096  The fundamental group of a circle is the integers.
          formal: π₁(S¹) ≅ ℤ
          depends: [MATH-091, MATH-049, MATH-031]
          source: Poincaré (1895)

MATH-097  A smooth manifold is a topological space locally homeomorphic to ℝⁿ with smooth transition maps.
          formal: M smooth manifold: ∀p ∈ M, ∃ chart (U, φ) with smooth transitions
          depends: [MATH-091, MATH-072]
          source: Riemann (1854), Whitney (1936)

MATH-098  The Gauss-Bonnet theorem: the integral of Gaussian curvature over a closed surface equals 2π times the Euler characteristic.
          formal: ∫∫_M K dA = 2πχ(M)
          depends: [MATH-097, MATH-094, MATH-076]
          source: Gauss, Bonnet (1848)
```

## VII. Probability & Combinatorics (MATH-099 → MATH-106)

```
MATH-099  The probability of an event is a real number between 0 and 1, the probability of the certain event is 1, and the probability of a countable union of disjoint events is the sum of their probabilities.
          formal: Kolmogorov axioms: P(A) ∈ [0,1]; P(Ω) = 1; P(∪Aᵢ) = ΣP(Aᵢ) for disjoint Aᵢ
          depends: [MATH-036, MATH-079]
          source: Kolmogorov (1933)

MATH-100  Bayes' theorem: P(A|B) = P(B|A)P(A)/P(B).
          formal: P(A|B) = P(B|A)·P(A) / P(B), where P(B) > 0
          depends: [MATH-099]
          source: Bayes (1763), Laplace

MATH-101  The law of large numbers: the sample mean converges to the expected value as sample size grows.
          formal: (1/n)Σᵢ Xᵢ →ᵖ E[X] as n → ∞
          depends: [MATH-099, MATH-067]
          source: Bernoulli (1713), Khinchin (1929)

MATH-102  The central limit theorem: the sum of many independent, identically distributed random variables approaches a normal distribution.
          formal: (Σᵢ Xᵢ − nμ)/(σ√n) →ᵈ N(0,1) as n → ∞
          depends: [MATH-101, MATH-043]
          source: de Moivre (1733), Laplace, Lyapunov

MATH-103  The number of ways to choose k items from n is C(n,k) = n!/(k!(n−k)!).
          formal: (n choose k) = n! / (k!(n−k)!)
          depends: [MATH-019, MATH-064]
          source: Pascal (1654)

MATH-104  The binomial theorem: (a + b)ⁿ = Σₖ C(n,k) aⁿ⁻ᵏbᵏ.
          formal: (a + b)ⁿ = Σₖ₌₀ⁿ (n choose k) aⁿ⁻ᵏ bᵏ
          depends: [MATH-103, MATH-017]
          source: Newton (generalized), Pascal

MATH-105  The pigeonhole principle: if n items are placed into m containers with n > m, at least one container holds more than one item.
          formal: f: A → B, |A| > |B| → ∃b ∈ B: |f⁻¹(b)| > 1
          depends: [MATH-013, MATH-025]
          source: Dirichlet (1834)

MATH-106  The inclusion-exclusion principle: |A₁ ∪ ... ∪ Aₙ| = Σ|Aᵢ| − Σ|Aᵢ ∩ Aⱼ| + ... + (−1)ⁿ⁺¹|A₁ ∩ ... ∩ Aₙ|.
          formal: |∪ᵢAᵢ| = Σ(−1)ᵏ⁺¹ Σ|Aᵢ₁ ∩ ... ∩ Aᵢₖ|
          depends: [MATH-018, MATH-031]
          source: De Moivre, Sylvester
```

## VIII. Foundational Limits & Meta-Mathematics (MATH-107 → MATH-111)

```
MATH-107  Any consistent formal system capable of expressing basic arithmetic contains statements that are true but unprovable within the system.
          formal: If F is consistent and sufficiently strong, ∃φ: F ⊬ φ and F ⊬ ¬φ, yet φ is true in ℕ
          depends: [MATH-017, MATH-003]
          source: Gödel's First Incompleteness Theorem (1931)

MATH-108  No consistent formal system capable of expressing basic arithmetic can prove its own consistency.
          formal: If F is consistent and sufficiently strong, F ⊬ Con(F)
          depends: [MATH-107]
          source: Gödel's Second Incompleteness Theorem (1931)

MATH-109  There is no general algorithm to determine whether an arbitrary program will halt.
          formal: The halting function H(p, x) = [p(x) halts?] is not computable
          depends: [MATH-107]
          source: Turing (1936), Halting Problem

MATH-110  There exists a well-ordering of the real numbers, but no such ordering can be explicitly constructed.
          formal: Assuming AC: ∃ well-ordering of ℝ; no constructive witness exists
          depends: [MATH-066, MATH-037]
          source: Zermelo (1904); follows from AC

MATH-111  The Continuum Hypothesis is independent of ZFC: it can neither be proved nor disproved from the standard axioms.
          formal: ZFC ⊬ CH and ZFC ⊬ ¬CH
          depends: [MATH-037, MATH-066, MATH-107]
          source: Gödel (1940) consistency; Cohen (1963) independence
```

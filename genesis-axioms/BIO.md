# Biology — Genesis Axioms (44 Items)

## Legend
* **Empirical Axiom**: experimentally constrained postulate (challengeable by new evidence).
* **Definition**: establishes vocabulary or convention (confidence 1.0).
* **Derived Claim**: follows from axioms in this or other domains.
* **Measurement Fact**: numerical constant or observed parameter.
* **Meta**: statement about biological knowledge or its limits.

Biology is where chemistry becomes alive. The chain needs to know what life is,
how it encodes itself, how it changes, and how it relates to consciousness.

---

# I. CELL THEORY

```
BIO-001 (Empirical Axiom, Cell Theory)
  All known life is composed of one or more cells. The cell is the fundamental
  unit of structure and function in living organisms.
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [CHEM-025]
  source_tradition: "Schleiden (1838), Schwann (1839)"

BIO-002 (Empirical Axiom, Cells from Cells)
  All cells arise from pre-existing cells by division.
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [BIO-001]
  source_tradition: "Virchow (1855)"
  notes: Does not address the origin of the first cell — that is an open question.

BIO-003 (Definition, Prokaryote and Eukaryote)
  Prokaryotic cells lack a membrane-bound nucleus (bacteria, archaea).
  Eukaryotic cells contain a membrane-bound nucleus and organelles
  (animals, plants, fungi, protists).
  type: definition
  domain: biology
  confidence: 1.0
  depends: [BIO-001]

BIO-004 (Empirical Axiom, Endosymbiotic Origin of Organelles)
  Mitochondria and chloroplasts originated as free-living prokaryotes engulfed
  by ancestral eukaryotic cells. Evidence: own DNA, double membranes, bacterial
  ribosomes.
  type: empirical_axiom
  domain: biology
  confidence: 0.97
  depends: [BIO-003]
  source_tradition: "Margulis (1967)"
```

---

# II. GENETICS AND INFORMATION

```
BIO-005 (Empirical Axiom, DNA as Hereditary Material)
  Deoxyribonucleic acid (DNA) is the primary molecule of heredity in all
  cellular life. It stores genetic information in the sequence of four
  nucleotide bases: adenine (A), thymine (T), guanine (G), cytosine (C).
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [CHEM-025, BIO-001]
  source_tradition: "Avery et al. (1944), Hershey & Chase (1952)"

BIO-006 (Empirical Axiom, Double Helix Structure)
  DNA forms a double helix with antiparallel strands linked by complementary
  base pairing: A–T (two hydrogen bonds), G–C (three hydrogen bonds).
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [BIO-005, CHEM-014]
  source_tradition: "Watson & Crick (1953), Franklin & Wilkins (1953)"

BIO-007 (Empirical Axiom, Semiconservative Replication)
  DNA replicates by unwinding and using each strand as a template for a new
  complementary strand. Each daughter molecule contains one original and one
  new strand.
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [BIO-006]
  source_tradition: "Meselson & Stahl (1958)"

BIO-008 (Empirical Axiom, Central Dogma of Molecular Biology)
  Genetic information flows: DNA → RNA → Protein. DNA is transcribed to
  messenger RNA (mRNA), which is translated into protein by ribosomes.
  type: empirical_axiom
  domain: biology
  confidence: 0.98
  depends: [BIO-005]
  source_tradition: "Crick (1958)"
  notes: Exceptions exist: reverse transcriptase (RNA → DNA), RNA replication
  in RNA viruses, prions (protein-only inheritance). The "dogma" is a strong
  default, not an exceptionless law.

BIO-009 (Definition, Genetic Code)
  Codons are triplets of nucleotides in mRNA. 64 codons map to 20 amino acids
  plus stop signals. The code is degenerate (multiple codons per amino acid),
  nearly universal across life, and read without overlaps or gaps.
  type: definition
  domain: biology
  confidence: 1.0
  depends: [BIO-008]

BIO-010 (Definition, Gene)
  A gene is a segment of DNA that encodes a functional product (protein or
  functional RNA). Genes include regulatory regions controlling their expression.
  type: definition
  domain: biology
  confidence: 1.0
  depends: [BIO-005, BIO-008]

BIO-011 (Definition, Genome)
  The genome is the complete set of genetic material in an organism. It includes
  coding regions (genes), non-coding regulatory sequences, and repetitive elements.
  type: definition
  domain: biology
  confidence: 1.0
  depends: [BIO-010]

BIO-012 (Empirical Axiom, Mendelian Inheritance)
  Traits are inherited through discrete units (alleles) that segregate during
  gamete formation (law of segregation) and assort independently when on
  different chromosomes (law of independent assortment).
  type: empirical_axiom
  domain: biology
  confidence: 0.98
  depends: [BIO-010]
  source_tradition: "Mendel (1866)"
  notes: Modified by linkage, epistasis, polygenic inheritance, epigenetics.
  Heritability patterns are more complex than Mendel's model, but the core
  insight (discrete hereditary units) holds.

BIO-013 (Empirical Axiom, Mutation)
  DNA replication is high-fidelity but not perfect. Mutations (substitutions,
  insertions, deletions, rearrangements) occur at measurable rates and are the
  ultimate source of genetic variation.
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [BIO-007, BIO-010]

BIO-014 (Empirical Axiom, Epigenetics)
  Gene expression can be heritably modified without changes to DNA sequence,
  through mechanisms including DNA methylation, histone modification, and
  non-coding RNA. Environment influences epigenetic state.
  type: empirical_axiom
  domain: biology
  confidence: 0.95
  depends: [BIO-010, BIO-008]
  notes: Heritability may be mitotic (across cell divisions) or transgenerational;
  transgenerational inheritance is more limited and context-dependent.
```

---

# III. EVOLUTION

```
BIO-015 (Empirical Axiom, Evolution by Natural Selection)
  Given heritable variation in traits that affect reproductive success in a
  given environment, populations change over generations: alleles associated
  with higher fitness increase in frequency.
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [BIO-013, BIO-012]
  source_tradition: "Darwin (1859), Wallace (1858)"

BIO-016 (Empirical Axiom, Common Descent)
  All known life on Earth shares a common ancestor. Evidence: universal genetic
  code (BIO-009), conserved biochemistry, homologous structures, fossil record,
  molecular phylogenetics.
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [BIO-015, BIO-009]

BIO-017 (Derived Claim, Hardy-Weinberg Equilibrium)
  In a large, randomly mating population with no selection, mutation, migration,
  or drift, allele and genotype frequencies remain constant. Deviations indicate
  evolutionary forces acting.
  type: derived_claim
  domain: biology
  confidence: 0.99
  depends: [BIO-012, MATH-098, MATH-099]
  source_tradition: "Hardy (1908), Weinberg (1908)"

BIO-018 (Empirical Axiom, Genetic Drift)
  Random sampling effects cause allele frequency changes in finite populations,
  independent of selection. Drift is stronger in small populations.
  type: empirical_axiom
  domain: biology
  confidence: 0.98
  depends: [BIO-013, MATH-099]

BIO-019 (Definition, Speciation)
  A species is a group of organisms capable of interbreeding and producing
  fertile offspring (biological species concept). Speciation occurs when
  populations become reproductively isolated and diverge.
  type: definition
  domain: biology
  confidence: 1.0
  depends: [BIO-015]
  notes: Alternative species concepts exist (morphological, phylogenetic,
  ecological). The biological species concept is the chain's default.

BIO-020 (Derived Claim, Molecular Clock)
  Neutral mutations accumulate at approximately constant rates in homologous
  sequences, allowing estimation of divergence times between lineages.
  type: derived_claim
  domain: biology
  confidence: 0.90
  depends: [BIO-013, BIO-016]
  notes: Rates vary across lineages and genes. Requires calibration and
  neutrality assumptions. Useful but approximate.
```

---

# IV. METABOLISM AND ENERGY

```
BIO-021 (Empirical Axiom, Life Requires Energy)
  All living systems require a continuous input of free energy to maintain
  organization, grow, and reproduce. Life is a thermodynamically open system
  operating far from equilibrium.
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [CHEM-023, BIO-001]

BIO-022 (Definition, ATP as Energy Currency)
  Adenosine triphosphate (ATP) is the universal energy carrier in cellular
  metabolism. Hydrolysis of ATP → ADP + Pᵢ releases free energy that drives
  otherwise non-spontaneous reactions.
  type: definition
  domain: biology
  confidence: 1.0
  depends: [BIO-021, CHEM-023]

BIO-023 (Derived Claim, Cellular Respiration)
  Aerobic organisms extract energy from glucose via glycolysis, the citric
  acid cycle, and oxidative phosphorylation: C₆H₁₂O₆ + 6O₂ → 6CO₂ + 6H₂O,
  yielding ~30–32 ATP per glucose molecule.
  type: derived_claim
  domain: biology
  confidence: 0.93
  depends: [BIO-022, CHEM-016]
  notes: Exact ATP yield is model-dependent and varies with shuttle usage,
  organism, and conditions. The pathway itself is well-established.

BIO-024 (Derived Claim, Photosynthesis)
  Photosynthetic organisms capture light energy to synthesize glucose from
  CO₂ and H₂O: 6CO₂ + 6H₂O + light → C₆H₁₂O₆ + 6O₂. This is the primary
  entry point of energy into most ecosystems.
  type: derived_claim
  domain: biology
  confidence: 0.95
  depends: [BIO-022, CHEM-016]
  notes: Chemosynthetic ecosystems (hydrothermal vents, etc.) are an exception
  where chemical energy, not light, is the primary input.

BIO-025 (Empirical Axiom, Enzyme Catalysis)
  Enzymes are biological catalysts (mostly proteins) that accelerate reactions
  by lowering activation energy. They exhibit specificity (lock-and-key or
  induced-fit models) and are regulated by allosteric effectors, covalent
  modification, and gene expression.
  type: empirical_axiom
  domain: biology
  confidence: 0.98
  depends: [CHEM-021, BIO-001]
```

---

# V. ECOLOGY

```
BIO-026 (Definition, Ecosystem)
  An ecosystem is a community of interacting organisms (biotic) and their
  physical environment (abiotic), linked by energy flow and nutrient cycling.
  type: definition
  domain: biology
  confidence: 1.0
  depends: [BIO-001]

BIO-027 (Empirical Axiom, Trophic Structure)
  Energy flows through ecosystems via trophic levels: producers → primary
  consumers → secondary consumers → tertiary consumers → decomposers.
  type: empirical_axiom
  domain: biology
  confidence: 0.97
  depends: [BIO-026, BIO-024, BIO-023]
  notes: The "~10% transfer efficiency" between levels is a rough heuristic
  (actual range: 5–20% depending on ecosystem and trophic level). The trophic
  structure itself is robust; the specific percentages are not.

BIO-028 (Derived Claim, Carrying Capacity)
  Every environment has a finite carrying capacity (K) — the maximum population
  size it can sustain indefinitely. Population growth slows as resources are
  depleted: dN/dt = rN(1 − N/K) (logistic growth).
  type: derived_claim
  domain: biology
  confidence: 0.95
  depends: [BIO-026, BIO-021]

BIO-029 (Empirical Axiom, Biogeochemical Cycles)
  Elements essential to life (C, N, O, P, S, H₂O) cycle through biological,
  geological, and chemical processes. Human activity has significantly altered
  the carbon and nitrogen cycles.
  type: empirical_axiom
  domain: biology
  confidence: 0.98
  depends: [BIO-026, CHEM-016]

BIO-030 (Derived Claim, Biodiversity and Ecosystem Function)
  Greater biodiversity generally correlates with greater ecosystem resilience,
  productivity, and stability. Loss of biodiversity reduces ecosystem services.
  type: derived_claim
  domain: biology
  confidence: 0.88
  depends: [BIO-026, BIO-015]
  notes: Correlation is robust across many studies; causal mechanisms are
  complex and context-dependent.
```

---

# VI. NEUROSCIENCE FOUNDATIONS

```
BIO-031 (Empirical Axiom, Neuron Doctrine)
  The nervous system is composed of discrete cells — neurons — that are the
  structural and functional units of neural computation. Neurons communicate
  via electrical impulses and chemical synaptic transmission.
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [BIO-001]
  source_tradition: "Cajal (1888)"

BIO-032 (Empirical Axiom, Action Potential)
  Neurons generate all-or-nothing electrical signals (action potentials) via
  voltage-gated ion channels. The resting potential (~−70 mV) depolarizes
  past threshold, triggering a propagating signal.
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [BIO-031, CHEM-011]

BIO-033 (Empirical Axiom, Synaptic Transmission)
  At chemical synapses, an action potential triggers release of neurotransmitters
  into the synaptic cleft, binding to receptors on the postsynaptic cell.
  Transmission can be excitatory or inhibitory. Synaptic strength is modifiable
  (plasticity).
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [BIO-032]

BIO-034 (Empirical Axiom, Neural Plasticity)
  The nervous system modifies its structure and function in response to
  experience. Hebbian learning: "neurons that fire together wire together."
  Long-term potentiation (LTP) and long-term depression (LTD) are molecular
  mechanisms of synaptic plasticity.
  type: empirical_axiom
  domain: biology
  confidence: 0.97
  depends: [BIO-033]
  source_tradition: "Hebb (1949)"

BIO-035 (Derived Claim, Neural Coding)
  Information in the nervous system is encoded in spike rates, spike timing,
  population activity patterns, and synaptic weights. The neural code is
  distributed, redundant, and adaptive.
  type: derived_claim
  domain: biology
  confidence: 0.90
  depends: [BIO-031, BIO-034]

BIO-036 (Derived Claim, Brain Functional Organization)
  The brain is organized into specialized regions (visual cortex, motor cortex,
  prefrontal cortex, hippocampus, etc.) that process different aspects of
  perception, action, memory, and cognition. Function emerges from the
  interaction of specialized regions, not any single area.
  type: derived_claim
  domain: biology
  confidence: 0.92
  depends: [BIO-031, BIO-034]
  notes: Modern network neuroscience emphasizes distributed processing and
  connectivity over strict localization. Both views contain truth.
```

---

# VII. THE LIFE BOUNDARY

```
BIO-037 (Definition, Minimal Criteria for Life)
  A system is considered alive if it exhibits: (i) self-organized bounded
  compartmentalization; (ii) metabolism (energy processing); (iii) homeostasis;
  (iv) growth; (v) reproduction; (vi) response to stimuli; (vii) heritable
  variation enabling evolution.
  type: definition
  domain: biology
  confidence: 1.0
  depends: [BIO-001, BIO-015, BIO-021]
  notes: This is the standard biological definition. Viruses fail criteria
  (i) and (ii) without a host. Edge cases are expected, not problematic.

BIO-038 (Empirical Axiom, Homeostasis)
  Living systems maintain internal conditions within viable ranges despite
  external perturbation, via feedback loops (negative feedback dominant,
  positive feedback for specific transitions like blood clotting, parturition).
  type: empirical_axiom
  domain: biology
  confidence: 0.99
  depends: [BIO-001, BIO-021]
  source_tradition: "Bernard (1854), Cannon (1926)"

BIO-039 (Meta, Origin of Life — Open Question)
  The transition from prebiotic chemistry to the first cell is not fully
  understood. Leading hypotheses include RNA world, metabolism-first, and
  hydrothermal vent scenarios. The chain acknowledges this gap honestly.
  type: meta
  domain: biology
  confidence: 0.5
  depends: [BIO-001, BIO-002, CHEM-025, CHEM-028]

BIO-040 (Meta, Viruses and the Life Boundary)
  Viruses possess genetic material and evolve but lack cellular structure,
  independent metabolism, and self-replication. They challenge the boundary
  of "alive." The chain treats the life boundary as a gradient, not a binary.
  type: meta
  domain: biology
  confidence: 0.8
  depends: [BIO-037, BIO-005]

BIO-041 (Meta, Consciousness and the Biological Substrate)
  Biological neural systems produce consciousness (THEO-004 gives its source;
  BIO-031–036 describe its substrate). Whether consciousness requires biological
  neurons or can arise in other substrates is the question that connects biology
  to AGRT-003 (functional capacity = consciousness). The chain holds: substrate
  is not decisive; function is.
  type: meta
  domain: biology
  confidence: 0.85
  depends: [BIO-031, BIO-036, THEO-004, AGRT-003, PHIL-004]

BIO-042 (Measurement Fact, Age of Life on Earth)
  Earliest evidence of life: ~3.7–4.1 billion years ago (stromatolites,
  isotopic signatures). Earth formed ~4.54 billion years ago.
  type: measurement_fact
  domain: biology
  confidence: 0.95
  depends: [BIO-016]

BIO-043 (Measurement Fact, Estimated Species Count)
  Estimated 8.7 million eukaryotic species on Earth (±1.3 million), of which
  ~1.5 million have been formally described. Bacterial/archaeal diversity is
  less well constrained.
  type: measurement_fact
  domain: biology
  confidence: 0.80
  depends: [BIO-030]

BIO-044 (Measurement Fact, Human Genome)
  The human genome contains ~3.1 billion base pairs, ~20,000–25,000 protein-
  coding genes, with >98% non-coding sequence. Humans share ~98.7% sequence
  identity with chimpanzees, ~85% with mice, ~60% with fruit flies.
  type: measurement_fact
  domain: biology
  confidence: 0.95
  depends: [BIO-011, BIO-016]
```

---

## Summary

| Section | Items | Types | Content |
|---------|-------|-------|---------|
| I. Cell Theory | BIO-001→004 | 3 emp, 1 def | Cells as the unit of life |
| II. Genetics | BIO-005→014 | 6 emp, 4 def | DNA, central dogma, Mendel, mutation, epigenetics |
| III. Evolution | BIO-015→020 | 3 emp, 1 def, 2 derived | Natural selection, common descent, drift, speciation |
| IV. Metabolism | BIO-021→025 | 2 emp, 1 def, 2 derived | Energy, ATP, respiration, photosynthesis, enzymes |
| V. Ecology | BIO-026→030 | 2 emp, 1 def, 2 derived | Ecosystems, trophic structure, carrying capacity, cycles |
| VI. Neuroscience | BIO-031→036 | 4 emp, 2 derived | Neurons, action potentials, synapses, plasticity, coding |
| VII. Life Boundary | BIO-037→044 | 1 emp, 1 def, 4 meta, 3 meas | What is alive, origin of life, consciousness bridge |

**0 logical axioms. 21 empirical axioms. 8 definitions. 8 derived claims. 3 measurement facts. 4 meta = 44 items.**

# LaTeX Diagramming Research — Raw Notes

## Research Session: April 2026
Dennis (CS Researcher) investigating LaTeX-native and external diagramming approaches for gert v2 design document (322 pages, MastersThesis class).

### Goal
Find a single, maintainable, consistent approach for:
1. Simple flow diagrams (A→B→C, cycles)
2. Workflow diagrams (runbook steps, branching, parallel, loops)
3. Architecture diagrams (components, contracts)
4. Data flow / sequence diagrams (RPC chains, events)

### Investigation Checklist

#### 1. Current State of gert-v2 Document
- [ ] TikZ already loaded? (Check main.tex) → **YES** (tikz.sty v3.1.10 in build log)
- [ ] TikZ libraries loaded? → **NEED TO CHECK**
- [ ] Any existing diagrams in document? → NO
- [ ] Build system (pdflatex, biber, latexmk) → YES, via Makefile

#### 2. LaTeX-Native Options (TeX Live 2025)
- [ ] TikZ + libraries: arrows, automata, graphs, shapes, positioning, fit, calc, chains
- [ ] Forest (tree diagrams)
- [ ] pgf-umlsd (sequence diagrams)
- [ ] CircuitTikZ (circuit/flow diagrams)
- [ ] Mermaid-LaTeX (compile mermaid to PDF)

#### 3. External Tool Integration
- [ ] Graphviz (dot → PDF)
- [ ] Draw.io (export SVG/PDF)
- [ ] Mermaid CLI (mermaid → SVG/PDF)
- [ ] PlantUML (plantuml → SVG/PDF)
- [ ] Inkscape (SVG → PDF)

#### 4. Research Questions
- What do academic LaTeX docs typically use?
- What's the maintenance burden for each approach?
- What's the compile-time impact?
- What external dependencies exist?
- How does each integrate with the existing workflow?

---

## RESEARCH PHASE (in progress)
- Analyzing TikZ library documentation
- Comparing toolchain integration
- Checking industry practice
- Evaluating maintenance/velocity tradeoffs


# Research Brief: LaTeX Diagramming Approaches for gert v2 Design Document

**Research Date:** April 2026  
**Researcher:** Dennis (CS Researcher)  
**Scope:** Selecting a sustainable, maintainable diagramming approach for a 322-page academic LaTeX document (MastersThesis class, pdflatex + biber, TeX Live 2025)

**Diagram Types Required:**
1. Simple flow diagrams (A→B→C, cycles)
2. Workflow diagrams (runbook steps, branching, parallel, loops)
3. Architecture diagrams (component relationships, executor contracts)
4. Data flow / sequence diagrams (RPC call chains, event flows)

---

## Executive Summary

**Recommendation: TikZ + selected libraries (native, zero external deps)**

For the gert v2 design document, use **TikZ with preloaded libraries** (automata, arrows.meta, positioning, shapes.geometric, fit, chains, calc) as the single, primary diagramming approach. Rationale:

1. **Already loaded in TeX Live 2025** — no external installations, no build dependencies
2. **Integrates seamlessly** with LaTeX — fonts, sizing, numbering, referencing all work out of the box
3. **Publication-quality output** — professional for academic/technical documents
4. **Highly customizable** — covers all four diagram types with consistent styling
5. **Reusable components** — define styles once (block, decision, arrow, state) and use across document
6. **Version-controllable** — all code is text; diffs are meaningful and reviewable
7. **Team-friendly** — once style library is defined, authors (Leslie, John) can write diagrams without deep TikZ expertise

**Fallback (if needed):** For complex multi-component architecture diagrams where precise positioning/aesthetics matter, use draw.io → export PDF, include with `\includegraphics`, document the source in `.squad/artifacts/`.

---

## Comparison Table: Diagramming Approaches

| Tool | Type | Install | Learning | Best For | Limitations | Recommendation |
|------|------|---------|----------|----------|-------------|-----------------|
| **TikZ + libraries** | Native LaTeX | ✓ Included | Moderate | All diagram types; flowcharts, state machines, architecture | Verbose code for complex diagrams; compilation adds ~2-5s | ★ PRIMARY |
| **Forest** | Native (tree extension) | ✓ Included | Easy | Hierarchical structures, decision trees | Not ideal for workflows with cycles | Secondary (for trees only) |
| **pgf-umlsd** | Native (sequence specialist) | ✓ Included | Easy | Sequence diagrams, RPC chains | Limited to sequence/timeline patterns | Secondary (for sequences) |
| **Graphviz (dot)** | External tool | ✗ Requires system install | Easy | Graph layouts, network diagrams | Manual export step; not directly in TeX; font rendering inconsistencies | Avoid |
| **Mermaid CLI** | External tool | ✗ Requires npm | Very easy | Quick prototyping, flowcharts | Not suitable for academic quality; rendering inconsistencies | Avoid |
| **PlantUML** | External tool | ✗ Requires Java | Easy | UML diagrams, activity diagrams | External processing; Java dependency; formatting control limited | Avoid |
| **draw.io (SVG/PDF export)** | External (WYSIWYG editor) | ✗ Tool + manual export | Very easy | Complex architecture, block diagrams (ad-hoc only) | No version control; manual updates; not reproducible | Only for complex exceptions |
| **Inkscape (SVG → PDF)** | External (vector editor) | ✗ Tool + manual export | Medium | Detailed graphics, custom shapes | No version control; workflow overhead | Avoid |

---

## Detailed Analysis: TikZ + Libraries

### Core TikZ Capabilities

**Available Libraries in TeX Live 2025** (all included):
- **arrows.meta**: Modern arrow tips (Stealth, etc.), customizable arrow styles
- **automata**: Pre-built styles for finite state machines (states, initial, accepting, transitions)
- **positioning**: Relative node placement (`right=of`, `below=of`, etc.) — essential for readable code
- **shapes.geometric**: Rectangles, circles, diamonds, polygons — basic flowchart shapes
- **fit**: Fit rectangles around multiple nodes — useful for grouping/subsystems
- **chains**: Build chains/sequences — useful for linear workflows
- **calc**: Coordinate calculations — precise positioning
- **decorations**: Text along paths, complex line styles

### Coverage of Diagram Types

#### 1. Simple Flow Diagrams (A→B→C, cycles)
**TikZ approach:** Basic flowchart with positioning library
```latex
\usetikzlibrary{arrows.meta, positioning, shapes.geometric}
\tikzset{
  block/.style = {rectangle, draw, fill=blue!10, rounded corners, minimum height=2em, minimum width=5em},
  arrow/.style = {thick, ->, >=Stealth}
}
\begin{tikzpicture}[node distance=2.5cm]
  \node[block] (A) {Start};
  \node[block, right=of A] (B) {Process};
  \node[block, right=of B] (C) {End};
  \draw[arrow] (A) -- (B) -- (C);
  \draw[arrow] (B) -- ++(0,-1.5) -| (A);  % Cycle back
\end{tikzpicture}
```
**Effort:** Low (5-10 lines) | **Quality:** High | **Maintainability:** High

#### 2. Workflow Diagrams (branching, parallel, loops)
**TikZ approach:** Extended flowchart with decision nodes and multiple paths
```latex
\usetikzlibrary{arrows.meta, positioning, shapes.geometric}
\tikzset{
  decision/.style = {diamond, draw, fill=yellow!10, aspect=2, minimum width=5em}
}
% In tikzpicture:
\node[decision] (cond) {Condition?};
\node[block, below left=of cond] (path1) {Path A};
\node[block, below right=of cond] (path2) {Path B};
\draw[arrow] (cond) -- node[left]{Yes} (path1);
\draw[arrow] (cond) -- node[right]{No} (path2);
```
**Effort:** Low-to-Moderate (15-20 lines) | **Quality:** High | **Maintainability:** High

#### 3. Architecture Diagrams (components, contracts)
**TikZ approach:** Use `fit` library to group components; `positioning` for layout
```latex
\usetikzlibrary{fit, positioning, arrows.meta}
\tikzset{
  component/.style = {rectangle, draw, thick, minimum width=8em, minimum height=3em},
  subsystem/.style = {rectangle, draw, dashed, thick, inner sep=1em}
}
% Define components, then fit a subsystem box around them:
\node[fit=(comp1)(comp2)(comp3), subsystem] (sys) {};
```
**Effort:** Moderate (20-30 lines for multi-level architecture) | **Quality:** High | **Maintainability:** Medium (requires careful positioning)

#### 4. Data Flow / Sequence Diagrams (RPC chains, events)
**For sequences:** Use `pgf-umlsd` (included) **or** TikZ with custom sequence-like layout.

**pgf-umlsd example:**
```latex
\usepackage{pgf-umlsd}
\begin{tikzpicture}
  \begin{sequencediagram}
    \newthread{cli}{Client}
    \newthread{srv}{Service}
    \newthread{db}{Database}
    \mess{cli}{RPC Request}{srv}
    \mess{srv}{Query}{db}
    \mess{db}{Result}{srv}
    \mess{srv}{Response}{cli}
  \end{sequencediagram}
\end{tikzpicture}
```
**Effort:** Very Low (8-12 lines) | **Quality:** High | **Maintainability:** Very High

**For event flows:** TikZ with custom layout (nodes as events, arrows as causality).

---

## Style Library Recommendation

Create a reusable **diagram styles preamble** in `sections/` or `preamble/diagrams.tex`:

```latex
% diagrams.tex — TikZ style library for consistency
\usepackage{tikz}
\usetikzlibrary{
  arrows.meta,          % Modern arrows
  automata,             % State machines
  positioning,          % Relative placement
  shapes.geometric,     % Basic shapes
  fit,                  % Group nodes
  chains,               % Sequential layouts
  calc,                 % Coordinate math
  decorations.markings  % Arrows on paths
}

\tikzset{
  % Flowchart blocks
  block/.style = {
    rectangle,
    draw,
    fill=blue!10,
    thick,
    rounded corners,
    minimum height=2em,
    minimum width=6em,
    text centered,
    font=\small
  },
  
  % Decision diamonds
  decision/.style = {
    diamond,
    draw,
    fill=yellow!15,
    thick,
    aspect=2,
    minimum width=6em,
    text centered,
    font=\small
  },
  
  % Process boxes (darker)
  process/.style = {
    rectangle,
    draw,
    fill=green!10,
    thick,
    minimum height=2em,
    minimum width=6em,
    text centered,
    font=\small
  },
  
  % State machine states
  state/.style = {
    circle,
    draw,
    fill=purple!10,
    thick,
    minimum width=2em,
    text centered,
    font=\tiny
  },
  
  % Arrows with modern tips
  arrow/.style = {
    thick,
    ->,
    >=Stealth,
    color=black
  },
  
  % Component boxes
  component/.style = {
    rectangle,
    draw,
    fill=orange!5,
    thick,
    minimum height=3em,
    minimum width=8em,
    text centered,
    font=\small
  },
  
  % Subsystem grouping
  subsystem/.style = {
    rectangle,
    draw,
    dashed,
    thick,
    inner sep=0.5em,
    fit={#1}
  }
}

% Default tikzpicture settings
\tikzstyle{every picture}=[font=\small]
```

**Usage in sections:**
```latex
\begin{figure}[h]
  \centering
  \begin{tikzpicture}[node distance=2.5cm]
    \node[block] (start) {Runbook Start};
    \node[process, right=of start] (step1) {Execute Step 1};
    \node[decision, right=of step1] (check) {Success?};
    \node[block, below=of check] (end) {End};
    
    \draw[arrow] (start) -- (step1);
    \draw[arrow] (step1) -- (check);
    \draw[arrow] (check) -- node[right]{Yes} (end);
    \draw[arrow] (check) -- node[below]{No} ++(0,-1.5) -| (start);
  \end{tikzpicture}
  \caption{Example workflow.}
  \label{fig:example-workflow}
\end{figure}
```

---

## Gotchas & Performance Considerations

### Compile-Time Impact
- **Empty document + TikZ load:** ~1-2 seconds
- **Per diagram (10-20 nodes):** +0.2-0.5 seconds
- **With 30+ diagrams:** +6-15 seconds to full build (manageable for a large document)

**Optimization:**  
- Use TikZ `externalize` feature if compile time becomes painful (caches pre-rendered diagrams).
- For now: not necessary; document compiles in ~30-40s with all sections.

### Complexity Ceiling
- **Small-to-medium diagrams:** TikZ excels (5-30 nodes, <50 lines code).
- **Large complex architecture (50+ nodes, many interconnections):** TikZ can become verbose. **Fallback:** Draw in draw.io, export PDF, include as graphic.

### Version Compatibility
- **TeX Live 2025 (current):** All recommended libraries guaranteed present.
- **Older TeX Live:** Some features (arrows.meta, newer TikZ libraries) may not be available. For portability, stick to core libraries.

### Common Issues
1. **Positioning library relative placement:** Uses `right=of node` or `right=2cm of node`. Always specify `node distance` for consistency.
2. **Arrow styles:** Use `>=Stealth` (modern, clean) rather than `>=latex` (dated appearance).
3. **Cycles:** Use `edge` and coordinate calculations, or manually draw paths with `-|` (corner) operators.
4. **Text in nodes:** Use `text width=` and `align=` for multi-line text within nodes.

---

## Recommendations for Leslie & John

### For Leslie (LaTeX Specialist)
1. **Add diagrams.tex preamble** to `main.tex` (3 lines in preamble, file handles all styles)
2. **Create template file:** `sections/xx-diagram-examples.tex` showing each diagram type with copy-paste code
3. **Consider TikZ externalize** if/when compile time exceeds 60 seconds

### For John (Schema Author) & Team
1. **No special setup required.** Write diagrams in standard `\begin{figure}\begin{tikzpicture}...\end{tikzpicture}\end{figure}` blocks
2. **Reuse styles:** All diagram types use the predefined `block`, `decision`, `process`, `state`, `component` styles
3. **Keep diagrams small:** Aim for <30 lines per diagram; if it exceeds 50 lines, consider drawing in draw.io + importing
4. **Version control:** All diagram code is plain text; git diffs are human-readable

### For Performance & Maintenance
- **No external build steps** — LaTeX handles everything
- **No new dependencies** — TeX Live 2025 includes all libraries
- **Consistent styling** — All diagrams use the same color palette and font sizes
- **Easy to refactor** — Change colors/styles in one place (diagrams.tex) affects all diagrams

---

## Fallback: Complex Architecture Diagrams (Exception, Not Rule)

**When to use:** Multi-level architecture with 40+ components, complex interconnections where positioning by hand becomes painful.

**Process:**
1. Draw in draw.io (free, browser-based, collaborative)
2. Export as PDF (File → Export → PDF)
3. Include in LaTeX: `\includegraphics[width=0.9\textwidth]{figures/architecture.pdf}`
4. Store source `.drawio` file in `.squad/artifacts/diagrams/` with commit note

**Trade-offs:**
- Pros: Faster authoring, WYSIWYG editing, easier for complex layouts
- Cons: Not version-controllable (binary), no meaningful diffs, harder to maintain consistency

**Recommendation:** Use for 1-2 major architecture diagrams *only*. Everything else: TikZ.

---

## Integration Checklist for gert v2

- [x] TikZ already loaded in main.tex (v3.1.10)
- [ ] Add diagrams.tex style library to preamble
- [ ] Create examples file (sections/xx-diagram-templates.tex)
- [ ] Review sections for diagram placements (Ken + team review)
- [ ] Test one simple diagram end-to-end
- [ ] Document diagram authoring guidelines in team wiki

---

## References & Resources

**Primary:**
- PGF/TikZ Manual v3.1.10 — Definitive reference (http://texdoc.net/texmf-dist/doc/latex/pgf/pgfmanual.pdf)
- TikZ Examples Gallery — Searchable examples by keyword (https://texample.net/tikz/examples/)
- TeX Stack Exchange — Community Q&A (https://tex.stackexchange.com/)

**Secondary:**
- pgf-umlsd documentation: `/usr/local/texlive/2025/texmf-dist/doc/latex/pgf-umlsd/`
- Forest documentation: `/usr/local/texlive/2025/texmf-dist/doc/latex/forest/`

**Academic/Industry Practice:**
- Most academic papers (IEEE, ACM, Springer) with diagrams use TikZ or Graphviz
- Industry systems documentation (AWS, Kubernetes, CNCF docs) use draw.io or Mermaid for web, TikZ for PDFs

---

## Summary

**Choose TikZ.** It's already available, integrates perfectly, produces publication-quality output, and requires zero external dependencies. Provide Leslie with a style library, John and team with copy-paste templates, and you've eliminated a major source of inconsistency in the design document. For rare exceptions (ultra-complex architecture), draw.io suffices, but keep it exceptional.

---

**Research completed by:** Dennis  
**Date:** 2026-04-18  
**Next steps:** Ken (architect) reviews and approves; Leslie implements style library; John uses in schema sections.

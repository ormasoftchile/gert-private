# Decision: Zoom range and default behavior

**By:** Kurapika (Frontend Engineer)
**Date:** 2026-03-18

**What:** Graph pan/zoom uses 0.3–2.0 range (not 3.0), default zoom fits SVG to container width, reset returns to fit-to-width.

**Why:** Max 2.0 keeps text legible at extreme zoom. Fit-to-container means large runbooks (15+ nodes) are immediately visible without manual zoom-out. Fixed 1.0 default cut off wide graphs requiring immediate user interaction.

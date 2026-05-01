### RunGraph wired into harness (harness-first)
**By:** Brian (Cristian directive)
**What:** RunGraph added to TUIApp as parallel backing store. Harness accessors delegate to RunGraph. TUI rendering still uses flat state — not changed.
**Why:** User directive: harness first. Validate RunGraph integration via tests before touching rendering.

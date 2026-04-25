# ClickUp Clone — Full Milestone Plan

Staged delivery of a comprehensive, pixel-close ClickUp clone on the existing Go + React + Postgres stack at `c:\Users\pc\Downloads\clickup`. I implement one milestone at a time, pause for review, then continue.

**Legend:** ✅ shipped · 🔨 in progress · ⏭️ queued

---

## Architectural locks (set in stone M1)

1. Per-list statuses replace `tasks.status TEXT` — Kanban drag-drop depends on it.
2. Task ordering uses `position DOUBLE PRECISION`, gap-inserted `(prev + next) / 2`; never rewrite N rows on a drop.
3. Custom fields precede Views so filters/sort/group don't need a rewrite.
4. Notifications is a spine feature — generic shape `(actor_id, recipient_id, entity_type, entity_id, verb, payload_jsonb, read_at)` from day one.
5. WS envelope is versioned: `{v, type, workspace_id, actor_id, entity_id, ts, payload}` with room naming `ws:/list:/task:/user:`.
6. RBAC policy layer with role hierarchy `owner > admin > member > guest` and resource-scoped checks (`RequireWorkspaceRole`, `RequireSpaceRole`, `RequireListAccess`, `RequireTaskAccess`).

## Trademark note

Pixel-close means matching layout/spacing/density/typography + the purple-pink brand ramp. The **wordmark + logo must be substituted** with our own (e.g. "Clikr"). No ClickUp marketing illustrations, empty-state art, or mascot.

## Library picks (locked)

| Area | Pick |
|---|---|
| Drag & drop | dnd-kit (`@dnd-kit/core` + `sortable` + `modifiers`) |
| Rich text (comments, descriptions, Docs) | TipTap v2 |
| Realtime collab (Docs/Whiteboard) | Y.js + y-websocket |
| Whiteboard | tldraw v2 SDK |
| Gantt | custom SVG + date-fns (MVP), frappe-gantt optional |
| Calendar | react-big-calendar |
| Table/List view | TanStack Table v8 + @tanstack/react-virtual |
| Charts | Recharts → ECharts for funnels/heatmaps |
| File upload | react-dropzone + tus-js-client |
| Command palette / hotkeys | cmdk + react-hotkeys-hook |
| Search | Postgres FTS → Meilisearch if needed |
| Forms | react-hook-form + zod |

---

## Milestones

### ✅ M1 — Complete the core

**Shipped.** Drag-drop Kanban, attachments, tags, subtasks, multi-assignees, per-list statuses, member management, notifications engine with live toast + unread badge, audit log, WS v1 envelope.

Routes: `/lists/:id/statuses`, `/tasks/:id/attachments`, `/tasks/:id/tags`, `/tasks/:id/assignees`, `/tasks/:id/position`, `/tasks/:id/subtasks`, `/workspaces/:id/members`, `/spaces/:id/members`, `/me/notifications`, `/workspaces/:id/audit`.

Frontend routes added: `/workspaces/:id/members`, `/workspaces/:id/audit`.

### ✅ M2 — Views & Power-filtering

**Shipped.** Views first-class entity (name/kind/ViewConfig JSONB). Five view kinds: Board (existing Kanban), List (grouped rows with inline complete), Calendar (react-big-calendar), Gantt (custom SVG timeline with weekend shading), Table (TanStack Table with sortable columns). Saved views synced to `?view=<id>` URL param. FilterBar with filter/sort/group-by chips. RBAC policy layer with role hierarchy + resource-scoped checks.

Routes: `/lists/:id/views`, `/spaces/:id/views`, `/views/:id`.

### ✅ M3 — Custom fields & Time tracking

**Shipped.** 12 custom-field kinds (text, number, dropdown, labels, date, checkbox, url, email, phone, money, progress, people). `CustomFieldsManager` dialog + polymorphic `CustomFieldInput` + `CustomFieldsBlock` on task detail. Time tracking: start/stop timer with live elapsed ticker, single-running-timer-per-user enforcement, manual log modal, per-task entries list with totals and billable badge, workspace-level time-report page with date range + per-teammate aggregation.

Routes: `/lists/:id/custom-fields`, `/workspaces/:id/custom-fields`, `/custom-fields/:id`, `/tasks/:id/custom-values`, `/tasks/:id/time-entries`, `/tasks/:id/timer/start`, `/time-entries/:id/stop`, `/me/timer`, `/workspaces/:id/time-report`.

Frontend route added: `/workspaces/:id/time-report`.

### ✅ M4 — Dependencies, Automations, Recurring tasks

**Shipped.** Task dependencies with `waiting_on`/`blocking` edges + recursive-CTE cycle detection (backend returns 409). Async automation engine: 5 triggers (`task.created`/`status_changed`/`assigned`/`completed`/`due_soon`), condition eval (`priority`/`status_id`/`assignee_id` with `eq/neq/gte/lte`), 5 action types (`change_status`/`assign_user`/`add_tag`/`add_comment`/`notify`). Recurring tasks via RRULE (rrule-go) — completing a recurring task auto-spawns the next occurrence with inherited assignees/priority/start-offset.

Routes: `/tasks/:id/dependencies`, `/workspaces/:id/automations`, `/lists/:id/automations`, `/automations/:id`.

Frontend: Dependencies panel (waiting-on + blocking sections, cycle-error surfacing), Recurrence picker with 6 RRULE presets + custom input, full rule builder at `/workspaces/:id/automations` (trigger/conditions/actions, enable toggle, run count, last-fired).

### ✅ M5 — Docs, Chat, Search

**Shipped.** Docs with nested parent→child tree, TipTap editor with 600ms debounced save, remote-edit refresh skipped while editor is focused. Chat with workspace / space channels, threaded replies, @mention resolution by email, real-time per-channel broadcast. Postgres FTS via generated `tsvector` columns + GIN indexes on tasks / docs / comments / messages — cross-entity search with `ts_headline` highlighted snippets, cmdk command palette triggered by Cmd/Ctrl+K.

Routes: `/workspaces/:id/docs`, `/docs/:id`, `/docs/:id/children`, `/workspaces/:id/channels`, `/spaces/:id/channels`, `/channels/:id`, `/channels/:id/members`, `/channels/:id/messages`, `/messages/:id`, `/messages/:id/replies`, `/channels/:id/read`, `/workspaces/:id/search`.

Frontend routes added: `/workspaces/:id/docs`, `/workspaces/:id/chat`.

### 🔨 M6 — Dashboards, Goals, Sprints (next)

- **Dashboards**: widget framework — burndown, velocity, task-count-by-status, time-per-user. Recharts.
- **Goals**: hierarchy of goal → target (number, currency, true/false, task-based).
- **Sprints**: sprint list with start/end + points + auto-rollover of open tasks to next sprint.

- **Whiteboards**: tldraw v2 SDK integration, shared boards scoped to a space. Snapshot persisted as JSONB; live-collab via the existing WS hub (last-write-wins on debounced save, proper Y.js collab deferred to M8).
- **Forms**: public intake endpoint `POST /public/forms/:id/submit` → creates a task in the linked list. Form builder UI: label/description + field array (text, textarea, number, select, checkbox, email), required/optional toggles, public URL + embed snippet.
- **Templates**: workspace-scoped templates for `list`, `doc`, and `task` kinds. Apply-template button clones the template's snapshot into real entities. Templates can be promoted from existing entities ("save as template").

<!-- ### ⏭️ M8 — Polish, Integrations, AI

- Command palette + global cross-entity search UI (cmdk).
- Email notifications (SMTP/SES) pluggable into notify dispatcher.
- SSO: Google + GitHub OAuth.
- Import: Trello + Asana CSV.
- AI assist: summarize thread, generate subtasks via Anthropic API.
- Mobile-responsive pass, a11y audit, dark mode, code splitting for the big chunks. -->

---

## Workflow

After each milestone I post a summary + demo script and wait for sign-off. You can redirect scope, re-order, or drop features at any boundary. The plan file is the source of truth — I update it after each milestone.

**Current status (last update this session): M1–M7 shipped. M8 queued.**

export interface User {
  id: string
  email: string
  name: string
  avatar_url?: string
  created_at: string
}

export interface Workspace {
  id: string
  name: string
  slug: string
  owner_id: string
  created_at: string
}

export interface Space {
  id: string
  workspace_id: string
  name: string
  color?: string
  position: number
  archived: boolean
}

export interface Folder {
  id: string
  space_id: string
  name: string
  position: number
  archived: boolean
}

export interface List {
  id: string
  space_id: string
  folder_id?: string
  name: string
  position: number
  archived: boolean
}

export interface Status {
  id: string
  list_id: string
  name: string
  color: string
  category: 'active' | 'done' | 'closed' | string
  order_index: number
  created_at: string
}

export interface Task {
  id: string
  list_id: string
  parent_task_id?: string
  name: string
  description: string
  status: string
  status_id?: string
  priority: number
  position: number
  assignee_id?: string
  creator_id?: string
  due_at?: string
  start_at?: string
  completed_at?: string
  archived: boolean
  recurring_rule?: string | null
  recurring_parent_id?: string
  created_at: string
  updated_at: string
}

export interface Tag {
  id: string
  workspace_id: string
  name: string
  color?: string
}

export interface Attachment {
  id: string
  task_id: string
  uploader_id?: string
  filename: string
  path: string
  size_bytes: number
  mime_type: string
  created_at: string
}

export interface Comment {
  id: string
  task_id: string
  author_id?: string
  body: string
  created_at: string
  updated_at: string
}

export type TaskStatus = 'open' | 'in_progress' | 'review' | 'completed' | 'cancelled'

export interface Notification {
  id: string
  user_id: string
  actor_id?: string
  kind: string
  entity_type?: string
  entity_id?: string
  payload: Record<string, unknown>
  read_at?: string
  created_at: string
}

export interface Member {
  workspace_id?: string
  space_id?: string
  user_id: string
  email?: string
  name?: string
  avatar_url?: string
  role: string
  created_at: string
}

export interface AuditEntry {
  id: string
  workspace_id?: string
  actor_id?: string
  entity_type: string
  entity_id?: string
  verb: string
  before?: unknown
  after?: unknown
  created_at: string
}

export type ViewKind = 'board' | 'list' | 'calendar' | 'gantt' | 'table' | 'timeline'

export interface FilterClause {
  field: string     // status_id | priority | assignee_id | tag_id | due_at | custom:<id>
  op: string        // eq | neq | in | nin | gt | lt | between | empty | notempty | before | after
  value?: unknown
}

export interface SortField {
  field: string
  desc?: boolean
}

export interface ViewConfig {
  filters?: FilterClause[]
  sort?: SortField[]
  group_by?: string
  visible_cols?: string[]
  is_shared?: boolean
  creator_id?: string
  options?: Record<string, unknown>
}

export interface View {
  id: string
  list_id?: string
  space_id?: string
  name: string
  kind: ViewKind
  config: ViewConfig
  created_at: string
}

export type FieldKind =
  | 'text' | 'number' | 'dropdown' | 'labels' | 'date' | 'checkbox'
  | 'url' | 'email' | 'phone' | 'money' | 'progress' | 'people'

export interface FieldOption {
  id: string
  label: string
  color?: string
}

export interface FieldConfig {
  options?: FieldOption[]      // dropdown / labels
  placeholder?: string          // text
  max_length?: number           // text
  min?: number                  // number / progress
  max?: number                  // number / progress
  step?: number                 // number
  precision?: number            // number / money
  currency?: string             // money (default 'USD')
  include_time?: boolean        // date
  multiple?: boolean            // people
}

export interface CustomField {
  id: string
  workspace_id: string
  list_id?: string
  name: string
  kind: FieldKind
  config: FieldConfig
  required: boolean
  order_index: number
  created_at: string
  updated_at: string
}

export interface CustomValue {
  task_id: string
  field_id: string
  value: unknown
}

export interface TimeEntry {
  id: string
  task_id: string
  user_id: string
  started_at: string
  stopped_at?: string
  duration_s?: number
  note: string
  billable: boolean
  created_at: string
  updated_at: string
}

export interface TimeReportBucket {
  user_id: string
  email?: string
  name?: string
  task_count: number
  total_s: number
  billable_s: number
}

export interface TaskDependency {
  task_id: string
  depends_on_id: string
  kind: 'waiting_on' | 'blocks' | string
  created_at: string
}

export interface DependencyGraph {
  waiting_on: TaskDependency[]
  blocking: TaskDependency[]
}

export type AutomationTriggerType =
  | 'task.status_changed' | 'task.created' | 'task.assigned' | 'task.completed' | 'task.due_soon'

export interface AutomationTrigger {
  type: AutomationTriggerType
  to_status_id?: string
  user_id?: string
  lead_days?: number
}

export interface AutomationCondition {
  field: 'priority' | 'status_id' | 'assignee_id' | string
  op: 'eq' | 'neq' | 'gte' | 'lte' | 'in' | string
  value?: unknown
}

export type AutomationActionType =
  | 'change_status' | 'assign_user' | 'add_tag' | 'add_comment' | 'notify'

export interface AutomationAction {
  type: AutomationActionType
  status_id?: string
  user_id?: string
  tag_id?: string
  body?: string
  kind?: string
  payload?: Record<string, unknown>
}

export interface Automation {
  id: string
  workspace_id: string
  list_id?: string
  name: string
  description: string
  trigger: AutomationTrigger
  conditions: AutomationCondition[]
  actions: AutomationAction[]
  enabled: boolean
  last_run_at?: string
  run_count: number
  created_at: string
  updated_at: string
}

export interface Doc {
  id: string
  workspace_id: string
  space_id?: string
  parent_id?: string
  creator_id?: string
  title: string
  icon?: string | null
  content: unknown
  content_text?: string
  version: number
  archived: boolean
  order_index: number
  created_at: string
  updated_at: string
}

export interface Channel {
  id: string
  workspace_id: string
  space_id?: string
  name: string
  topic: string
  kind: 'channel' | 'dm' | string
  is_private: boolean
  creator_id?: string
  created_at: string
  updated_at: string
}

export interface ChatMessage {
  id: string
  channel_id: string
  author_id?: string
  parent_message_id?: string
  body: string
  edited_at?: string
  deleted_at?: string
  created_at: string
}

export interface SearchHit {
  entity_type: 'task' | 'doc' | 'comment' | 'message' | string
  entity_id: string
  title: string
  snippet: string
  rank: number
  workspace_id?: string
  list_id?: string
  task_id?: string
  doc_id?: string
  channel_id?: string
  created_at?: string
}

export type TargetKind = 'number' | 'currency' | 'boolean' | 'task_completed'

export interface GoalTarget {
  id: string
  goal_id: string
  name: string
  kind: TargetKind
  target_number?: number
  current_number: number
  currency?: string
  target_boolean?: boolean
  current_boolean: boolean
  task_list_id?: string
  order_index: number
  created_at: string
  updated_at: string
  total_tasks?: number
  completed_tasks?: number
  progress: number
}

export interface Goal {
  id: string
  workspace_id: string
  parent_goal_id?: string
  owner_id?: string
  name: string
  description: string
  start_at?: string
  due_at?: string
  archived: boolean
  created_at: string
  updated_at: string
  progress: number
}

export type SprintStatus = 'planned' | 'active' | 'closed'

export interface Sprint {
  id: string
  list_id: string
  name: string
  starts_at: string
  ends_at: string
  goal_points: number
  status: SprintStatus
  created_at: string
  updated_at: string
}

export interface BurndownPoint {
  day: string
  remaining_points: number
  completed_points: number
  total_points: number
}

export interface CloseSprintResult {
  closed: Sprint
  rolled_over: number
  next_sprint_id?: string
}

export type WidgetKind = 'burndown' | 'velocity' | 'task_count' | 'time_per_user'

export interface Dashboard {
  id: string
  workspace_id: string
  space_id?: string
  name: string
  creator_id?: string
  created_at: string
  updated_at: string
}

export interface Widget {
  id: string
  dashboard_id: string
  kind: WidgetKind
  title: string
  config: Record<string, unknown>
  order_index: number
  created_at: string
  updated_at: string
}

export interface VelocityPoint {
  sprint_id: string
  sprint_name: string
  ends_at: string
  completed_points: number
  goal_points: number
}

export interface Whiteboard {
  id: string
  workspace_id: string
  space_id?: string
  name: string
  snapshot: unknown
  version: number
  creator_id?: string
  created_at: string
  updated_at: string
}

export type FormFieldKind = 'text' | 'textarea' | 'number' | 'email' | 'select' | 'checkbox'

export interface FormFieldDef {
  id: string
  label: string
  kind: FormFieldKind
  required: boolean
  options?: string[]
  placeholder?: string
}

export interface FormDef {
  id: string
  list_id: string
  name: string
  description: string
  fields: FormFieldDef[]
  is_public: boolean
  submit_count: number
  creator_id?: string
  created_at: string
  updated_at: string
}

export interface FormSubmission {
  id: string
  form_id: string
  task_id?: string
  payload: Record<string, unknown>
  submitted_at: string
  submitter_ip?: string
}

export type TemplateKind = 'list' | 'doc' | 'task'

export interface Template {
  id: string
  workspace_id: string
  kind: TemplateKind
  name: string
  description: string
  snapshot: unknown
  creator_id?: string
  created_at: string
  updated_at: string
}

export interface AccomplishmentTask {
  id: string
  name: string
  list_id: string
  list_name: string
  archived: boolean
  completed_at: string
}

export interface AccomplishmentBucket {
  period: string
  starts_at: string
  ends_at: string
  count: number
  tasks: AccomplishmentTask[]
}

export interface Credential {
  id: string
  workspace_id: string
  user_id: string
  name: string
  url: string
  username: string
  // Only present on the detail fetch (GET /workspaces/{wsId}/credentials/{id}).
  // Omitted from list responses and from create/update responses.
  password?: string
  notes: string
  created_at: string
  updated_at: string
}

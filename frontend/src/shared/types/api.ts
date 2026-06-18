export type User = {
  id: string;
  email: string;
  last_name: string;
  first_name: string;
  middle_name: string | null;
  full_name: string;
  created_at: string;
};

export type AttendanceEvent = {
  event_type: "check_in" | "check_out";
  event_at: string;
  status: "normal" | "outside_allowed_time" | "system_outage";
};

export type AttendanceToday = {
  business_date: string;
  check_in: AttendanceEvent | null;
  check_out: AttendanceEvent | null;
  late_minutes: number;
  early_leave_minutes: number;
  impacted_by_outage: boolean;
  can_check_in: boolean;
  can_check_out: boolean;
};

export type AttendanceMarkPayload = {
  device_id: string;
  phone_model: string;
  browser: string;
};

export type AttendanceDayStatus = "empty" | "in_progress" | "complete";
export type AttendanceExplanationStatus = "pending" | "approved" | "rejected";
export type AttendanceExplanationReason =
  | "late"
  | "early_leave"
  | "missing_check_in"
  | "missing_check_out"
  | "missing_day"
  | "void_day_request";

export type AttendanceExplanation = {
  id: string;
  user_id: string;
  business_date: string;
  reason_type: AttendanceExplanationReason;
  comment: string;
  status: AttendanceExplanationStatus;
  reviewed_by_admin_email: string | null;
  reviewed_at: string | null;
  review_note: string | null;
  created_at: string;
  updated_at: string;
};

export type AttendanceDaySummary = {
  date: string;
  check_in_at: string | null;
  check_out_at: string | null;
  worked_minutes: number;
  late_minutes: number;
  early_leave_minutes: number;
  status: AttendanceDayStatus;
  impacted_by_outage: boolean;
  voided: boolean;
  void_reason: string | null;
  voided_by_admin: string | null;
  voided_at: string | null;
  explanations: AttendanceExplanation[];
};

export type AttendanceSummary = {
  from: string;
  to: string;
  workday_start: string;
  workday_end: string;
  target_minutes_per_day: number;
  days: AttendanceDaySummary[];
};

export type AdminMe = {
  is_admin: boolean;
  email: string;
  full_name: string;
};

export type AdminReportRun = {
  id: string;
  period_start: string;
  period_end: string;
  sent_at: string;
};

export type AdminReportList = {
  items: AdminReportRun[];
};

export type AdminAccess = {
  email: string;
  full_name: string | null;
  created_at: string;
  created_by: string | null;
  revoked_at: string | null;
  is_active: boolean;
  has_session: boolean;
};

export type AdminAccessList = {
  items: AdminAccess[];
};

export type AdminSession = {
  id: string;
  email: string;
  full_name: string;
  created_at: string;
  expires_at: string;
  revoked_at: string | null;
  is_active: boolean;
};

export type AdminSessionList = {
  items: AdminSession[];
};

export type AdminEmployeeSummary = {
  user_id: string;
  email: string;
  full_name: string;
  worked_minutes: number;
  target_minutes: number;
  check_in_count: number;
  check_out_count: number;
  late_count: number;
  early_leave_count: number;
  missing_check_outs: number;
  worked_days: number;
};

export type AdminEmployeesMonth = {
  from: string;
  to: string;
  workday_start: string;
  workday_end: string;
  target_minutes_per_day: number;
  employees: AdminEmployeeSummary[];
};

export type AdminEmployeeMonthDetail = AdminEmployeeSummary & {
  days: AttendanceDaySummary[];
};

export type AdminAttendanceEvent = {
  event_id: string;
  user_id: string;
  email: string;
  full_name: string;
  business_date: string;
  event_type: "check_in" | "check_out";
  event_at: string;
  device_id: string;
  external_ip: string;
};

export type AdminSuspiciousActor = {
  user_id: string;
  email: string;
  full_name: string;
};

export type AdminSuspiciousDeviceMatch = {
  device_id: string;
  owner: AdminSuspiciousActor;
  event: AdminAttendanceEvent;
};

export type AdminSuspiciousIPMatch = {
  external_ip: string;
  event: AdminAttendanceEvent;
  previous_event: AdminAttendanceEvent;
  minutes_between: number;
};

export type AdminSuspiciousActivity = {
  from: string;
  to: string;
  device_matches: AdminSuspiciousDeviceMatch[];
  ip_matches: AdminSuspiciousIPMatch[];
};

export type AdminSystemOutage = {
  id: string;
  started_at: string;
  ended_at: string;
  reason: string | null;
  created_at: string;
  affected_business_date: string | null;
  impacts_work_hours: boolean;
  resolved_at: string | null;
  resolved_by: string | null;
  resolution_note: string | null;
};

export type AdminSystemOutageList = {
  items: AdminSystemOutage[];
};

export type AdminOutageDayEmployee = {
  user_id: string;
  email: string;
  full_name: string;
  check_in_at: string | null;
  check_out_at: string | null;
};

export type AdminOutageDay = {
  outage: AdminSystemOutage;
  employees: AdminOutageDayEmployee[];
};

export type AdminOutageRepairItem = {
  user_id: string;
  check_in_at?: string;
  check_out_at?: string;
};

export type AdminExplanation = AttendanceExplanation & {
  email: string;
  full_name: string;
  check_in_at: string | null;
  check_out_at: string | null;
};

export type AdminExplanationList = {
  items: AdminExplanation[];
};

export type AdminExplanationDecision = {
  review_note?: string;
  check_in_at?: string;
  check_out_at?: string;
};

export type AdminAuditAction =
  | "day_voided"
  | "day_restored"
  | "check_in_changed"
  | "check_out_changed"
  | "explanation_approved"
  | "explanation_rejected"
  | "explanation_rollback"
  | "system_outage_resolved";

export type AdminAuditDecisionSource = "admin_decision" | "employee_request";

export type AdminAuditLog = {
  id: string;
  admin_email: string;
  user_id: string | null;
  explanation_id: string | null;
  email: string | null;
  full_name: string | null;
  business_date: string | null;
  action: AdminAuditAction;
  old_check_in_at: string | null;
  old_check_out_at: string | null;
  new_check_in_at: string | null;
  new_check_out_at: string | null;
  decision_source: AdminAuditDecisionSource;
  reason: string;
  created_at: string;
};

export type AdminAuditLogList = {
  items: AdminAuditLog[];
};

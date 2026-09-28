import React from "react";
import { Building2, User } from "lucide-react";

// Marks a row as a brand (company) or a person at a glance — wherever a
// subject name appears in a table or feed, across every tab.
export default function SubjectTypeIcon({ type }) {
  const Icon = type === "person" ? User : Building2;
  return <Icon size={12} style={{ color: "var(--muted)" }} aria-label={type} />;
}

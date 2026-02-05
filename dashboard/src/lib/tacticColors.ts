/**
 * Color mapping for MITRE ATT&CK tactics.
 * Shared across components for consistent tactic styling.
 */

/**
 * Badge colors for tactic labels (used in tables and lists).
 */
export const tacticBadgeColors: Record<string, string> = {
  reconnaissance: 'bg-purple-100 text-purple-700',
  resource_development: 'bg-blue-100 text-blue-700',
  initial_access: 'bg-red-100 text-red-700',
  execution: 'bg-orange-100 text-orange-700',
  persistence: 'bg-yellow-100 text-yellow-700',
  privilege_escalation: 'bg-pink-100 text-pink-700',
  defense_evasion: 'bg-green-100 text-green-700',
  credential_access: 'bg-indigo-100 text-indigo-700',
  discovery: 'bg-cyan-100 text-cyan-700',
  lateral_movement: 'bg-teal-100 text-teal-700',
  collection: 'bg-lime-100 text-lime-700',
  command_and_control: 'bg-amber-100 text-amber-700',
  exfiltration: 'bg-rose-100 text-rose-700',
  impact: 'bg-red-100 text-red-700',
};

/**
 * Get the badge color class for a tactic.
 * Normalizes the tactic string (replaces hyphens with underscores).
 */
export function getTacticBadgeColor(tactic: string): string {
  const normalizedTactic = String(tactic).replaceAll('-', '_');
  return tacticBadgeColors[normalizedTactic] || 'bg-gray-100 text-gray-700';
}

/**
 * Format tactic name for display.
 * Replaces hyphens and underscores with spaces.
 */
export function formatTacticName(tactic: string): string {
  return String(tactic).replaceAll('-', '_').replaceAll('_', ' ');
}

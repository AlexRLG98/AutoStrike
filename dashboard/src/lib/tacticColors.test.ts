import { describe, it, expect } from 'vitest';
import { tacticBadgeColors, getTacticBadgeColor, formatTacticName } from './tacticColors';

describe('tacticColors', () => {
  describe('tacticBadgeColors', () => {
    it('contains all 14 MITRE ATT&CK tactics', () => {
      const tactics = [
        'reconnaissance',
        'resource_development',
        'initial_access',
        'execution',
        'persistence',
        'privilege_escalation',
        'defense_evasion',
        'credential_access',
        'discovery',
        'lateral_movement',
        'collection',
        'command_and_control',
        'exfiltration',
        'impact',
      ];

      tactics.forEach((tactic) => {
        expect(tacticBadgeColors[tactic]).toBeDefined();
        expect(tacticBadgeColors[tactic]).toContain('bg-');
        expect(tacticBadgeColors[tactic]).toContain('text-');
      });
    });

    it('has correct color for reconnaissance', () => {
      expect(tacticBadgeColors.reconnaissance).toBe('bg-purple-100 text-purple-700');
    });

    it('has correct color for discovery', () => {
      expect(tacticBadgeColors.discovery).toBe('bg-cyan-100 text-cyan-700');
    });

    it('has correct color for execution', () => {
      expect(tacticBadgeColors.execution).toBe('bg-orange-100 text-orange-700');
    });
  });

  describe('getTacticBadgeColor', () => {
    it('returns correct color for known tactic', () => {
      expect(getTacticBadgeColor('discovery')).toBe('bg-cyan-100 text-cyan-700');
    });

    it('returns correct color for tactic with underscores', () => {
      expect(getTacticBadgeColor('lateral_movement')).toBe('bg-teal-100 text-teal-700');
    });

    it('normalizes hyphens to underscores', () => {
      expect(getTacticBadgeColor('lateral-movement')).toBe('bg-teal-100 text-teal-700');
    });

    it('returns default gray for unknown tactic', () => {
      expect(getTacticBadgeColor('unknown_tactic')).toBe('bg-gray-100 text-gray-700');
    });

    it('returns default gray for empty string', () => {
      expect(getTacticBadgeColor('')).toBe('bg-gray-100 text-gray-700');
    });

    it('handles mixed case input by not matching', () => {
      // tacticBadgeColors keys are lowercase, so mixed case won't match
      expect(getTacticBadgeColor('Discovery')).toBe('bg-gray-100 text-gray-700');
    });
  });

  describe('formatTacticName', () => {
    it('replaces underscores with spaces', () => {
      expect(formatTacticName('lateral_movement')).toBe('lateral movement');
    });

    it('replaces hyphens with spaces', () => {
      expect(formatTacticName('lateral-movement')).toBe('lateral movement');
    });

    it('handles mixed hyphens and underscores', () => {
      expect(formatTacticName('command-and_control')).toBe('command and control');
    });

    it('returns same string if no separators', () => {
      expect(formatTacticName('discovery')).toBe('discovery');
    });

    it('handles empty string', () => {
      expect(formatTacticName('')).toBe('');
    });

    it('handles multiple consecutive separators', () => {
      expect(formatTacticName('test__name')).toBe('test  name');
    });
  });
});

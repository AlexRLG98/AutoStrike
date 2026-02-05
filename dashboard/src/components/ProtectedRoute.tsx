import { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { UserRole } from '../lib/api';

interface ProtectedRouteProps {
  readonly children: ReactNode;
  readonly requiredRole?: UserRole;
  readonly allowedRoles?: UserRole[];
}

/**
 * Role hierarchy levels (higher number = more privileges)
 * admin (5) > rssi (4) > operator (3) > analyst (2) > viewer (1)
 */
const roleHierarchy: Record<UserRole, number> = {
  admin: 5,
  rssi: 4,
  operator: 3,
  analyst: 2,
  viewer: 1,
};

/**
 * Checks if a user has the required role to access a resource.
 *
 * This function supports TWO MODES of authorization:
 *
 * 1. **Hierarchy mode** (using `requiredRole`):
 *    - Uses role hierarchy where higher roles can access lower role resources
 *    - Example: requiredRole="operator" allows admin, rssi, and operator
 *    - Use this for "minimum required privilege" checks
 *
 * 2. **Exact match mode** (using `allowedRoles`):
 *    - Only the explicitly listed roles are allowed (no hierarchy)
 *    - Example: allowedRoles=["operator"] allows ONLY operator, not admin
 *    - Use this for role-specific features (e.g., only analysts can export)
 *
 * If both are provided, `allowedRoles` takes precedence (exact match mode).
 * If neither is provided, any authenticated user is allowed.
 *
 * @param userRole - The current user's role
 * @param requiredRole - Minimum role required (hierarchy mode)
 * @param allowedRoles - Exact list of allowed roles (exact match mode)
 * @returns true if the user has access, false otherwise
 */
function hasRequiredRole(userRole: UserRole | undefined, requiredRole?: UserRole, allowedRoles?: UserRole[]): boolean {
  if (!userRole) return false;

  // Exact match mode: check if user's role is in the allowed list
  // Takes precedence over hierarchy mode when both are specified
  if (allowedRoles && allowedRoles.length > 0) {
    return allowedRoles.includes(userRole);
  }

  // Hierarchy mode: check if user's role level meets the minimum required
  if (requiredRole) {
    return roleHierarchy[userRole] >= roleHierarchy[requiredRole];
  }

  // No role requirement - any authenticated user is allowed
  return true;
}

export function ProtectedRoute({ children, requiredRole, allowedRoles }: ProtectedRouteProps) {
  const { isAuthenticated, isLoading, user, authEnabled } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    // Redirect to login, preserving the intended destination
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // Skip role check if auth is disabled (development mode - full access)
  if (!authEnabled) {
    return <>{children}</>;
  }

  // Check role-based access
  if ((requiredRole || allowedRoles) && !hasRequiredRole(user?.role as UserRole, requiredRole, allowedRoles)) {
    // User doesn't have required role - show unauthorized
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-900">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-red-500">403</h1>
          <p className="mt-4 text-gray-400">Access Denied</p>
          <p className="mt-2 text-sm text-gray-500">You don't have permission to access this page.</p>
          <a href="/dashboard" className="mt-4 inline-block text-primary-400 hover:text-primary-300">
            Return to Dashboard
          </a>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}

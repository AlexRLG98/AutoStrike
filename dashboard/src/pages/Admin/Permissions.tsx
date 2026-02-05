import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { ShieldCheckIcon, CheckIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { permissionApi, UserRole } from '../../lib/api';
import { LoadingState } from '../../components/LoadingState';
import { useAuth } from '../../contexts/AuthContext';

const ROLE_COLORS: Record<UserRole, string> = {
  admin: 'bg-red-100 text-red-800',
  rssi: 'bg-purple-100 text-purple-800',
  operator: 'bg-blue-100 text-blue-800',
  analyst: 'bg-green-100 text-green-800',
  viewer: 'bg-gray-100 text-gray-800',
};

const ROLE_DISPLAY_NAMES: Record<UserRole, string> = {
  admin: 'Admin',
  rssi: 'RSSI',
  operator: 'Operator',
  analyst: 'Analyst',
  viewer: 'Viewer',
};

export default function Permissions() {
  const { user } = useAuth();

  const { data: matrix, isLoading } = useQuery({
    queryKey: ['permissions', 'matrix'],
    queryFn: () => permissionApi.getMatrix().then((res) => res.data),
  });

  const { data: myPermissions } = useQuery({
    queryKey: ['permissions', 'me'],
    queryFn: () => permissionApi.getMyPermissions().then((res) => res.data),
  });

  if (isLoading || !matrix) {
    return <LoadingState message="Loading permissions..." />;
  }

  const roles = matrix.roles;
  const permissionsByCategory = matrix.categories.map((category) => ({
    ...category,
    permissionDetails: matrix.permissions.filter((p) => p.category === category.name),
  }));

  const hasPermission = (role: UserRole, permission: string): boolean => {
    const rolePerms = matrix.matrix[role];
    return rolePerms?.includes(permission) ?? false;
  };

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-3xl font-bold">Permissions Matrix</h1>
        <p className="text-gray-600 mt-1">
          Overview of role-based permissions across the system
        </p>
      </div>

      {/* Current User Permissions */}
      {myPermissions && (
        <div className="card mb-8">
          <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
            <ShieldCheckIcon className="h-5 w-5 text-primary-600" />
            Your Permissions
          </h2>
          <div className="flex items-center gap-4 mb-4">
            <span className="text-gray-600">Role:</span>
            <span
              className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${ROLE_COLORS[(myPermissions.role as UserRole) || user?.role || 'viewer']}`}
            >
              {ROLE_DISPLAY_NAMES[(myPermissions.role as UserRole) || user?.role || 'viewer']}
            </span>
          </div>
          <div className="flex flex-wrap gap-2">
            {myPermissions.permissions.map((perm) => (
              <span
                key={perm}
                className="inline-flex items-center px-2 py-1 rounded bg-gray-100 text-gray-700 text-xs"
              >
                {perm}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Permission Matrix Table */}
      <div className="card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider sticky left-0 bg-gray-50 z-10 min-w-[200px]">
                  Permission
                </th>
                {roles.map((role) => (
                  <th
                    key={role}
                    className="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider min-w-[100px]"
                  >
                    <span
                      className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${ROLE_COLORS[role]}`}
                    >
                      {ROLE_DISPLAY_NAMES[role]}
                    </span>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {permissionsByCategory.map((category) => (
                <React.Fragment key={`category-${category.name}`}>
                  {/* Category Header */}
                  <tr className="bg-gray-100">
                    <td
                      colSpan={roles.length + 1}
                      className="px-4 py-2 text-sm font-semibold text-gray-700 sticky left-0 bg-gray-100"
                    >
                      {category.name}
                      <span className="text-gray-500 font-normal ml-2">
                        — {category.description}
                      </span>
                    </td>
                  </tr>
                  {/* Permission Rows */}
                  {category.permissionDetails.map((perm) => (
                    <tr key={perm.permission} className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-sm sticky left-0 bg-white z-10">
                        <div>
                          <span className="font-medium text-gray-900">{perm.name}</span>
                          <p className="text-xs text-gray-500">{perm.description}</p>
                        </div>
                      </td>
                      {roles.map((role) => (
                        <td key={`${perm.permission}-${role}`} className="px-4 py-3 text-center">
                          {hasPermission(role, perm.permission) ? (
                            <CheckIcon className="h-5 w-5 text-green-500 mx-auto" />
                          ) : (
                            <XMarkIcon className="h-5 w-5 text-gray-300 mx-auto" />
                          )}
                        </td>
                      ))}
                    </tr>
                  ))}
                </React.Fragment>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Role Descriptions */}
      <div className="mt-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {roles.map((role) => {
          const permCount = matrix.matrix[role]?.length ?? 0;
          return (
            <div key={role} className="card">
              <div className="flex items-center gap-3 mb-2">
                <span
                  className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${ROLE_COLORS[role]}`}
                >
                  {ROLE_DISPLAY_NAMES[role]}
                </span>
              </div>
              <p className="text-sm text-gray-600">
                {getRoleDescription(role)}
              </p>
              <p className="text-xs text-gray-400 mt-2">
                {permCount} permission{permCount === 1 ? '' : 's'}
              </p>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function getRoleDescription(role: UserRole): string {
  switch (role) {
    case 'admin':
      return 'Full system access. Can manage users, configure settings, and perform all operations.';
    case 'rssi':
      return 'Security Officer. Can view all data, access analytics and reports, but cannot execute scenarios.';
    case 'operator':
      return 'Can manage agents, scenarios, and execute attack simulations. No user management access.';
    case 'analyst':
      return 'Read-only access with analytics capabilities. Can view and export reports.';
    case 'viewer':
      return 'Basic read-only access. Can view agents, techniques, scenarios, and execution results.';
    default:
      return 'Unknown role';
  }
}

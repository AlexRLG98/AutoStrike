import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  UserIcon,
  PlusIcon,
  PencilIcon,
  TrashIcon,
  ArrowPathIcon,
  KeyIcon,
  CheckIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline';
import { formatDistanceToNow } from 'date-fns';
import {
  adminApi,
  User,
  UserRole,
  CreateUserRequest,
  UpdateUserRequest,
  UpdateRoleRequest,
  ResetPasswordRequest,
} from '../../lib/api';
import { LoadingState } from '../../components/LoadingState';
import { EmptyState } from '../../components/EmptyState';
import { Modal } from '../../components/Modal';
import { useAuth } from '../../contexts/AuthContext';

const ROLES: { value: UserRole; label: string; description: string }[] = [
  { value: 'admin', label: 'Administrator', description: 'Full system access' },
  { value: 'rssi', label: 'Security Officer (RSSI)', description: 'View reports and analytics' },
  { value: 'operator', label: 'Operator', description: 'Execute scenarios' },
  { value: 'analyst', label: 'Analyst', description: 'Read-only with analytics' },
  { value: 'viewer', label: 'Viewer', description: 'Read-only basic access' },
];

type ModalType = 'create' | 'edit' | 'role' | 'password' | 'deactivate' | 'reactivate' | null;

export default function Users() {
  const queryClient = useQueryClient();
  const { user: currentUser } = useAuth();
  const [includeInactive, setIncludeInactive] = useState(false);
  const [modalType, setModalType] = useState<ModalType>(null);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [formError, setFormError] = useState<string | null>(null);

  // Form states
  const [createForm, setCreateForm] = useState<CreateUserRequest>({
    username: '',
    email: '',
    password: '',
    role: 'viewer',
  });
  const [editForm, setEditForm] = useState<UpdateUserRequest>({});
  const [roleForm, setRoleForm] = useState<UpdateRoleRequest>({ role: 'viewer' });
  const [passwordForm, setPasswordForm] = useState<ResetPasswordRequest>({ new_password: '' });

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'users', includeInactive],
    queryFn: () => adminApi.listUsers(includeInactive).then((res) => res.data),
  });

  const createMutation = useMutation({
    mutationFn: (data: CreateUserRequest) => adminApi.createUser(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      closeModal();
    },
    onError: (error: Error & { response?: { data?: { error?: string } } }) => {
      setFormError(error.response?.data?.error || 'Failed to create user');
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateUserRequest }) =>
      adminApi.updateUser(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      closeModal();
    },
    onError: (error: Error & { response?: { data?: { error?: string } } }) => {
      setFormError(error.response?.data?.error || 'Failed to update user');
    },
  });

  const roleMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateRoleRequest }) =>
      adminApi.updateUserRole(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      closeModal();
    },
    onError: (error: Error & { response?: { data?: { error?: string } } }) => {
      setFormError(error.response?.data?.error || 'Failed to update role');
    },
  });

  const passwordMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: ResetPasswordRequest }) =>
      adminApi.resetPassword(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      closeModal();
    },
    onError: (error: Error & { response?: { data?: { error?: string } } }) => {
      setFormError(error.response?.data?.error || 'Failed to reset password');
    },
  });

  const deactivateMutation = useMutation({
    mutationFn: (id: string) => adminApi.deactivateUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      closeModal();
    },
    onError: (error: Error & { response?: { data?: { error?: string } } }) => {
      setFormError(error.response?.data?.error || 'Failed to deactivate user');
    },
  });

  const reactivateMutation = useMutation({
    mutationFn: (id: string) => adminApi.reactivateUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
      closeModal();
    },
    onError: (error: Error & { response?: { data?: { error?: string } } }) => {
      setFormError(error.response?.data?.error || 'Failed to reactivate user');
    },
  });

  const closeModal = () => {
    setModalType(null);
    setSelectedUser(null);
    setFormError(null);
    setCreateForm({ username: '', email: '', password: '', role: 'viewer' });
    setEditForm({});
    setRoleForm({ role: 'viewer' });
    setPasswordForm({ new_password: '' });
  };

  const openCreateModal = () => {
    setModalType('create');
  };

  const openEditModal = (user: User) => {
    setSelectedUser(user);
    setEditForm({ username: user.username, email: user.email });
    setModalType('edit');
  };

  const openRoleModal = (user: User) => {
    setSelectedUser(user);
    setRoleForm({ role: user.role });
    setModalType('role');
  };

  const openPasswordModal = (user: User) => {
    setSelectedUser(user);
    setPasswordForm({ new_password: '' });
    setModalType('password');
  };

  const openDeactivateModal = (user: User) => {
    setSelectedUser(user);
    setModalType('deactivate');
  };

  const openReactivateModal = (user: User) => {
    setSelectedUser(user);
    setModalType('reactivate');
  };

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    createMutation.mutate(createForm);
  };

  const handleEdit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedUser) return;
    setFormError(null);
    updateMutation.mutate({ id: selectedUser.id, data: editForm });
  };

  const handleRoleChange = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedUser) return;
    setFormError(null);
    roleMutation.mutate({ id: selectedUser.id, data: roleForm });
  };

  const handlePasswordReset = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedUser) return;
    setFormError(null);
    passwordMutation.mutate({ id: selectedUser.id, data: passwordForm });
  };

  const handleDeactivate = () => {
    if (!selectedUser) return;
    setFormError(null);
    deactivateMutation.mutate(selectedUser.id);
  };

  const handleReactivate = () => {
    if (!selectedUser) return;
    setFormError(null);
    reactivateMutation.mutate(selectedUser.id);
  };

  const getRoleBadgeClass = (role: UserRole) => {
    switch (role) {
      case 'admin':
        return 'bg-red-100 text-red-800';
      case 'rssi':
        return 'bg-purple-100 text-purple-800';
      case 'operator':
        return 'bg-blue-100 text-blue-800';
      case 'analyst':
        return 'bg-green-100 text-green-800';
      case 'viewer':
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const getRoleLabel = (role: UserRole) => {
    return ROLES.find((r) => r.value === role)?.label || role;
  };

  if (isLoading) {
    return <LoadingState message="Loading users..." />;
  }

  const users = data?.users || [];

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-3xl font-bold">User Management</h1>
          <p className="text-gray-600 mt-1">Manage users and their permissions</p>
        </div>
        <button className="btn-primary flex items-center gap-2" onClick={openCreateModal}>
          <PlusIcon className="h-5 w-5" />
          Add User
        </button>
      </div>

      {/* Filters */}
      <div className="mb-6">
        <label className="flex items-center gap-2 text-sm text-gray-600">
          <input
            type="checkbox"
            checked={includeInactive}
            onChange={(e) => setIncludeInactive(e.target.checked)}
            className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />{' '}
          <span>Show inactive users</span>
        </label>
      </div>

      {/* Users Table */}
      {users.length > 0 ? (
        <div className="card overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  User
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Role
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Last Login
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {users.map((user) => (
                <tr key={user.id} className={user.is_active ? '' : 'bg-gray-50'}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      <div className="h-10 w-10 flex-shrink-0">
                        <div className="h-10 w-10 rounded-full bg-primary-100 flex items-center justify-center">
                          <span className="text-primary-700 font-medium">
                            {user.username.charAt(0).toUpperCase()}
                          </span>
                        </div>
                      </div>
                      <div className="ml-4">
                        <div className="text-sm font-medium text-gray-900">
                          {user.username}
                          {user.id === currentUser?.id && (
                            <span className="ml-2 text-xs text-gray-500">(you)</span>
                          )}
                        </div>
                        <div className="text-sm text-gray-500">{user.email}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getRoleBadgeClass(user.role)}`}
                    >
                      {getRoleLabel(user.role)}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        user.is_active
                          ? 'bg-green-100 text-green-800'
                          : 'bg-gray-100 text-gray-800'
                      }`}
                    >
                      {user.is_active ? 'Active' : 'Inactive'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {user.last_login_at
                      ? formatDistanceToNow(new Date(user.last_login_at), { addSuffix: true })
                      : 'Never'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => openEditModal(user)}
                        className="p-1 text-gray-400 hover:text-gray-600"
                        title="Edit user"
                      >
                        <PencilIcon className="h-5 w-5" />
                      </button>
                      <button
                        onClick={() => openRoleModal(user)}
                        className="p-1 text-gray-400 hover:text-blue-600"
                        title="Change role"
                        disabled={user.id === currentUser?.id}
                      >
                        <UserIcon className="h-5 w-5" />
                      </button>
                      <button
                        onClick={() => openPasswordModal(user)}
                        className="p-1 text-gray-400 hover:text-yellow-600"
                        title="Reset password"
                      >
                        <KeyIcon className="h-5 w-5" />
                      </button>
                      {user.is_active ? (
                        <button
                          onClick={() => openDeactivateModal(user)}
                          className="p-1 text-gray-400 hover:text-red-600"
                          title="Deactivate user"
                          disabled={user.id === currentUser?.id}
                        >
                          <TrashIcon className="h-5 w-5" />
                        </button>
                      ) : (
                        <button
                          onClick={() => openReactivateModal(user)}
                          className="p-1 text-gray-400 hover:text-green-600"
                          title="Reactivate user"
                        >
                          <ArrowPathIcon className="h-5 w-5" />
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyState
          icon={UserIcon}
          title="No users found"
          description="Create a new user to get started"
        />
      )}

      {/* Create User Modal */}
      {modalType === 'create' && (
        <Modal title="Create User" onClose={closeModal}>
          <form onSubmit={handleCreate} className="space-y-4">
            {formError && <ErrorAlert message={formError} />}
            <div>
              <label htmlFor="create-username" className="block text-sm font-medium text-gray-700">Username</label>
              <input
                id="create-username"
                type="text"
                required
                minLength={3}
                value={createForm.username}
                onChange={(e) => setCreateForm({ ...createForm, username: e.target.value })}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
              />
            </div>
            <div>
              <label htmlFor="create-email" className="block text-sm font-medium text-gray-700">Email</label>
              <input
                id="create-email"
                type="email"
                required
                value={createForm.email}
                onChange={(e) => setCreateForm({ ...createForm, email: e.target.value })}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
              />
            </div>
            <div>
              <label htmlFor="create-password" className="block text-sm font-medium text-gray-700">Password</label>
              <input
                id="create-password"
                type="password"
                required
                minLength={8}
                value={createForm.password}
                onChange={(e) => setCreateForm({ ...createForm, password: e.target.value })}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
              />
              <p className="mt-1 text-xs text-gray-500">Minimum 8 characters</p>
            </div>
            <div>
              <label htmlFor="create-role" className="block text-sm font-medium text-gray-700">Role</label>
              <select
                id="create-role"
                value={createForm.role}
                onChange={(e) =>
                  setCreateForm({ ...createForm, role: e.target.value as UserRole })
                }
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
              >
                {ROLES.map((role) => (
                  <option key={role.value} value={role.value}>
                    {role.label} - {role.description}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex justify-end gap-3 pt-4">
              <button type="button" onClick={closeModal} className="btn-secondary">
                Cancel
              </button>
              <button type="submit" className="btn-primary" disabled={createMutation.isPending}>
                {createMutation.isPending ? 'Creating...' : 'Create User'}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* Edit User Modal */}
      {modalType === 'edit' && selectedUser && (
        <Modal title="Edit User" onClose={closeModal}>
          <form onSubmit={handleEdit} className="space-y-4">
            {formError && <ErrorAlert message={formError} />}
            <div>
              <label htmlFor="edit-username" className="block text-sm font-medium text-gray-700">Username</label>
              <input
                id="edit-username"
                type="text"
                required
                minLength={3}
                value={editForm.username || ''}
                onChange={(e) => setEditForm({ ...editForm, username: e.target.value })}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
              />
            </div>
            <div>
              <label htmlFor="edit-email" className="block text-sm font-medium text-gray-700">Email</label>
              <input
                id="edit-email"
                type="email"
                required
                value={editForm.email || ''}
                onChange={(e) => setEditForm({ ...editForm, email: e.target.value })}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
              />
            </div>
            <div className="flex justify-end gap-3 pt-4">
              <button type="button" onClick={closeModal} className="btn-secondary">
                Cancel
              </button>
              <button type="submit" className="btn-primary" disabled={updateMutation.isPending}>
                {updateMutation.isPending ? 'Saving...' : 'Save Changes'}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* Change Role Modal */}
      {modalType === 'role' && selectedUser && (
        <Modal title="Change Role" onClose={closeModal}>
          <form onSubmit={handleRoleChange} className="space-y-4">
            {formError && <ErrorAlert message={formError} />}
            <p className="text-sm text-gray-600">
              Change role for <strong>{selectedUser.username}</strong>
            </p>
            <div>
              <label htmlFor="change-role" className="block text-sm font-medium text-gray-700">Role</label>
              <select
                id="change-role"
                value={roleForm.role}
                onChange={(e) => setRoleForm({ role: e.target.value as UserRole })}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
              >
                {ROLES.map((role) => (
                  <option key={role.value} value={role.value}>
                    {role.label} - {role.description}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex justify-end gap-3 pt-4">
              <button type="button" onClick={closeModal} className="btn-secondary">
                Cancel
              </button>
              <button type="submit" className="btn-primary" disabled={roleMutation.isPending}>
                {roleMutation.isPending ? 'Updating...' : 'Update Role'}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* Reset Password Modal */}
      {modalType === 'password' && selectedUser && (
        <Modal title="Reset Password" onClose={closeModal}>
          <form onSubmit={handlePasswordReset} className="space-y-4">
            {formError && <ErrorAlert message={formError} />}
            <p className="text-sm text-gray-600">
              Reset password for <strong>{selectedUser.username}</strong>
            </p>
            <div>
              <label htmlFor="reset-password" className="block text-sm font-medium text-gray-700">New Password</label>
              <input
                id="reset-password"
                type="password"
                required
                minLength={8}
                value={passwordForm.new_password}
                onChange={(e) => setPasswordForm({ new_password: e.target.value })}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
              />
              <p className="mt-1 text-xs text-gray-500">Minimum 8 characters</p>
            </div>
            <div className="flex justify-end gap-3 pt-4">
              <button type="button" onClick={closeModal} className="btn-secondary">
                Cancel
              </button>
              <button type="submit" className="btn-primary" disabled={passwordMutation.isPending}>
                {passwordMutation.isPending ? 'Resetting...' : 'Reset Password'}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* Deactivate User Modal */}
      {modalType === 'deactivate' && selectedUser && (
        <Modal title="Deactivate User" onClose={closeModal}>
          <div className="space-y-4">
            {formError && <ErrorAlert message={formError} />}
            <div className="flex items-start gap-3">
              <div className="flex-shrink-0">
                <ExclamationTriangleIcon className="h-6 w-6 text-yellow-500" />
              </div>
              <div>
                <p className="text-sm text-gray-600">
                  Are you sure you want to deactivate <strong>{selectedUser.username}</strong>?
                </p>
                <p className="text-sm text-gray-500 mt-2">
                  The user will no longer be able to log in, but their data will be preserved.
                </p>
              </div>
            </div>
            <div className="flex justify-end gap-3 pt-4">
              <button type="button" onClick={closeModal} className="btn-secondary">
                Cancel
              </button>
              <button
                type="button"
                onClick={handleDeactivate}
                className="btn-danger"
                disabled={deactivateMutation.isPending}
              >
                {deactivateMutation.isPending ? 'Deactivating...' : 'Deactivate'}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Reactivate User Modal */}
      {modalType === 'reactivate' && selectedUser && (
        <Modal title="Reactivate User" onClose={closeModal}>
          <div className="space-y-4">
            {formError && <ErrorAlert message={formError} />}
            <div className="flex items-start gap-3">
              <div className="flex-shrink-0">
                <CheckIcon className="h-6 w-6 text-green-500" />
              </div>
              <div>
                <p className="text-sm text-gray-600">
                  Are you sure you want to reactivate <strong>{selectedUser.username}</strong>?
                </p>
                <p className="text-sm text-gray-500 mt-2">
                  The user will be able to log in again with their existing credentials.
                </p>
              </div>
            </div>
            <div className="flex justify-end gap-3 pt-4">
              <button type="button" onClick={closeModal} className="btn-secondary">
                Cancel
              </button>
              <button
                type="button"
                onClick={handleReactivate}
                className="btn-primary"
                disabled={reactivateMutation.isPending}
              >
                {reactivateMutation.isPending ? 'Reactivating...' : 'Reactivate'}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}


// Error Alert Component
function ErrorAlert({ message }: { readonly message: string }) {
  return (
    <div className="bg-red-50 border border-red-200 rounded-md p-3">
      <div className="flex items-center gap-2">
        <ExclamationTriangleIcon className="h-5 w-5 text-red-500" />
        <p className="text-sm text-red-700">{message}</p>
      </div>
    </div>
  );
}

import { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
  HomeIcon,
  ComputerDesktopIcon,
  ShieldExclamationIcon,
  Squares2X2Icon,
  DocumentTextIcon,
  PlayIcon,
  Cog6ToothIcon,
  ArrowLeftStartOnRectangleIcon,
} from '@heroicons/react/24/outline';
import clsx from 'clsx';
import { useAuth } from '../contexts/AuthContext';

interface LayoutProps {
  readonly children: ReactNode;
}

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: HomeIcon },
  { name: 'Agents', href: '/agents', icon: ComputerDesktopIcon },
  { name: 'Techniques', href: '/techniques', icon: ShieldExclamationIcon },
  { name: 'ATT&CK Matrix', href: '/matrix', icon: Squares2X2Icon },
  { name: 'Scenarios', href: '/scenarios', icon: DocumentTextIcon },
  { name: 'Executions', href: '/executions', icon: PlayIcon },
  { name: 'Settings', href: '/settings', icon: Cog6ToothIcon },
];

export default function Layout({ children }: LayoutProps) {
  const location = useLocation();
  const { user, logout, authEnabled } = useAuth();

  const handleLogout = async () => {
    await logout();
  };

  return (
    <div className="min-h-screen flex">
      {/* Sidebar */}
      <div className="w-64 bg-gray-900 text-white flex flex-col">
        <div className="p-6">
          <h1 className="text-2xl font-bold text-primary-400">AutoStrike</h1>
          <p className="text-sm text-gray-400 mt-1">BAS Platform</p>
        </div>

        <nav className="flex-1 px-4 space-y-1">
          {navigation.map((item) => {
            const isActive = location.pathname === item.href;
            return (
              <Link
                key={item.name}
                to={item.href}
                className={clsx(
                  'flex items-center gap-3 px-4 py-3 rounded-lg transition-colors',
                  isActive
                    ? 'bg-primary-600 text-white'
                    : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                )}
              >
                <item.icon className="h-5 w-5" />
                {item.name}
              </Link>
            );
          })}
        </nav>

        <div className="p-4 border-t border-gray-800">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-full bg-primary-600 flex items-center justify-center">
                <span className="text-sm font-medium">
                  {(user?.username || 'Admin').charAt(0).toUpperCase()}
                </span>
              </div>
              <div>
                <p className="text-sm font-medium">
                  {user?.username || 'Admin'}
                </p>
                <p className="text-xs text-gray-400">
                  {user?.email || 'admin@autostrike.local'}
                </p>
              </div>
            </div>
            {authEnabled && (
              <button
                onClick={handleLogout}
                className="p-2 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
                title="Logout"
              >
                <ArrowLeftStartOnRectangleIcon className="h-5 w-5" />
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 overflow-auto">
        <main className="p-8">{children}</main>
      </div>
    </div>
  );
}

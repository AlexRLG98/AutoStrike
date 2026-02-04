import { Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import { ErrorBoundary } from './components/ErrorBoundary';
import { ProtectedRoute } from './components/ProtectedRoute';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Agents from './pages/Agents';
import Techniques from './pages/Techniques';
import Scenarios from './pages/Scenarios';
import Executions from './pages/Executions';
import ExecutionDetails from './pages/ExecutionDetails';
import Settings from './pages/Settings';
import Matrix from './pages/Matrix';

/**
 * Root application component.
 * Sets up routing and global error handling.
 *
 * @returns The App component with routes
 */
function App() {
  return (
    <ErrorBoundary>
      <Routes>
        {/* Public route */}
        <Route path="/login" element={<Login />} />

        {/* Protected routes */}
        <Route
          path="/*"
          element={
            <ProtectedRoute>
              <Layout>
                <Routes>
                  <Route path="/" element={<Navigate to="/dashboard" replace />} />
                  <Route path="/dashboard" element={<Dashboard />} />
                  <Route path="/agents" element={<Agents />} />
                  <Route path="/techniques" element={<Techniques />} />
                  <Route path="/matrix" element={<Matrix />} />
                  <Route path="/scenarios" element={<Scenarios />} />
                  <Route path="/executions" element={<Executions />} />
                  <Route path="/executions/:id" element={<ExecutionDetails />} />
                  <Route path="/settings" element={<Settings />} />
                </Routes>
              </Layout>
            </ProtectedRoute>
          }
        />
      </Routes>
    </ErrorBoundary>
  );
}

export default App;

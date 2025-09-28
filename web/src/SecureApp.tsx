import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { SecureAuthProvider } from './components/auth/SecureAuthProvider';
import { SecureProtectedRoute } from './components/auth/SecureProtectedRoute';
import { SessionManager } from './components/auth/SessionManager';
import { SentryErrorBoundary } from './components/error/SentryErrorBoundary';
import { SecureLoginPage } from './pages/auth/SecureLoginPage';
import { RegisterPage } from './pages/auth/RegisterPage';
import Layout from './components/layout/Layout';
import { Dashboard } from './pages/Dashboard';
import { DashboardDemo } from './pages/DashboardDemo';
import ClusterDetail from './pages/ClusterDetail';
import ClustersPage from './pages/ClustersPage';
import NodeDetail from './pages/NodeDetail';
import NodesPage from './pages/NodesPage';
import { UserProfile } from './components/auth/UserProfile';
import type { User, LoginRequest, RegisterRequest } from './types';
import './App.css';

function SecureApp() {
  return (
    <SentryErrorBoundary>
      <Router>
        <SecureAuthProvider>
          <SessionManager>
            <Routes>
              {/* Public authentication routes */}
              <Route path="/login" element={<SecureLoginPage />} />
              <Route path="/register" element={<RegisterPage />} />

              {/* Temporary demo route for testing UI */}
              <Route path="/demo" element={<DashboardDemo />} />

              {/* Main landing page - new demo dashboard */}
              <Route path="/" element={<DashboardDemo />} />

              {/* Protected routes that require authentication */}
              <Route
                path="/admin"
                element={
                  <SecureProtectedRoute>
                    <Layout />
                  </SecureProtectedRoute>
                }
              >
                {/* Dashboard and main application routes */}
                <Route index element={<Dashboard />} />
                <Route path="dashboard" element={<Navigate to="/admin" replace />} />

                {/* Cluster management routes */}
                <Route path="clusters" element={<ClustersPage />} />
                <Route path="clusters/:id" element={<ClusterDetail />} />

                {/* Node management routes */}
                <Route path="nodes" element={<NodesPage />} />
                <Route path="nodes/:id" element={<NodeDetail />} />

                {/* User profile route */}
                <Route path="profile" element={<UserProfile />} />

                {/* Admin-only routes */}
                <Route
                  path="settings/*"
                  element={
                    <SecureProtectedRoute requiredRole="admin">
                      <AdminRoutes />
                    </SecureProtectedRoute>
                  }
                />

                {/* Catch-all route for 404 */}
                <Route path="*" element={<NotFound />} />
              </Route>
            </Routes>
          </SessionManager>
        </SecureAuthProvider>
      </Router>
    </SentryErrorBoundary>
  );
}

// Admin-only routes component
const AdminRoutes: React.FC = () => {
  return (
    <Routes>
      <Route path="/users" element={<AdminUsersPage />} />
      <Route path="/system" element={<AdminSystemPage />} />
      <Route path="/settings" element={<AdminSettingsPage />} />
      <Route path="*" element={<AdminDashboard />} />
    </Routes>
  );
};

// Placeholder admin components
const AdminDashboard: React.FC = () => (
  <div className="p-6">
    <h1 className="text-2xl font-bold mb-4">Admin Dashboard</h1>
    <p>Welcome to the admin panel. Admin features will be implemented here.</p>
    <div className="mt-4 p-4 bg-green-50 border border-green-200 rounded-md">
      <p className="text-sm text-green-700">
        🔒 <strong>Security Note:</strong> This admin area is protected with secure httpOnly cookies and CSRF tokens.
      </p>
    </div>
  </div>
);

const AdminUsersPage: React.FC = () => (
  <div className="p-6">
    <h1 className="text-2xl font-bold mb-4">User Management</h1>
    <p>User management features will be implemented here.</p>
  </div>
);

const AdminSystemPage: React.FC = () => (
  <div className="p-6">
    <h1 className="text-2xl font-bold mb-4">System Administration</h1>
    <p>System administration features will be implemented here.</p>
  </div>
);

const AdminSettingsPage: React.FC = () => (
  <div className="p-6">
    <h1 className="text-2xl font-bold mb-4">System Settings</h1>
    <p>System settings will be implemented here.</p>
  </div>
);

// Enhanced 404 component with security context
const NotFound: React.FC = () => {
  return (
    <div className="min-h-full flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="text-center">
        <div className="text-6xl mb-4">🔒</div>
        <h1 className="text-4xl font-bold text-gray-900 mb-4">404</h1>
        <h2 className="text-2xl font-semibold text-gray-700 mb-4">Secure Page Not Found</h2>
        <p className="text-gray-600 mb-8 max-w-md">
          The page you're looking for doesn't exist or you don't have permission to access it.
        </p>
        <div className="space-x-4">
          <a
            href="/admin"
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
          >
            Go to Dashboard
          </a>
          <button
            onClick={() => window.history.back()}
            className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
          >
            Go Back
          </button>
        </div>
      </div>
    </div>
  );
};

export default SecureApp;
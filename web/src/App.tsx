import React, { createContext, useContext } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './components/auth/AuthProvider';
import { ProtectedRoute } from './components/auth/ProtectedRoute';
import { SessionManager } from './components/auth/SessionManager';
import { SentryErrorBoundary } from './components/error/SentryErrorBoundary';
import { LoginPage } from './pages/auth/LoginPage';
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

function App() {
  return (
    <SentryErrorBoundary>
      <Router>
        <AuthProvider>
          <SessionManager>
            <Routes>
          {/* Public authentication routes */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />

          {/* Temporary demo route for testing UI */}
          <Route path="/demo" element={<DashboardDemo />} />

          {/* Main landing page - new demo dashboard */}
          <Route path="/" element={<DashboardDemo />} />

          {/* Protected routes that require authentication */}
          <Route
            path="/admin"
            element={
              <ProtectedRoute>
                <Layout />
              </ProtectedRoute>
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
                <ProtectedRoute requiredRole="admin">
                  <AdminRoutes />
                </ProtectedRoute>
              }
            />

            {/* Catch-all route for 404 */}
            <Route path="*" element={<NotFound />} />
          </Route>
            </Routes>
          </SessionManager>
        </AuthProvider>
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

// Enhanced 404 component
const NotFound: React.FC = () => {
  return (
    <div className="min-h-full flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="text-center">
        <div className="text-6xl mb-4">🤖</div>
        <h1 className="text-4xl font-bold text-gray-900 mb-4">404</h1>
        <h2 className="text-2xl font-semibold text-gray-700 mb-4">Page Not Found</h2>
        <p className="text-gray-600 mb-8 max-w-md">
          The page you're looking for doesn't exist or you don't have permission to access it.
        </p>
        <div className="space-x-4">
          <a
            href="/"
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

// Mock auth context for demo
interface MockAuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  login: (credentials: LoginRequest) => Promise<void>;
  register: (userData: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  clearError: () => void;
}

const MockAuthContext = createContext<MockAuthContextType | undefined>(undefined);

const DemoWrapper: React.FC = () => {
  const mockUser: User = {
    id: '1',
    username: 'demo-admin',
    email: 'admin@demo.com',
    role: 'admin',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };

  const mockAuthValue: MockAuthContextType = {
    user: mockUser,
    isAuthenticated: true,
    isLoading: false,
    error: null,
    login: async () => {},
    register: async () => {},
    logout: async () => {},
    clearError: () => {},
  };

  return (
    <MockAuthContext.Provider value={mockAuthValue}>
      <Layout>
        <Dashboard />
      </Layout>
    </MockAuthContext.Provider>
  );
};

export default App;

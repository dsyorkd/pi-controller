import React, { useEffect, useState, useCallback } from 'react';
import { useAuthContext } from './AuthProvider';
import { getTimeUntilExpiry, isTokenExpired } from '../../utils/session';

interface SessionManagerProps {
  children: React.ReactNode;
}

/**
 * SessionManager component handles automatic session management including:
 * - Session expiry warnings
 * - Auto-logout on token expiry
 * - Session extension prompts
 */
export const SessionManager: React.FC<SessionManagerProps> = ({ children }) => {
  const { user, logout } = useAuthContext();
  const [showWarning, setShowWarning] = useState(false);
  const [timeLeft, setTimeLeft] = useState(0);

  const WARNING_TIME = 5 * 60 * 1000; // Show warning 5 minutes before expiry
  const UPDATE_INTERVAL = 1000; // Update every second

  const checkSession = useCallback(() => {
    if (!user) {
      setShowWarning(false);
      return;
    }

    const timeUntilExpiry = getTimeUntilExpiry();

    if (isTokenExpired()) {
      // Token has expired, logout immediately
      logout();
      return;
    }

    if (timeUntilExpiry <= WARNING_TIME && timeUntilExpiry > 0) {
      setShowWarning(true);
      setTimeLeft(timeUntilExpiry);
    } else {
      setShowWarning(false);
    }
  }, [user, logout]);

  // Check session status periodically
  useEffect(() => {
    if (!user) return;

    checkSession();

    const interval = setInterval(checkSession, UPDATE_INTERVAL);

    return () => clearInterval(interval);
  }, [user, checkSession]);

  const formatTimeLeft = (milliseconds: number): string => {
    const minutes = Math.floor(milliseconds / (1000 * 60));
    const seconds = Math.floor((milliseconds % (1000 * 60)) / 1000);
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  };

  const handleExtendSession = async () => {
    try {
      // Force a profile refresh which will extend the session
      window.location.reload();
    } catch (error) {
      console.error('Failed to extend session:', error);
      logout();
    }
  };

  const handleLogoutNow = () => {
    logout();
  };

  return (
    <>
      {children}

      {/* Session Warning Modal */}
      {showWarning && (
        <div className="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50">
          <div className="relative top-20 mx-auto p-5 border w-96 shadow-lg rounded-md bg-white">
            <div className="mt-3 text-center">
              <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-yellow-100">
                <svg
                  className="h-6 w-6 text-yellow-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"
                  />
                </svg>
              </div>

              <h3 className="text-lg leading-6 font-medium text-gray-900 mt-4">
                Session Expiring Soon
              </h3>

              <div className="mt-2 px-7 py-3">
                <p className="text-sm text-gray-500">
                  Your session will expire in{' '}
                  <span className="font-semibold text-red-600">
                    {formatTimeLeft(timeLeft)}
                  </span>
                </p>
                <p className="text-sm text-gray-500 mt-2">
                  Would you like to extend your session or log out now?
                </p>
              </div>

              <div className="flex gap-3 mt-6">
                <button
                  type="button"
                  onClick={handleExtendSession}
                  className="flex-1 inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-blue-600 text-base font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:text-sm"
                >
                  Extend Session
                </button>

                <button
                  type="button"
                  onClick={handleLogoutNow}
                  className="flex-1 inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-4 py-2 bg-white text-base font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:text-sm"
                >
                  Logout Now
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
};
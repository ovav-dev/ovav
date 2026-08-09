'use client';

import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import api, { SessionUser } from './api';

interface AuthContextType {
  user: SessionUser | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string) => Promise<void>;
  logout: () => void;
  refreshAuth: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<SessionUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Check for existing session on mount
    const checkAuth = async () => {
      if (!api.isAuthenticated()) {
        setIsLoading(false);
        return;
      }

      try {
        const userData = await api.getSession();
        setUser(userData);
      } catch (error) {
        api.clearToken();
        setUser(null);
      }
      
      setIsLoading(false);
    };

    checkAuth();
  }, []);

  const login = async (email: string) => {
    // For magic link login, the user will receive an email
    // The actual token is set after they click the link
    // For demo purposes, we'll just simulate a logged-in state
    await api.login(email);
  };

  const logout = () => {
    api.clearToken();
    setUser(null);
  };

  const refreshAuth = async () => {
    try {
      const userData = await api.getSession();
      setUser(userData);
    } catch {
      logout();
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        login,
        logout,
        refreshAuth,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

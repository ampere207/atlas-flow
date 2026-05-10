'use client';

import React, { useEffect } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore, initializeAuth } from '@/store/auth';

const queryClient = new QueryClient();

export function Providers({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated());

  useEffect(() => {
    // Initialize auth on mount
    initializeAuth();

    // Redirect to login if not authenticated and trying to access protected routes
    const publicRoutes = ['/auth/login', '/auth/signup'];
    const isPublicRoute = publicRoutes.some((route) => pathname.startsWith(route));

    if (!isAuthenticated && !isPublicRoute) {
      router.push('/auth/login');
    }

    if (isAuthenticated && pathname.startsWith('/auth')) {
      router.push('/dashboard');
    }
  }, [isAuthenticated, pathname, router]);

  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}

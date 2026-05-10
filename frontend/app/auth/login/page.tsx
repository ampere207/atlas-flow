import { Suspense } from 'react';

import LoginForm from './login-form';

function LoginFallback() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8">
        <div>
          <h1 className="text-center text-3xl font-bold text-gray-900">AtlasFlow</h1>
          <p className="text-center text-gray-600 mt-2">Distributed Workflow Orchestration</p>
          <h2 className="mt-6 text-center text-2xl font-bold text-gray-900">Sign in</h2>
        </div>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={<LoginFallback />}>
      <LoginForm />
    </Suspense>
  );
}

import { AuthLayout } from '../../components/layout/AuthLayout'
import { authTitleClass } from '../../lib/formStyles'
import { AuthForm } from './AuthForm'
import { AuthTabs } from './AuthTabs'

export function AuthPage({ tab }: { tab: 'login' | 'register' }) {
  return (
    <AuthLayout>
      <h1 key={tab} className={`${authTitleClass} animate-fade-in`}>
        {tab === 'login' ? 'Welcome back' : 'Create your account'}
      </h1>
      <AuthTabs active={tab} />
      <AuthForm mode={tab} />
    </AuthLayout>
  )
}

import { AuthLayout } from '../../components/layout/AuthLayout'
import { authTitleClass } from '../../lib/formStyles'
import { AuthTabs } from './AuthTabs'
import { LoginForm } from './LoginForm'
import { RegisterForm } from './RegisterForm'

export function AuthPage({ tab }: { tab: 'login' | 'register' }) {
  return (
    <AuthLayout>
      <div>
        <h1 className={authTitleClass}>{tab === 'login' ? 'Welcome back' : 'Create your account'}</h1>
        <p className="mt-1.5 text-[15px] leading-[22px] text-ink-muted">
          {tab === 'login' ? 'Log in to pick up where you left off.' : 'Start logging what you spend in a few taps.'}
        </p>
      </div>
      <AuthTabs active={tab} />
      {tab === 'login' ? <LoginForm /> : <RegisterForm />}
    </AuthLayout>
  )
}

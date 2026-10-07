import { AuthLayout } from '../../components/layout/AuthLayout'
import { AuthTabs } from './AuthTabs'
import { LoginForm } from './LoginForm'
import { RegisterForm } from './RegisterForm'

export function AuthPage({ tab }: { tab: 'login' | 'register' }) {
  return (
    <AuthLayout>
      <div>
        <h1 className="heading text-[26px] leading-8 text-ink">{tab === 'login' ? 'Welcome back' : 'Create your account'}</h1>
        <p className="mt-1 text-[15px] leading-[22px] text-ink-muted">
          {tab === 'login' ? 'Log in to pick up where you left off.' : 'A few details and you’re in.'}
        </p>
      </div>
      <AuthTabs active={tab} />
      {tab === 'login' ? <LoginForm /> : <RegisterForm />}
    </AuthLayout>
  )
}

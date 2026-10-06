import { AuthLayout } from '../../components/layout/AuthLayout'
import { AuthTabs } from './AuthTabs'
import { LoginForm } from './LoginForm'
import { RegisterForm } from './RegisterForm'

export function AuthPage({ tab }: { tab: 'login' | 'register' }) {
  return (
    <AuthLayout>
      <h1 className="sr-only">{tab === 'login' ? 'Log in to $pend' : 'Create a $pend account'}</h1>
      <AuthTabs active={tab} />
      {tab === 'login' ? <LoginForm /> : <RegisterForm />}
    </AuthLayout>
  )
}

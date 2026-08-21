// SPDX-License-Identifier: AGPL-3.0-only

import { useEffect, useState } from 'react'

import { DashboardPage } from '@/components/dashboard-page'
import { LoadingPage } from '@/components/loading-page'
import { LoginPage } from '@/components/login-page'
import { SetupPage } from '@/components/setup-page'
import {
  ApiError,
  getSession,
  getSetupStatus,
  getVersion,
  type User,
  type VersionInfo,
} from '@/lib/api'

type AppState =
  | { name: 'loading' }
  | { name: 'setup'; version: VersionInfo }
  | { name: 'login'; version: VersionInfo }
  | { name: 'dashboard'; user: User; version: VersionInfo }
  | { name: 'error'; message: string }

export default function App() {
  const [state, setState] = useState<AppState>({ name: 'loading' })

  useEffect(() => {
    const controller = new AbortController()

    async function bootstrap() {
      try {
        const [setup, version, user] = await Promise.all([
          getSetupStatus(controller.signal),
          getVersion(controller.signal),
          getSession(controller.signal).catch((error: unknown) => {
            if (error instanceof ApiError && error.status === 401) {
              return null
            }
            throw error
          }),
        ])

        if (!setup.completed) {
          setState({ name: 'setup', version })
        } else if (user) {
          setState({ name: 'dashboard', user, version })
        } else {
          setState({ name: 'login', version })
        }
      } catch (error) {
        if (!controller.signal.aborted) {
          setState({ name: 'error', message: getErrorMessage(error) })
        }
      }
    }

    void bootstrap()
    return () => controller.abort()
  }, [])

  switch (state.name) {
    case 'loading':
      return <LoadingPage />
    case 'setup':
      return (
        <SetupPage
          version={state.version}
          onComplete={(user) =>
            setState({ name: 'dashboard', user, version: state.version })
          }
        />
      )
    case 'login':
      return (
        <LoginPage
          version={state.version}
          onAuthenticated={(user) =>
            setState({ name: 'dashboard', user, version: state.version })
          }
        />
      )
    case 'dashboard':
      return (
        <DashboardPage
          user={state.user}
          version={state.version}
          onLoggedOut={() => setState({ name: 'login', version: state.version })}
        />
      )
    case 'error':
      return <LoadingPage error={state.message} />
  }
}

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'An unexpected error occurred.'
}

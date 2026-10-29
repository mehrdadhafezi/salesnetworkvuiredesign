import type { AllowedAction, AllowedActions } from '../api'

/**
 * Reads a server-authoritative allowed_actions hint. The SPA never infers a
 * permission from a role name, hierarchy, route or hidden control: an action
 * absent from the server hint is treated as not allowed, and the server
 * re-checks every command regardless of what the UI shows.
 */
export function actionHint(actions: AllowedActions | undefined, name: string): AllowedAction {
  const hint = actions?.[name]
  return hint ?? { allowed: false }
}

// Vitest setup: React Testing Library cleans up automatically via the globals-free import below.
import { cleanup, configure } from '@testing-library/react'
import { afterEach } from 'vitest'

// Routes are lazy and the RM-E02 shell resolves navigation from two server
// answers; under a parallel full-suite run the first render of a file can
// exceed RTL's 1 s default. Waiting longer never weakens an assertion.
configure({ asyncUtilTimeout: 3000 })

afterEach(() => cleanup())

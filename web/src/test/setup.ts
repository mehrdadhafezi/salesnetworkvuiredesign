// Vitest setup: React Testing Library cleans up automatically via the globals-free import below.
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

afterEach(() => cleanup())

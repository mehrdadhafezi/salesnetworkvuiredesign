// The E2E suite runs under Node (Playwright); only the environment is read.
declare const process: { env: Record<string, string | undefined> }
// RM-E01 evidence spec writes one JSON report; only this Node API is used.
declare module 'node:fs' { export function writeFileSync(path: string, data: string): void }

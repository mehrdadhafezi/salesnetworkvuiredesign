// Fixture: violates the Slice 11 reconcile-before-retry rule. Never compiled.
package writeback

// s11-blind-retry: an UNKNOWN command is never re-applied blindly.
func retryUnknown() { conn.Apply(ctx, in) }

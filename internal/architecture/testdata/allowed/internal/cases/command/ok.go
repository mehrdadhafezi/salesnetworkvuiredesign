// Fixture: a command write with its outbox append. Never compiled.
package command

import _ "example.test/mod/internal/platform/outbox"

const bump = "UPDATE cases.work_case SET version = version + 1 WHERE case_id = $1"

func record() { outbox.Append(ctx, tx, ev, now) }

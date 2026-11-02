// Fixture: Apply only from apply(). Never compiled.
package writeback

func apply() { w.connector.Apply(ctx, in) }

func reconcile() { w.connector.Observe(ctx, in) }

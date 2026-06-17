# Architecture

What the library provides, and how it sequences a reconcile.

## What's in the box

- **The reconciler** — the heart of the library: the loop, and the small set of operations a provider plugs into it.
- **The resource model** — the shape a managed resource must have (its conditions live on its status, its list can enumerate its items), finalizer handling, and helpers to read credentials from secrets and config maps.
- **Standard conditions and status types** — the `Ready` and `Synced` conditions and their reasons.
- **Cross-cutting helpers** — rate limiters, controller options, a logging abstraction, optional telemetry, an errors helper, and a test toolkit (a mock client and comparison helpers).

The event recorder comes from the shared Krateo libraries, not from this repo.

## How the reconcile loop is sequenced

The loop runs in a fixed order on every reconcile. The order is the important part — read it top to bottom:

1. **Set timeouts** for the reconcile and for the external calls.
2. **Get the managed resource**; a not-found is treated as "nothing to do".
3. **Pause** — if the resource is paused, record a paused condition and stop.
4. **Orphan fast-delete** — if the resource is being deleted and its policy says not to delete the external resource, just drop the finalizer and stop.
5. **Create-incomplete guard** — if a previous create may have started without confirming, refuse to proceed (this prevents leaking an external resource).
6. **Connect**, then **Observe**.
7. **Creation grace period** — if the resource doesn't exist yet but was created very recently, wait and retry (tolerates eventually-consistent backends).
8. **Delete path** — if being deleted and the external resource exists, delete it, then drop the finalizer once it's gone.
9. **Add the finalizer** for live resources.
10. **Create path** — mark the create as pending, create, then record success or failure.
11. **Persist any defaults** Observe filled in.
12. **Up to date** — record success and requeue after the poll interval.
13. **Update path** — update, record success, requeue.

Two behaviors run through nearly every step: conditions and status are **persisted after each branch**, and a **conflict is treated as a requeue, not an error** (so concurrent writers simply retry). Mirror that conflict handling in any custom finalizer or updater you add.

> Constructing the reconciler requires the resource's kind to be registered in the manager's scheme — if it isn't, construction fails immediately rather than misbehaving later. Register your scheme before wiring the reconciler.

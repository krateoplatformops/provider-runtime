# Equivalence: provider-runtime ⟷ unstructured-runtime

> These two libraries deliberately implement the **same managed-resource lifecycle**. `provider-runtime` (this library) drives it for **typed** custom resources; `unstructured-runtime` drives it for **untyped** objects at a resource type chosen at runtime. This appendix lines them up so a change to one can be mirrored in the other. **If you change lifecycle semantics in one, mirror it in the other — divergence is a bug, not a feature.**

This same appendix appears in the **unstructured-runtime** developer guide.

## Concept map

| Concept | provider-runtime (typed) | unstructured-runtime (dynamic) |
| --- | --- | --- |
| What you manage | a typed custom resource, registered in a scheme | untyped objects at a resource type chosen at runtime |
| Operations you implement | Observe / Create / Update / Delete | the same four operations |
| Per-resource setup | a **Connect** step builds a client for each resource | **none** — you register one client for the whole controller |
| What Observe reports | exists, up-to-date, plus "defaults filled in" and a drift description | exists, up-to-date (minimal) |
| How it's wired | a reconciler into a controller-runtime manager | a controller built directly on lower-level primitives — no manager |
| Work queue | the manager's rate-limited queue | a local priority queue (de-duplicating, priority-aware) |
| Concurrency | a max-concurrent-reconciles setting | a fixed number of workers — **no autoscaling** |
| Finalizer | a configurable finalizer | a fixed finalizer name |
| Conditions / status | standard conditions on the typed resource's status | the same conditions written onto the untyped object's status |
| Standard conditions | `Ready` and `Synced`, with the same reasons | the same |
| Pause | a paused annotation short-circuits to a paused condition | the same |
| Create safety | pending / succeeded / failed create-tracking, plus a grace period | the same tracking |
| Lifecycle policies | the loop may skip operations or orphan on delete | the same |
| Type resolution | scheme / RESTMapper (compile-time types) | runtime pluralization |
| Event recorder, logger | shared Krateo helpers | the same |
| Origin | trimmed fork of crossplane-runtime's managed reconciler | the dynamic analog of the same lifecycle |

## Invariants that must stay equivalent

- **Branching from Observe** — missing ⇒ create; exists but drifted ⇒ update; otherwise mark success and requeue.
- **Finalizer discipline** — add the finalizer before creating the external resource; remove it only after a confirmed delete.
- **Create safety** — mark the create pending before doing it, record success or failure after, and refuse to proceed while a create is unconfirmed.
- **Pause** — a paused resource short-circuits to a paused condition without touching the external resource.
- **Conditions** — maintain `Ready` and `Synced` with the same reasons; the framework persists them, not your code.
- **Idempotency** — operations must be idempotent and non-blocking, and a conflict is a requeue, not an error.

## Where they legitimately differ (and why)

- **The Connect step** exists only in provider-runtime — typed providers often build a per-resource client; the dynamic controller registers a single client instead.
- **What Observe reports is richer** in provider-runtime (it also carries "defaults filled in" and a drift description); the dynamic side keeps the minimal two-signal form.
- **The plumbing differs** — provider-runtime rides a controller-runtime manager; unstructured-runtime wires the lower-level primitives itself and brings its own priority queue.
- **Concurrency** — a max-concurrent setting versus a fixed worker count (the dynamic side explicitly favors sharding over autoscaling).
- **Type handling** — compile-time types versus runtime resolution of untyped objects.

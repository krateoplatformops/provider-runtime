# Building a provider

The steps to stand up a provider on this library, using **core-provider** as the worked reference.

## 1. Define the typed resource

Define your custom resource so it fits the managed-resource shape: put the standard conditions on its status (and delegate the condition getters/setters to them), let its list type enumerate its items, and register it in a scheme. The provider supplies its own scheme registration — the library doesn't provide one.

## 2. Implement the operations

Implement **Connect** (build the client that talks to the external system for a given resource) and the four operations — **Observe**, **Create**, **Update**, **Delete** — following the contract in [`01-mental-model.md`](./01-mental-model.md). `Observe` reports whether the external resource exists and is up to date; the others make it so.

## 3. Wire the reconciler into a manager

Construct the reconciler for your resource's kind, passing your `Connect` implementation and options (timeout, poll interval, logger, recorder, metrics). Then register a controller for your resource, wrapping the reconciler with a rate limiter. core-provider does exactly this for its `CompositionDefinition`.

## 4. Bootstrap the process

Register your scheme **before** wiring the reconciler (construction fails fast if the kind isn't in the scheme), set up the controller options (concurrency, poll interval, the global rate limiter, and — if used — the telemetry recorder), and start the manager.

## Testing

The library ships a test toolkit so you can exercise the loop without a cluster: a mock Kubernetes client with per-method hooks, fakes for the managed resource, and comparison helpers for errors and conditions. You can also implement the operations inline for a table-driven test rather than building a full client. The library's own reconciler tests are the best reference for testing your implementation.

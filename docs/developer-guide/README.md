# provider-runtime — Developer Guide

A contributor-facing guide to the library that gives Krateo's **typed** providers a managed-resource controller: implement a small set of operations, and the framework runs the reconcile loop.

> Audience: engineers **building a provider on top of this library, or maintaining the library itself**. This guide explains *ideas and flows*, not line-by-line code. For product concepts, see [docs.krateo.io](https://docs.krateo.io).

## What it is

`provider-runtime` implements the **managed-resource controller pattern** for Krateo providers (core-provider, git-provider, github-provider, and others). A provider author defines a typed custom resource, implements a handful of operations, and wires a reconciler into a manager. The library owns the reconcile loop: finalizers, the safety steps around creating an external resource, the `Ready`/`Synced` conditions, pause and management/deletion policies, requeue and rate-limiting, and optional metrics.

It is a **trimmed, rebranded fork of crossplane-runtime**, with Krateo-specific conventions (its own annotation prefix and finalizer, a shared event recorder) and the removal of crossplane's provider-config and connection-detail machinery. See [`01-mental-model.md`](./01-mental-model.md).

> **Sibling library.** `unstructured-runtime` is the **dynamic/unstructured analog** of this library — the same lifecycle, applied to untyped objects at a resource type chosen at runtime. The two are meant to stay functionally equivalent; the mapping is in [`04-equivalence.md`](./04-equivalence.md).

## Documents in this folder

| Document | What it covers |
| --- | --- |
| [`01-mental-model.md`](./01-mental-model.md) | The managed-resource lifecycle and the operations you implement. |
| [`02-architecture.md`](./02-architecture.md) | What the library provides, and how the reconcile loop is sequenced. |
| [`03-building-a-provider.md`](./03-building-a-provider.md) | The steps to stand up a provider, using core-provider as the worked reference. |
| [`04-equivalence.md`](./04-equivalence.md) | How `provider-runtime` and `unstructured-runtime` line up, concept by concept. |

## See also

- **Ecosystem overview (canonical)** — how Krateo Composable Operations fits together lives in the **core-provider** repo: `core-provider/docs/developer-guide/00-ecosystem-overview.md`.
- **The exemplar consumer** — **core-provider** is the reference provider built on this library.

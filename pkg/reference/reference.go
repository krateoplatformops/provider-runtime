/*
Copyright 2019 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package reference contains utilities for working with cross-resource
// references.
package reference

import (
	"context"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rpv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	"github.com/krateoplatformops/provider-runtime/pkg/errors"
	"github.com/krateoplatformops/provider-runtime/pkg/meta"
	"github.com/krateoplatformops/provider-runtime/pkg/resource"
)

// Error strings.
const (
	errGetManaged  = "cannot get referenced resource"
	errListManaged = "cannot list resources that match selector"
	errNoMatches   = "no resources matched selector"
	errNoValue     = "referenced field was empty (referenced resource may not yet be ready)"
)

// NOTE(negz): There are many equivalents of FromPtrValue and ToPtrValue
// throughout the Crossplane codebase. We duplicate them here to reduce the
// number of packages our API types have to import to support references.

// FromPtrValue adapts a string pointer field for use as a CurrentValue.
func FromPtrValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// ToPtrValue adapts a ResolvedValue for use as a string pointer field.
func ToPtrValue(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// FromPtrValues adapts a slice of string pointer fields for use as CurrentValues.
// NOTE: Do not use this utility function unless you have to.
// Using pointer slices does not adhere to our current API practices.
// The current use case is where generated code creates reference-able fields in a provider which are
// string pointers and need to be resolved as part of `ResolveMultiple`
func FromPtrValues(v []*string) []string {
	var res = make([]string, len(v))
	for i := 0; i < len(v); i++ {
		res[i] = FromPtrValue(v[i])
	}
	return res
}

// ToPtrValues adapts ResolvedValues for use as a slice of string pointer fields.
// NOTE: Do not use this utility function unless you have to.
// Using pointer slices does not adhere to our current API practices.
// The current use case is where generated code creates reference-able fields in a provider which are
// string pointers and need to be resolved as part of `ResolveMultiple`
func ToPtrValues(v []string) []*string {
	var res = make([]*string, len(v))
	for i := 0; i < len(v); i++ {
		res[i] = ToPtrValue(v[i])
	}
	return res
}

// An ExtractValueFn specifies how to extract a value from the resolved managed
// resource.
type ExtractValueFn func(resource.Object) string

// ExternalName extracts the resolved managed resource's external name from its
// external name annotation.
func ExternalName() ExtractValueFn {
	return func(mg resource.Object) string {
		return meta.GetExternalName(mg)
	}
}

// A ResolutionRequest requests that a reference to a particular kind of
// managed resource be resolved.
type ResolutionRequest struct {
	CurrentValue string
	Reference    *rpv1.Reference
	To           resource.Object
	Extract      ExtractValueFn
}

// IsNoOp returns true if the supplied ResolutionRequest cannot or should not be
// processed.
func (rr *ResolutionRequest) IsNoOp() bool {
	isAlways := false
	if rr.Reference != nil {
		if rr.Reference.Policy.IsResolvePolicyAlways() {
			isAlways = true
		}
	}

	// We don't resolve values that are already set (if reference resolution policy
	// is not set to Always); we effectively cache resolved values. The CR author
	// can invalidate the cache and trigger a new resolution by explicitly clearing
	// the resolved value.
	if rr.CurrentValue != "" && !isAlways {
		return true
	}

	// We can't resolve anything if neither a reference nor a selector were
	// provided.
	return rr.Reference == nil
}

// A ResolutionResponse returns the result of a reference resolution. The
// returned values are always safe to set if resolution was successful.
type ResolutionResponse struct {
	ResolvedValue     string
	ResolvedReference *rpv1.Reference
}

// Validate this ResolutionResponse.
func (rr ResolutionResponse) Validate() error {
	if rr.ResolvedValue == "" {
		return errors.New(errNoValue)
	}

	return nil
}

// An APIResolver selects and resolves references to managed resources in the
// Kubernetes API server.
type APIResolver struct {
	client client.Reader
	from   resource.Object
}

// NewAPIResolver returns a Resolver that selects and resolves references from
// the supplied managed resource to other managed resources in the Kubernetes
// API server.
func NewAPIResolver(c client.Reader, from resource.Object) *APIResolver {
	return &APIResolver{client: c, from: from}
}

// Resolve the supplied ResolutionRequest. The returned ResolutionResponse
// always contains valid values unless an error was returned.
func (r *APIResolver) Resolve(ctx context.Context, req ResolutionRequest) (ResolutionResponse, error) {
	// Return early if from is being deleted, or the request is a no-op.
	if meta.WasDeleted(r.from) || req.IsNoOp() {
		return ResolutionResponse{ResolvedValue: req.CurrentValue, ResolvedReference: req.Reference}, nil
	}

	// The reference is already set - resolve it.
	if req.Reference != nil {
		if err := r.client.Get(ctx, types.NamespacedName{Name: req.Reference.Name}, req.To); err != nil {
			if kerrors.IsNotFound(err) {
				return ResolutionResponse{}, getResolutionError(req.Reference.Policy, errors.Wrap(err, errGetManaged))
			}
			return ResolutionResponse{}, errors.Wrap(err, errGetManaged)
		}

		rsp := ResolutionResponse{ResolvedValue: req.Extract(req.To), ResolvedReference: req.Reference}
		return rsp, getResolutionError(req.Reference.Policy, rsp.Validate())
	}

	// We couldn't resolve anything.
	return ResolutionResponse{}, errors.New(errNoMatches)

}

func getResolutionError(p *rpv1.Policy, err error) error {
	if !p.IsResolutionPolicyOptional() {
		return err
	}
	return nil
}

// ControllersMustMatch returns true if the supplied Selector requires that a
// reference be to a managed resource whose controller reference matches the
// referencing resource.
func ControllersMustMatch(s *rpv1.Selector) bool {
	if s == nil {
		return false
	}
	return s.MatchControllerRef != nil && *s.MatchControllerRef
}

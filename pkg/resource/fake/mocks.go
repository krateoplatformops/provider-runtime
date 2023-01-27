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

// Package fake provides fake Crossplane resources for use in tests.
package fake

import (
	"encoding/json"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	prv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
)

// Conditioned is a mock that implements Conditioned interface.
type Conditioned struct{ Conditions []prv1.Condition }

// SetConditions sets the Conditions.
func (m *Conditioned) SetConditions(c ...prv1.Condition) { m.Conditions = c }

// GetCondition get the Condition with the given ConditionType.
func (m *Conditioned) GetCondition(ct prv1.ConditionType) prv1.Condition {
	return prv1.Condition{Type: ct, Status: corev1.ConditionUnknown}
}

// ManagedResourceReferencer is a mock that implements ManagedResourceReferencer interface.
type ManagedResourceReferencer struct{ Ref *corev1.ObjectReference }

// SetResourceReference sets the ResourceReference.
func (m *ManagedResourceReferencer) SetResourceReference(r *corev1.ObjectReference) { m.Ref = r }

// GetResourceReference gets the ResourceReference.
func (m *ManagedResourceReferencer) GetResourceReference() *corev1.ObjectReference { return m.Ref }

// ProviderReferencer is a mock that implements ProviderReferencer interface.
type ProviderReferencer struct{ Ref *prv1.Reference }

// SetProviderReference sets the ProviderReference.
func (m *ProviderReferencer) SetProviderReference(p *prv1.Reference) { m.Ref = p }

// GetProviderReference gets the ProviderReference.
func (m *ProviderReferencer) GetProviderReference() *prv1.Reference { return m.Ref }

// ProviderConfigReferencer is a mock that implements ProviderConfigReferencer interface.
type ProviderConfigReferencer struct{ Ref *prv1.Reference }

// SetProviderConfigReference sets the ProviderConfigReference.
func (m *ProviderConfigReferencer) SetProviderConfigReference(p *prv1.Reference) { m.Ref = p }

// GetProviderConfigReference gets the ProviderConfigReference.
func (m *ProviderConfigReferencer) GetProviderConfigReference() *prv1.Reference { return m.Ref }

// RequiredProviderConfigReferencer is a mock that implements the
// RequiredProviderConfigReferencer interface.
type RequiredProviderConfigReferencer struct{ Ref prv1.Reference }

// SetProviderConfigReference sets the ProviderConfigReference.
func (m *RequiredProviderConfigReferencer) SetProviderConfigReference(p prv1.Reference) {
	m.Ref = p
}

// GetProviderConfigReference gets the ProviderConfigReference.
func (m *RequiredProviderConfigReferencer) GetProviderConfigReference() prv1.Reference {
	return m.Ref
}

// RequiredTypedResourceReferencer is a mock that implements the
// RequiredTypedResourceReferencer interface.
type RequiredTypedResourceReferencer struct{ Ref prv1.TypedReference }

// SetResourceReference sets the ResourceReference.
func (m *RequiredTypedResourceReferencer) SetResourceReference(p prv1.TypedReference) {
	m.Ref = p
}

// GetResourceReference gets the ResourceReference.
func (m *RequiredTypedResourceReferencer) GetResourceReference() prv1.TypedReference {
	return m.Ref
}

// Orphanable implements the Orphanable interface.
type Orphanable struct{ Policy prv1.DeletionPolicy }

// SetDeletionPolicy sets the DeletionPolicy.
func (m *Orphanable) SetDeletionPolicy(p prv1.DeletionPolicy) { m.Policy = p }

// GetDeletionPolicy gets the DeletionPolicy.
func (m *Orphanable) GetDeletionPolicy() prv1.DeletionPolicy { return m.Policy }

// An EnvironmentConfigReferencer is a mock that implements the
// EnvironmentConfigReferencer interface.
type EnvironmentConfigReferencer struct{ Refs []corev1.ObjectReference }

// SetEnvironmentConfigReferences sets the EnvironmentConfig references.
func (m *EnvironmentConfigReferencer) SetEnvironmentConfigReferences(refs []corev1.ObjectReference) {
	m.Refs = refs
}

// GetEnvironmentConfigReferences gets the EnvironmentConfig references.
func (m *EnvironmentConfigReferencer) GetEnvironmentConfigReferences() []corev1.ObjectReference {
	return m.Refs
}

// Object is a mock that implements Object interface.
type Object struct {
	metav1.ObjectMeta
	runtime.Object
}

// GetObjectKind returns schema.ObjectKind.
func (o *Object) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

// DeepCopyObject returns a copy of the object as runtime.Object
func (o *Object) DeepCopyObject() runtime.Object {
	out := &Object{}
	j, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}
	_ = json.Unmarshal(j, out)
	return out
}

// Managed is a mock that implements Managed interface.
type Managed struct {
	metav1.ObjectMeta
	ProviderReferencer
	ProviderConfigReferencer
	Orphanable
	prv1.ConditionedStatus
}

// GetObjectKind returns schema.ObjectKind.
func (m *Managed) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

// DeepCopyObject returns a copy of the object as runtime.Object
func (m *Managed) DeepCopyObject() runtime.Object {
	out := &Managed{}
	j, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	_ = json.Unmarshal(j, out)
	return out
}

// Manager is a mock object that satisfies manager.Manager interface.
type Manager struct {
	manager.Manager

	Client     client.Client
	Scheme     *runtime.Scheme
	Config     *rest.Config
	RESTMapper meta.RESTMapper
}

// Elected returns a closed channel.
func (m *Manager) Elected() <-chan struct{} {
	e := make(chan struct{})
	close(e)
	return e
}

// GetClient returns the client.
func (m *Manager) GetClient() client.Client { return m.Client }

// GetScheme returns the scheme.
func (m *Manager) GetScheme() *runtime.Scheme { return m.Scheme }

// GetConfig returns the config.
func (m *Manager) GetConfig() *rest.Config { return m.Config }

// GetRESTMapper returns the REST mapper.
func (m *Manager) GetRESTMapper() meta.RESTMapper { return m.RESTMapper }

// GV returns a mock schema.GroupVersion.
var GV = schema.GroupVersion{Group: "g", Version: "v"}

// GVK returns the mock GVK of the given object.
func GVK(o runtime.Object) schema.GroupVersionKind {
	return GV.WithKind(reflect.TypeOf(o).Elem().Name())
}

// SchemeWith returns a scheme with list of `runtime.Object`s registered.
func SchemeWith(o ...runtime.Object) *runtime.Scheme {
	s := runtime.NewScheme()
	s.AddKnownTypes(GV, o...)
	return s
}

// ProviderConfig is a mock implementation of the ProviderConfig interface.
type ProviderConfig struct {
	metav1.ObjectMeta

	prv1.ConditionedStatus
}

// GetObjectKind returns schema.ObjectKind.
func (p *ProviderConfig) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

// DeepCopyObject returns a copy of the object as runtime.Object
func (p *ProviderConfig) DeepCopyObject() runtime.Object {
	out := &ProviderConfig{}
	j, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	_ = json.Unmarshal(j, out)
	return out
}

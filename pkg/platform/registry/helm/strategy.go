/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2019 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

package helm

import (
	"context"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	platforminternalclient "tkestack.io/tke/api/client/clientset/internalversion/typed/platform/internalversion"
	"tkestack.io/tke/api/platform"
	"tkestack.io/tke/pkg/apiserver/authentication"
	"tkestack.io/tke/pkg/platform/controller/addon/helm/images"
	"tkestack.io/tke/pkg/util/log"
	namesutil "tkestack.io/tke/pkg/util/names"
)

// Strategy implements verification logic for helm.
type Strategy struct {
	runtime.ObjectTyper
	names.NameGenerator
	platformClient platforminternalclient.PlatformInterface
}

var _ rest.RESTCreateStrategy = &Strategy{}
var _ rest.RESTUpdateStrategy = &Strategy{}
var _ rest.RESTDeleteStrategy = &Strategy{}

// NewStrategy creates a strategy that is the default logic that applies when
// creating and updating namespace set objects.
func NewStrategy(platformClient platforminternalclient.PlatformInterface) *Strategy {
	return &Strategy{platform.Scheme, namesutil.Generator, platformClient}
}

// DefaultGarbageCollectionPolicy returns the default garbage collection behavior.
func (Strategy) DefaultGarbageCollectionPolicy(ctx context.Context) rest.GarbageCollectionPolicy {
	return rest.Unsupported
}

// NamespaceScoped is false for namespaceSets
func (Strategy) NamespaceScoped() bool {
	return false
}

// Export strips fields that can not be set by the user.
func (Strategy) Export(ctx context.Context, obj runtime.Object, exact bool) error {
	return nil
}

// PrepareForCreate is invoked on create before validation to normalize
// the object.
func (Strategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	_, tenantID := authentication.UsernameAndTenantID(ctx)
	helm, _ := obj.(*platform.Helm)

	if len(tenantID) != 0 {
		helm.Spec.TenantID = tenantID
	}

	if helm.Name == "" && helm.GenerateName == "" {
		helm.GenerateName = "hm-"
	}

	if helm.Spec.Version == "" {
		helm.Spec.Version = images.V1Version
	}
}

// PrepareForUpdate is invoked on update before validation to normalize the
// object.
func (Strategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	_, tenantID := authentication.UsernameAndTenantID(ctx)
	if len(tenantID) != 0 {
		oldHelm := old.(*platform.Helm)
		helm, _ := obj.(*platform.Helm)
		if oldHelm.Spec.TenantID != tenantID {
			log.Panic("Unauthorized update helm information", log.String("oldTenantID", oldHelm.Spec.TenantID), log.String("newTenantID", helm.Spec.TenantID), log.String("userTenantID", tenantID))
		}
		helm.Spec.TenantID = tenantID
	}
}

// Validate validates a new helm.
func (s *Strategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return ValidateHelm(ctx, obj.(*platform.Helm), s.platformClient)
}

// AllowCreateOnUpdate is false for persistent events
func (Strategy) AllowCreateOnUpdate() bool {
	return false
}

// AllowUnconditionalUpdate returns true if the object can be updated
// unconditionally (irrespective of the latest resource version), when there is
// no resource version specified in the object.
func (Strategy) AllowUnconditionalUpdate() bool {
	return false
}

// Canonicalize normalizes the object after validation.
func (Strategy) Canonicalize(obj runtime.Object) {
}

// ValidateUpdate is the default update validation for an end namespace set.
func (s *Strategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return ValidateHelmUpdate(ctx, obj.(*platform.Helm), old.(*platform.Helm), s.platformClient)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	helm, _ := obj.(*platform.Helm)
	return helm.ObjectMeta.Labels, ToSelectableFields(helm), nil
}

// MatchHelm returns a generic matcher for a given label and field selector.
func MatchHelm(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
		IndexFields: []string{
			"spec.tenantID",
			"spec.clusterName",
			"spec.version",
			"status.version",
			"status.phase"},
	}
}

// ToSelectableFields returns a field set that represents the object
func ToSelectableFields(helm *platform.Helm) fields.Set {
	objectMetaFieldsSet := genericregistry.ObjectMetaFieldsSet(&helm.ObjectMeta, false)
	specificFieldsSet := fields.Set{
		"spec.tenantID":    helm.Spec.TenantID,
		"spec.clusterName": helm.Spec.ClusterName,
		"spec.version":     helm.Spec.Version,
		"status.version":   helm.Status.Version,
		"status.phase":     string(helm.Status.Phase),
	}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}

// StatusStrategy implements verification logic for status of Helm.
type StatusStrategy struct {
	*Strategy
}

var _ rest.RESTUpdateStrategy = &StatusStrategy{}

// NewStatusStrategy create the StatusStrategy object by given strategy.
func NewStatusStrategy(strategy *Strategy) *StatusStrategy {
	return &StatusStrategy{strategy}
}

// PrepareForUpdate is invoked on update before validation to normalize
// the object.
func (StatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newHelm := obj.(*platform.Helm)
	oldHelm := old.(*platform.Helm)
	newHelm.Spec = oldHelm.Spec
}

// ValidateUpdate is invoked after default fields in the object have been
// filled in before the object is persisted.
func (StatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

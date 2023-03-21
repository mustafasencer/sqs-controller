// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Code generated by ack-generate. DO NOT EDIT.

package queue

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamapitypes "github.com/aws-controllers-k8s/iam-controller/apis/v1alpha1"
	kmsapitypes "github.com/aws-controllers-k8s/kms-controller/apis/v1alpha1"
	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	acktypes "github.com/aws-controllers-k8s/runtime/pkg/types"

	svcapitypes "github.com/aws-controllers-k8s/sqs-controller/apis/v1alpha1"
)

// +kubebuilder:rbac:groups=kms.services.k8s.aws,resources=keys,verbs=get;list
// +kubebuilder:rbac:groups=kms.services.k8s.aws,resources=keys/status,verbs=get;list

// +kubebuilder:rbac:groups=iam.services.k8s.aws,resources=policies,verbs=get;list
// +kubebuilder:rbac:groups=iam.services.k8s.aws,resources=policies/status,verbs=get;list

// ResolveReferences finds if there are any Reference field(s) present
// inside AWSResource passed in the parameter and attempts to resolve
// those reference field(s) into target field(s).
// It returns an AWSResource with resolved reference(s), and an error if the
// passed AWSResource's reference field(s) cannot be resolved.
// This method also adds/updates the ConditionTypeReferencesResolved for the
// AWSResource.
func (rm *resourceManager) ResolveReferences(
	ctx context.Context,
	apiReader client.Reader,
	res acktypes.AWSResource,
) (acktypes.AWSResource, error) {
	namespace := res.MetaObject().GetNamespace()
	ko := rm.concreteResource(res).ko.DeepCopy()
	err := validateReferenceFields(ko)
	if err == nil {
		err = resolveReferenceForKMSMasterKeyID(ctx, apiReader, namespace, ko)
	}
	if err == nil {
		err = resolveReferenceForPolicy(ctx, apiReader, namespace, ko)
	}

	// If there was an error while resolving any reference, reset all the
	// resolved values so that they do not get persisted inside etcd
	if err != nil {
		ko = rm.concreteResource(res).ko.DeepCopy()
	}
	if hasNonNilReferences(ko) {
		return ackcondition.WithReferencesResolvedCondition(&resource{ko}, err)
	}
	return &resource{ko}, err
}

// validateReferenceFields validates the reference field and corresponding
// identifier field.
func validateReferenceFields(ko *svcapitypes.Queue) error {
	if ko.Spec.KMSMasterKeyRef != nil && ko.Spec.KMSMasterKeyID != nil {
		return ackerr.ResourceReferenceAndIDNotSupportedFor("KMSMasterKeyID", "KMSMasterKeyRef")
	}
	if ko.Spec.PolicyRef != nil && ko.Spec.Policy != nil {
		return ackerr.ResourceReferenceAndIDNotSupportedFor("Policy", "PolicyRef")
	}
	return nil
}

// hasNonNilReferences returns true if resource contains a reference to another
// resource
func hasNonNilReferences(ko *svcapitypes.Queue) bool {
	return false || (ko.Spec.KMSMasterKeyRef != nil) || (ko.Spec.PolicyRef != nil)
}

// resolveReferenceForKMSMasterKeyID reads the resource referenced
// from KMSMasterKeyRef field and sets the KMSMasterKeyID
// from referenced resource
func resolveReferenceForKMSMasterKeyID(
	ctx context.Context,
	apiReader client.Reader,
	namespace string,
	ko *svcapitypes.Queue,
) error {
	if ko.Spec.KMSMasterKeyRef != nil && ko.Spec.KMSMasterKeyRef.From != nil {
		arr := ko.Spec.KMSMasterKeyRef.From
		if arr == nil || arr.Name == nil || *arr.Name == "" {
			return fmt.Errorf("provided resource reference is nil or empty: KMSMasterKeyRef")
		}
		obj := &kmsapitypes.Key{}
		if err := getReferencedResourceState_Key(ctx, apiReader, obj, *arr.Name, namespace); err != nil {
			return err
		}
		ko.Spec.KMSMasterKeyID = (*string)(obj.Status.KeyID)
	}

	return nil
}

// getReferencedResourceState_Key looks up whether a referenced resource
// exists and is in a ACK.ResourceSynced=True state. If the referenced resource does exist and is
// in a Synced state, returns nil, otherwise returns `ackerr.ResourceReferenceTerminalFor` or
// `ResourceReferenceNotSyncedFor` depending on if the resource is in a Terminal state.
func getReferencedResourceState_Key(
	ctx context.Context,
	apiReader client.Reader,
	obj *kmsapitypes.Key,
	name string, // the Kubernetes name of the referenced resource
	namespace string, // the Kubernetes namespace of the referenced resource
) error {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err := apiReader.Get(ctx, namespacedName, obj)
	if err != nil {
		return err
	}
	var refResourceSynced, refResourceTerminal bool
	for _, cond := range obj.Status.Conditions {
		if cond.Type == ackv1alpha1.ConditionTypeResourceSynced &&
			cond.Status == corev1.ConditionTrue {
			refResourceSynced = true
		}
		if cond.Type == ackv1alpha1.ConditionTypeTerminal &&
			cond.Status == corev1.ConditionTrue {
			return ackerr.ResourceReferenceTerminalFor(
				"Key",
				namespace, name)
		}
	}
	if refResourceTerminal {
		return ackerr.ResourceReferenceTerminalFor(
			"Key",
			namespace, name)
	}
	if !refResourceSynced {
		return ackerr.ResourceReferenceNotSyncedFor(
			"Key",
			namespace, name)
	}
	if obj.Status.KeyID == nil {
		return ackerr.ResourceReferenceMissingTargetFieldFor(
			"Key",
			namespace, name,
			"Status.KeyID")
	}
	return nil
}

// resolveReferenceForPolicy reads the resource referenced
// from PolicyRef field and sets the Policy
// from referenced resource
func resolveReferenceForPolicy(
	ctx context.Context,
	apiReader client.Reader,
	namespace string,
	ko *svcapitypes.Queue,
) error {
	if ko.Spec.PolicyRef != nil && ko.Spec.PolicyRef.From != nil {
		arr := ko.Spec.PolicyRef.From
		if arr == nil || arr.Name == nil || *arr.Name == "" {
			return fmt.Errorf("provided resource reference is nil or empty: PolicyRef")
		}
		obj := &iamapitypes.Policy{}
		if err := getReferencedResourceState_Policy(ctx, apiReader, obj, *arr.Name, namespace); err != nil {
			return err
		}
		ko.Spec.Policy = (*string)(obj.Spec.PolicyDocument)
	}

	return nil
}

// getReferencedResourceState_Policy looks up whether a referenced resource
// exists and is in a ACK.ResourceSynced=True state. If the referenced resource does exist and is
// in a Synced state, returns nil, otherwise returns `ackerr.ResourceReferenceTerminalFor` or
// `ResourceReferenceNotSyncedFor` depending on if the resource is in a Terminal state.
func getReferencedResourceState_Policy(
	ctx context.Context,
	apiReader client.Reader,
	obj *iamapitypes.Policy,
	name string, // the Kubernetes name of the referenced resource
	namespace string, // the Kubernetes namespace of the referenced resource
) error {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err := apiReader.Get(ctx, namespacedName, obj)
	if err != nil {
		return err
	}
	var refResourceSynced, refResourceTerminal bool
	for _, cond := range obj.Status.Conditions {
		if cond.Type == ackv1alpha1.ConditionTypeResourceSynced &&
			cond.Status == corev1.ConditionTrue {
			refResourceSynced = true
		}
		if cond.Type == ackv1alpha1.ConditionTypeTerminal &&
			cond.Status == corev1.ConditionTrue {
			return ackerr.ResourceReferenceTerminalFor(
				"Policy",
				namespace, name)
		}
	}
	if refResourceTerminal {
		return ackerr.ResourceReferenceTerminalFor(
			"Policy",
			namespace, name)
	}
	if !refResourceSynced {
		return ackerr.ResourceReferenceNotSyncedFor(
			"Policy",
			namespace, name)
	}
	if obj.Spec.PolicyDocument == nil {
		return ackerr.ResourceReferenceMissingTargetFieldFor(
			"Policy",
			namespace, name,
			"Spec.PolicyDocument")
	}
	return nil
}

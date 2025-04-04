/*
 * Copyright (C) 2025 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions
 * and limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

//nolint:stylecheck,revive // Inherited from opensource.
package nexus_client

import (
	"errors"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsNotFound(err error) bool {
	return k8serrors.IsNotFound(err)
}

func IsUnauthorized(err error) bool {
	return k8serrors.IsUnauthorized(err)
}

func IsAlreadyExists(err error) bool {
	return k8serrors.IsAlreadyExists(err)
}

func IsConflict(err error) bool {
	return k8serrors.IsConflict(err)
}

func IsInvalid(err error) bool {
	return k8serrors.IsInvalid(err)
}

func IsGone(err error) bool {
	return k8serrors.IsGone(err)
}

func IsResourceExpired(err error) bool {
	return k8serrors.IsResourceExpired(err)
}

func IsNotAcceptable(err error) bool {
	return k8serrors.IsNotAcceptable(err)
}

func IsUnsupportedMediaType(err error) bool {
	return k8serrors.IsUnsupportedMediaType(err)
}

func IsMethodNotSupported(err error) bool {
	return k8serrors.IsMethodNotSupported(err)
}

func IsServiceUnavailable(err error) bool {
	return k8serrors.IsServiceUnavailable(err)
}

func IsBadRequest(err error) bool {
	return k8serrors.IsBadRequest(err)
}

func IsForbidden(err error) bool {
	return k8serrors.IsForbidden(err)
}

func IsTimeout(err error) bool {
	return k8serrors.IsTimeout(err)
}

func IsServerTimeout(err error) bool {
	return k8serrors.IsServerTimeout(err)
}

func IsInternalError(err error) bool {
	return k8serrors.IsInternalError(err)
}

func IsTooManyRequests(err error) bool {
	return k8serrors.IsTooManyRequests(err)
}

func IsRequestEntityTooLargeError(err error) bool {
	return k8serrors.IsRequestEntityTooLargeError(err)
}

func IsUnexpectedServerError(err error) bool {
	return k8serrors.IsUnexpectedServerError(err)
}

func IsUnexpectedObjectError(err error) bool {
	return k8serrors.IsUnexpectedObjectError(err)
}

type ParentNotFoundError struct {
	errMessage string
}

func NewParentNotFound(displayName string, objectType metav1.Type) ParentNotFoundError {
	return ParentNotFoundError{
		errMessage: fmt.Sprintf("parent not found for %s: %s", objectType, displayName),
	}
}

func (p ParentNotFoundError) Error() string {
	return p.errMessage
}

func IsParentNotFound(err error) bool {
	return errors.As(err, &ParentNotFoundError{})
}

type ChildNotFoundError struct {
	errMessage string
}

func NewChildNotFound(parentDisplayName string, parentType string,
	childVarName string, childDisplayName ...string) ChildNotFoundError {
	if len(childDisplayName) == 0 {
		return ChildNotFoundError{
			errMessage: fmt.Sprintf("child %s not found for %s: %s", childVarName, parentType, parentDisplayName),
		}
	}
	return ChildNotFoundError{
		errMessage: fmt.Sprintf("child %s: %s not found for %s: %s",
			childVarName, childDisplayName[0], parentType, parentDisplayName),
	}
}

func (p ChildNotFoundError) Error() string {
	return p.errMessage
}

func IsChildNotFound(err error) bool {
	return errors.As(err, &ChildNotFoundError{})
}

type LinkNotFoundError struct {
	errMessage string
}

func NewLinkNotFound(parentDisplayName string, parentType string,
	linkVarName string, linkDisplayName ...string) LinkNotFoundError {
	if len(linkDisplayName) == 0 {
		return LinkNotFoundError{
			errMessage: fmt.Sprintf("link %s not found for %s: %s", linkVarName, parentType, parentDisplayName),
		}
	}
	return LinkNotFoundError{
		errMessage: fmt.Sprintf("link %s: %s not found for %s: %s",
			linkVarName, linkDisplayName[0], parentType, parentDisplayName),
	}
}

func (p LinkNotFoundError) Error() string {
	return p.errMessage
}

func IsLinkNotFound(err error) bool {
	return errors.As(err, &LinkNotFoundError{})
}

type SingletonNameError struct {
	errMessage string
}

func NewSingletonNameError(displayName string) SingletonNameError {
	return SingletonNameError{
		errMessage: fmt.Sprintf("wrong name of singleton object: %s, singleton can have only"+
			"'default' as a display name", displayName),
	}
}

func (p SingletonNameError) Error() string {
	return p.errMessage
}

func IsSingletonNameError(err error) bool {
	return errors.As(err, &SingletonNameError{})
}

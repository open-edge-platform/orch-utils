//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

/*
Copyright The Kubernetes Authors.

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Child) DeepCopyInto(out *Child) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Child.
func (in *Child) DeepCopy() *Child {
	if in == nil {
		return nil
	}
	out := new(Child)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Link) DeepCopyInto(out *Link) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Link.
func (in *Link) DeepCopy() *Link {
	if in == nil {
		return nil
	}
	out := new(Link)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NexusStatus) DeepCopyInto(out *NexusStatus) {
	*out = *in
	out.SyncerStatus = in.SyncerStatus
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NexusStatus.
func (in *NexusStatus) DeepCopy() *NexusStatus {
	if in == nil {
		return nil
	}
	out := new(NexusStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RuntimeFolder) DeepCopyInto(out *RuntimeFolder) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RuntimeFolder.
func (in *RuntimeFolder) DeepCopy() *RuntimeFolder {
	if in == nil {
		return nil
	}
	out := new(RuntimeFolder)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RuntimeFolder) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RuntimeFolderList) DeepCopyInto(out *RuntimeFolderList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RuntimeFolder, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RuntimeFolderList.
func (in *RuntimeFolderList) DeepCopy() *RuntimeFolderList {
	if in == nil {
		return nil
	}
	out := new(RuntimeFolderList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RuntimeFolderList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RuntimeFolderNexusStatus) DeepCopyInto(out *RuntimeFolderNexusStatus) {
	*out = *in
	out.Nexus = in.Nexus
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RuntimeFolderNexusStatus.
func (in *RuntimeFolderNexusStatus) DeepCopy() *RuntimeFolderNexusStatus {
	if in == nil {
		return nil
	}
	out := new(RuntimeFolderNexusStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RuntimeFolderSpec) DeepCopyInto(out *RuntimeFolderSpec) {
	*out = *in
	if in.ProjectsGvk != nil {
		in, out := &in.ProjectsGvk, &out.ProjectsGvk
		*out = make(map[string]Child, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RuntimeFolderSpec.
func (in *RuntimeFolderSpec) DeepCopy() *RuntimeFolderSpec {
	if in == nil {
		return nil
	}
	out := new(RuntimeFolderSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SyncerStatus) DeepCopyInto(out *SyncerStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SyncerStatus.
func (in *SyncerStatus) DeepCopy() *SyncerStatus {
	if in == nil {
		return nil
	}
	out := new(SyncerStatus)
	in.DeepCopyInto(out)
	return out
}

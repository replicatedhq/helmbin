//go:build !ignore_autogenerated

/*
Copyright 2023.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArtifactsLocation) DeepCopyInto(out *ArtifactsLocation) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArtifactsLocation.
func (in *ArtifactsLocation) DeepCopy() *ArtifactsLocation {
	if in == nil {
		return nil
	}
	out := new(ArtifactsLocation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BuiltInExtension) DeepCopyInto(out *BuiltInExtension) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BuiltInExtension.
func (in *BuiltInExtension) DeepCopy() *BuiltInExtension {
	if in == nil {
		return nil
	}
	out := new(BuiltInExtension)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Config) DeepCopyInto(out *Config) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Config.
func (in *Config) DeepCopy() *Config {
	if in == nil {
		return nil
	}
	out := new(Config)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Config) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigList) DeepCopyInto(out *ConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Config, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigList.
func (in *ConfigList) DeepCopy() *ConfigList {
	if in == nil {
		return nil
	}
	out := new(ConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigSecret) DeepCopyInto(out *ConfigSecret) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigSecret.
func (in *ConfigSecret) DeepCopy() *ConfigSecret {
	if in == nil {
		return nil
	}
	out := new(ConfigSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigSpec) DeepCopyInto(out *ConfigSpec) {
	*out = *in
	in.Roles.DeepCopyInto(&out.Roles)
	in.UnsupportedOverrides.DeepCopyInto(&out.UnsupportedOverrides)
	in.Extensions.DeepCopyInto(&out.Extensions)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigSpec.
func (in *ConfigSpec) DeepCopy() *ConfigSpec {
	if in == nil {
		return nil
	}
	out := new(ConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConfigStatus) DeepCopyInto(out *ConfigStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConfigStatus.
func (in *ConfigStatus) DeepCopy() *ConfigStatus {
	if in == nil {
		return nil
	}
	out := new(ConfigStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Extensions) DeepCopyInto(out *Extensions) {
	*out = *in
	if in.Helm != nil {
		in, out := &in.Helm, &out.Helm
		*out = new(k0sv1beta1.HelmExtensions)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Extensions.
func (in *Extensions) DeepCopy() *Extensions {
	if in == nil {
		return nil
	}
	out := new(Extensions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Installation) DeepCopyInto(out *Installation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Installation.
func (in *Installation) DeepCopy() *Installation {
	if in == nil {
		return nil
	}
	out := new(Installation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Installation) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstallationList) DeepCopyInto(out *InstallationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Installation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstallationList.
func (in *InstallationList) DeepCopy() *InstallationList {
	if in == nil {
		return nil
	}
	out := new(InstallationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InstallationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstallationSpec) DeepCopyInto(out *InstallationSpec) {
	*out = *in
	if in.Artifacts != nil {
		in, out := &in.Artifacts, &out.Artifacts
		*out = new(ArtifactsLocation)
		**out = **in
	}
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(ConfigSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.LicenseInfo != nil {
		in, out := &in.LicenseInfo, &out.LicenseInfo
		*out = new(LicenseInfo)
		**out = **in
	}
	if in.ConfigSecret != nil {
		in, out := &in.ConfigSecret, &out.ConfigSecret
		*out = new(ConfigSecret)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstallationSpec.
func (in *InstallationSpec) DeepCopy() *InstallationSpec {
	if in == nil {
		return nil
	}
	out := new(InstallationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstallationStatus) DeepCopyInto(out *InstallationStatus) {
	*out = *in
	if in.NodesStatus != nil {
		in, out := &in.NodesStatus, &out.NodesStatus
		*out = make([]NodeStatus, len(*in))
		copy(*out, *in)
	}
	if in.PendingCharts != nil {
		in, out := &in.PendingCharts, &out.PendingCharts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstallationStatus.
func (in *InstallationStatus) DeepCopy() *InstallationStatus {
	if in == nil {
		return nil
	}
	out := new(InstallationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LicenseInfo) DeepCopyInto(out *LicenseInfo) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LicenseInfo.
func (in *LicenseInfo) DeepCopy() *LicenseInfo {
	if in == nil {
		return nil
	}
	out := new(LicenseInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeCount) DeepCopyInto(out *NodeCount) {
	*out = *in
	if in.Values != nil {
		in, out := &in.Values, &out.Values
		*out = make([]int, len(*in))
		copy(*out, *in)
	}
	if in.Range != nil {
		in, out := &in.Range, &out.Range
		*out = new(NodeRange)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeCount.
func (in *NodeCount) DeepCopy() *NodeCount {
	if in == nil {
		return nil
	}
	out := new(NodeCount)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeRange) DeepCopyInto(out *NodeRange) {
	*out = *in
	if in.Min != nil {
		in, out := &in.Min, &out.Min
		*out = new(int)
		**out = **in
	}
	if in.Max != nil {
		in, out := &in.Max, &out.Max
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeRange.
func (in *NodeRange) DeepCopy() *NodeRange {
	if in == nil {
		return nil
	}
	out := new(NodeRange)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeRole) DeepCopyInto(out *NodeRole) {
	*out = *in
	if in.NodeCount != nil {
		in, out := &in.NodeCount, &out.NodeCount
		*out = new(NodeCount)
		(*in).DeepCopyInto(*out)
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeRole.
func (in *NodeRole) DeepCopy() *NodeRole {
	if in == nil {
		return nil
	}
	out := new(NodeRole)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeStatus) DeepCopyInto(out *NodeStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeStatus.
func (in *NodeStatus) DeepCopy() *NodeStatus {
	if in == nil {
		return nil
	}
	out := new(NodeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Roles) DeepCopyInto(out *Roles) {
	*out = *in
	in.Controller.DeepCopyInto(&out.Controller)
	if in.Custom != nil {
		in, out := &in.Custom, &out.Custom
		*out = make([]NodeRole, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Roles.
func (in *Roles) DeepCopy() *Roles {
	if in == nil {
		return nil
	}
	out := new(Roles)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UnsupportedOverrides) DeepCopyInto(out *UnsupportedOverrides) {
	*out = *in
	if in.BuiltInExtensions != nil {
		in, out := &in.BuiltInExtensions, &out.BuiltInExtensions
		*out = make([]BuiltInExtension, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UnsupportedOverrides.
func (in *UnsupportedOverrides) DeepCopy() *UnsupportedOverrides {
	if in == nil {
		return nil
	}
	out := new(UnsupportedOverrides)
	in.DeepCopyInto(out)
	return out
}

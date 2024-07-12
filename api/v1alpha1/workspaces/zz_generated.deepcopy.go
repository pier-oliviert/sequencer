//go:build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package workspaces

import (
	"github.com/pier-oliviert/sequencer/api/v1alpha1/conditions"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/tunneling"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DNS) DeepCopyInto(out *DNS) {
	*out = *in
	if in.Records != nil {
		in, out := &in.Records, &out.Records
		*out = make([]DNSRecord, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DNS.
func (in *DNS) DeepCopy() *DNS {
	if in == nil {
		return nil
	}
	out := new(DNS)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DNSRecord) DeepCopyInto(out *DNSRecord) {
	*out = *in
	if in.Properties != nil {
		in, out := &in.Properties, &out.Properties
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DNSRecord.
func (in *DNSRecord) DeepCopy() *DNSRecord {
	if in == nil {
		return nil
	}
	out := new(DNSRecord)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DNSSpec) DeepCopyInto(out *DNSSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DNSSpec.
func (in *DNSSpec) DeepCopy() *DNSSpec {
	if in == nil {
		return nil
	}
	out := new(DNSSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IngressSpec) DeepCopyInto(out *IngressSpec) {
	*out = *in
	if in.Rules != nil {
		in, out := &in.Rules, &out.Rules
		*out = make([]RuleSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ClassName != nil {
		in, out := &in.ClassName, &out.ClassName
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IngressSpec.
func (in *IngressSpec) DeepCopy() *IngressSpec {
	if in == nil {
		return nil
	}
	out := new(IngressSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkingSpec) DeepCopyInto(out *NetworkingSpec) {
	*out = *in
	out.DNS = in.DNS
	if in.Tunnel != nil {
		in, out := &in.Tunnel, &out.Tunnel
		*out = new(TunnelSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Ingress != nil {
		in, out := &in.Ingress, &out.Ingress
		*out = new(IngressSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkingSpec.
func (in *NetworkingSpec) DeepCopy() *NetworkingSpec {
	if in == nil {
		return nil
	}
	out := new(NetworkingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RuleSpec) DeepCopyInto(out *RuleSpec) {
	*out = *in
	if in.Subdomain != nil {
		in, out := &in.Subdomain, &out.Subdomain
		*out = new(string)
		**out = **in
	}
	if in.Path != nil {
		in, out := &in.Path, &out.Path
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RuleSpec.
func (in *RuleSpec) DeepCopy() *RuleSpec {
	if in == nil {
		return nil
	}
	out := new(RuleSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Status) DeepCopyInto(out *Status) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]conditions.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Tunnel != nil {
		in, out := &in.Tunnel, &out.Tunnel
		*out = new(Tunnel)
		(*in).DeepCopyInto(*out)
	}
	if in.DNS != nil {
		in, out := &in.DNS, &out.DNS
		*out = new(DNS)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Status.
func (in *Status) DeepCopy() *Status {
	if in == nil {
		return nil
	}
	out := new(Status)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Tunnel) DeepCopyInto(out *Tunnel) {
	*out = *in
	if in.ProviderMeta != nil {
		in, out := &in.ProviderMeta, &out.ProviderMeta
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Tunnel.
func (in *Tunnel) DeepCopy() *Tunnel {
	if in == nil {
		return nil
	}
	out := new(Tunnel)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TunnelSpec) DeepCopyInto(out *TunnelSpec) {
	*out = *in
	if in.Cloudflare != nil {
		in, out := &in.Cloudflare, &out.Cloudflare
		*out = new(tunneling.CloudflareTunnelSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TunnelSpec.
func (in *TunnelSpec) DeepCopy() *TunnelSpec {
	if in == nil {
		return nil
	}
	out := new(TunnelSpec)
	in.DeepCopyInto(out)
	return out
}

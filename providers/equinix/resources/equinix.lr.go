// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Code generated by resources. DO NOT EDIT.

package resources

import (
	"errors"
	"time"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/types"
)

var resourceFactories map[string]plugin.ResourceFactory

func init() {
	resourceFactories = map[string]plugin.ResourceFactory {
		"equinix.metal.project": {
			Init: initEquinixMetalProject,
			Create: createEquinixMetalProject,
		},
		"equinix.metal.organization": {
			Init: initEquinixMetalOrganization,
			Create: createEquinixMetalOrganization,
		},
		"equinix.metal.user": {
			// to override args, implement: initEquinixMetalUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createEquinixMetalUser,
		},
		"equinix.metal.sshkey": {
			// to override args, implement: initEquinixMetalSshkey(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createEquinixMetalSshkey,
		},
		"equinix.metal.device": {
			// to override args, implement: initEquinixMetalDevice(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createEquinixMetalDevice,
		},
	}
}

// NewResource is used by the runtime of this plugin to create new resources.
// Its arguments may be provided by users. This function is generally not
// used by initializing resources from recordings or from lists.
func NewResource(runtime *plugin.Runtime, name string, args map[string]*llx.RawData) (plugin.Resource, error) {
	f, ok := resourceFactories[name]
	if !ok {
		return nil, errors.New("cannot find resource " + name + " in this provider")
	}

	if f.Init != nil {
		cargs, res, err := f.Init(runtime, args)
		if err != nil {
			return res, err
		}

		if res != nil {
			id := name+"\x00"+res.MqlID()
			if x, ok := runtime.Resources.Get(id); ok {
				return x, nil
			}
			runtime.Resources.Set(id, res)
			return res, nil
		}

		args = cargs
	}

	res, err := f.Create(runtime, args)
	if err != nil {
		return nil, err
	}

	id := name+"\x00"+res.MqlID()
	if x, ok := runtime.Resources.Get(id); ok {
		return x, nil
	}

	runtime.Resources.Set(id, res)
	return res, nil
}

// CreateResource is used by the runtime of this plugin to create resources.
// Its arguments must be complete and pre-processed. This method is used
// for initializing resources from recordings or from lists.
func CreateResource(runtime *plugin.Runtime, name string, args map[string]*llx.RawData) (plugin.Resource, error) {
	f, ok := resourceFactories[name]
	if !ok {
		return nil, errors.New("cannot find resource " + name + " in this provider")
	}

	res, err := f.Create(runtime, args)
	if err != nil {
		return nil, err
	}

	id := name+"\x00"+res.MqlID()
	if x, ok := runtime.Resources.Get(id); ok {
		return x, nil
	}

	runtime.Resources.Set(id, res)
	return res, nil
}

var getDataFields = map[string]func(r plugin.Resource) *plugin.DataRes{
	"equinix.metal.project.id": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalProject).GetId()).ToDataRes(types.String)
	},
	"equinix.metal.project.name": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalProject).GetName()).ToDataRes(types.String)
	},
	"equinix.metal.project.organization": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalProject).GetOrganization()).ToDataRes(types.Resource("equinix.metal.organization"))
	},
	"equinix.metal.project.createdAt": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalProject).GetCreatedAt()).ToDataRes(types.Time)
	},
	"equinix.metal.project.updatedAt": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalProject).GetUpdatedAt()).ToDataRes(types.Time)
	},
	"equinix.metal.project.url": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalProject).GetUrl()).ToDataRes(types.String)
	},
	"equinix.metal.project.sshKeys": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalProject).GetSshKeys()).ToDataRes(types.Array(types.Resource("equinix.metal.sshkey")))
	},
	"equinix.metal.project.devices": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalProject).GetDevices()).ToDataRes(types.Array(types.Resource("equinix.metal.device")))
	},
	"equinix.metal.organization.id": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetId()).ToDataRes(types.String)
	},
	"equinix.metal.organization.name": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetName()).ToDataRes(types.String)
	},
	"equinix.metal.organization.description": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetDescription()).ToDataRes(types.String)
	},
	"equinix.metal.organization.website": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetWebsite()).ToDataRes(types.String)
	},
	"equinix.metal.organization.twitter": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetTwitter()).ToDataRes(types.String)
	},
	"equinix.metal.organization.createdAt": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetCreatedAt()).ToDataRes(types.Time)
	},
	"equinix.metal.organization.updatedAt": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetUpdatedAt()).ToDataRes(types.Time)
	},
	"equinix.metal.organization.address": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetAddress()).ToDataRes(types.Dict)
	},
	"equinix.metal.organization.taxId": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetTaxId()).ToDataRes(types.String)
	},
	"equinix.metal.organization.mainPhone": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetMainPhone()).ToDataRes(types.String)
	},
	"equinix.metal.organization.billingPhone": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetBillingPhone()).ToDataRes(types.String)
	},
	"equinix.metal.organization.creditAmount": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetCreditAmount()).ToDataRes(types.Float)
	},
	"equinix.metal.organization.url": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetUrl()).ToDataRes(types.String)
	},
	"equinix.metal.organization.users": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalOrganization).GetUsers()).ToDataRes(types.Array(types.Resource("equinix.metal.user")))
	},
	"equinix.metal.user.id": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetId()).ToDataRes(types.String)
	},
	"equinix.metal.user.firstName": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetFirstName()).ToDataRes(types.String)
	},
	"equinix.metal.user.lastName": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetLastName()).ToDataRes(types.String)
	},
	"equinix.metal.user.fullName": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetFullName()).ToDataRes(types.String)
	},
	"equinix.metal.user.email": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetEmail()).ToDataRes(types.String)
	},
	"equinix.metal.user.twoFactorAuth": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetTwoFactorAuth()).ToDataRes(types.String)
	},
	"equinix.metal.user.avatarUrl": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetAvatarUrl()).ToDataRes(types.String)
	},
	"equinix.metal.user.twitter": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetTwitter()).ToDataRes(types.String)
	},
	"equinix.metal.user.facebook": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetFacebook()).ToDataRes(types.String)
	},
	"equinix.metal.user.linkedin": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetLinkedin()).ToDataRes(types.String)
	},
	"equinix.metal.user.createdAt": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetCreatedAt()).ToDataRes(types.Time)
	},
	"equinix.metal.user.updatedAt": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetUpdatedAt()).ToDataRes(types.Time)
	},
	"equinix.metal.user.timezone": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetTimezone()).ToDataRes(types.String)
	},
	"equinix.metal.user.phoneNumber": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetPhoneNumber()).ToDataRes(types.String)
	},
	"equinix.metal.user.url": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalUser).GetUrl()).ToDataRes(types.String)
	},
	"equinix.metal.sshkey.id": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalSshkey).GetId()).ToDataRes(types.String)
	},
	"equinix.metal.sshkey.label": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalSshkey).GetLabel()).ToDataRes(types.String)
	},
	"equinix.metal.sshkey.key": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalSshkey).GetKey()).ToDataRes(types.String)
	},
	"equinix.metal.sshkey.fingerPrint": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalSshkey).GetFingerPrint()).ToDataRes(types.String)
	},
	"equinix.metal.sshkey.createdAt": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalSshkey).GetCreatedAt()).ToDataRes(types.Time)
	},
	"equinix.metal.sshkey.updatedAt": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalSshkey).GetUpdatedAt()).ToDataRes(types.Time)
	},
	"equinix.metal.sshkey.url": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalSshkey).GetUrl()).ToDataRes(types.String)
	},
	"equinix.metal.device.id": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalDevice).GetId()).ToDataRes(types.String)
	},
	"equinix.metal.device.shortID": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalDevice).GetShortID()).ToDataRes(types.String)
	},
	"equinix.metal.device.url": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalDevice).GetUrl()).ToDataRes(types.String)
	},
	"equinix.metal.device.hostname": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalDevice).GetHostname()).ToDataRes(types.String)
	},
	"equinix.metal.device.description": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalDevice).GetDescription()).ToDataRes(types.String)
	},
	"equinix.metal.device.state": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalDevice).GetState()).ToDataRes(types.String)
	},
	"equinix.metal.device.createdAt": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalDevice).GetCreatedAt()).ToDataRes(types.Time)
	},
	"equinix.metal.device.updatedAt": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalDevice).GetUpdatedAt()).ToDataRes(types.Time)
	},
	"equinix.metal.device.locked": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalDevice).GetLocked()).ToDataRes(types.Bool)
	},
	"equinix.metal.device.billingCycle": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalDevice).GetBillingCycle()).ToDataRes(types.String)
	},
	"equinix.metal.device.spotInstance": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalDevice).GetSpotInstance()).ToDataRes(types.Bool)
	},
	"equinix.metal.device.os": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlEquinixMetalDevice).GetOs()).ToDataRes(types.Dict)
	},
}

func GetData(resource plugin.Resource, field string, args map[string]*llx.RawData) *plugin.DataRes {
	f, ok := getDataFields[resource.MqlName()+"."+field]
	if !ok {
		return &plugin.DataRes{Error: "cannot find '" + field + "' in resource '" + resource.MqlName() + "'"}
	}

	return f(resource)
}

var setDataFields = map[string]func(r plugin.Resource, v *llx.RawData) bool {
	"equinix.metal.project.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlEquinixMetalProject).__id, ok = v.Value.(string)
			return
		},
	"equinix.metal.project.id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalProject).Id, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.project.name": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalProject).Name, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.project.organization": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalProject).Organization, ok = plugin.RawToTValue[*mqlEquinixMetalOrganization](v.Value, v.Error)
		return
	},
	"equinix.metal.project.createdAt": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalProject).CreatedAt, ok = plugin.RawToTValue[*time.Time](v.Value, v.Error)
		return
	},
	"equinix.metal.project.updatedAt": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalProject).UpdatedAt, ok = plugin.RawToTValue[*time.Time](v.Value, v.Error)
		return
	},
	"equinix.metal.project.url": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalProject).Url, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.project.sshKeys": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalProject).SshKeys, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"equinix.metal.project.devices": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalProject).Devices, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlEquinixMetalOrganization).__id, ok = v.Value.(string)
			return
		},
	"equinix.metal.organization.id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).Id, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.name": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).Name, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.description": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).Description, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.website": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).Website, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.twitter": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).Twitter, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.createdAt": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).CreatedAt, ok = plugin.RawToTValue[*time.Time](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.updatedAt": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).UpdatedAt, ok = plugin.RawToTValue[*time.Time](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.address": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).Address, ok = plugin.RawToTValue[interface{}](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.taxId": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).TaxId, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.mainPhone": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).MainPhone, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.billingPhone": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).BillingPhone, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.creditAmount": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).CreditAmount, ok = plugin.RawToTValue[float64](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.url": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).Url, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.organization.users": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalOrganization).Users, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"equinix.metal.user.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlEquinixMetalUser).__id, ok = v.Value.(string)
			return
		},
	"equinix.metal.user.id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).Id, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.user.firstName": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).FirstName, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.user.lastName": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).LastName, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.user.fullName": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).FullName, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.user.email": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).Email, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.user.twoFactorAuth": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).TwoFactorAuth, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.user.avatarUrl": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).AvatarUrl, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.user.twitter": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).Twitter, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.user.facebook": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).Facebook, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.user.linkedin": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).Linkedin, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.user.createdAt": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).CreatedAt, ok = plugin.RawToTValue[*time.Time](v.Value, v.Error)
		return
	},
	"equinix.metal.user.updatedAt": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).UpdatedAt, ok = plugin.RawToTValue[*time.Time](v.Value, v.Error)
		return
	},
	"equinix.metal.user.timezone": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).Timezone, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.user.phoneNumber": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).PhoneNumber, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.user.url": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalUser).Url, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.sshkey.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlEquinixMetalSshkey).__id, ok = v.Value.(string)
			return
		},
	"equinix.metal.sshkey.id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalSshkey).Id, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.sshkey.label": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalSshkey).Label, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.sshkey.key": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalSshkey).Key, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.sshkey.fingerPrint": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalSshkey).FingerPrint, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.sshkey.createdAt": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalSshkey).CreatedAt, ok = plugin.RawToTValue[*time.Time](v.Value, v.Error)
		return
	},
	"equinix.metal.sshkey.updatedAt": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalSshkey).UpdatedAt, ok = plugin.RawToTValue[*time.Time](v.Value, v.Error)
		return
	},
	"equinix.metal.sshkey.url": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalSshkey).Url, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.device.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlEquinixMetalDevice).__id, ok = v.Value.(string)
			return
		},
	"equinix.metal.device.id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalDevice).Id, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.device.shortID": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalDevice).ShortID, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.device.url": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalDevice).Url, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.device.hostname": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalDevice).Hostname, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.device.description": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalDevice).Description, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.device.state": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalDevice).State, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.device.createdAt": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalDevice).CreatedAt, ok = plugin.RawToTValue[*time.Time](v.Value, v.Error)
		return
	},
	"equinix.metal.device.updatedAt": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalDevice).UpdatedAt, ok = plugin.RawToTValue[*time.Time](v.Value, v.Error)
		return
	},
	"equinix.metal.device.locked": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalDevice).Locked, ok = plugin.RawToTValue[bool](v.Value, v.Error)
		return
	},
	"equinix.metal.device.billingCycle": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalDevice).BillingCycle, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"equinix.metal.device.spotInstance": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalDevice).SpotInstance, ok = plugin.RawToTValue[bool](v.Value, v.Error)
		return
	},
	"equinix.metal.device.os": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlEquinixMetalDevice).Os, ok = plugin.RawToTValue[interface{}](v.Value, v.Error)
		return
	},
}

func SetData(resource plugin.Resource, field string, val *llx.RawData) error {
	f, ok := setDataFields[resource.MqlName() + "." + field]
	if !ok {
		return errors.New("[equinix] cannot set '"+field+"' in resource '"+resource.MqlName()+"', field not found")
	}

	if ok := f(resource, val); !ok {
		return errors.New("[equinix] cannot set '"+field+"' in resource '"+resource.MqlName()+"', type does not match")
	}
	return nil
}

func SetAllData(resource plugin.Resource, args map[string]*llx.RawData) error {
	var err error
	for k, v := range args {
		if err = SetData(resource, k, v); err != nil {
			return err
		}
	}
	return nil
}

// mqlEquinixMetalProject for the equinix.metal.project resource
type mqlEquinixMetalProject struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlEquinixMetalProjectInternal it will be used here
	Id plugin.TValue[string]
	Name plugin.TValue[string]
	Organization plugin.TValue[*mqlEquinixMetalOrganization]
	CreatedAt plugin.TValue[*time.Time]
	UpdatedAt plugin.TValue[*time.Time]
	Url plugin.TValue[string]
	SshKeys plugin.TValue[[]interface{}]
	Devices plugin.TValue[[]interface{}]
}

// createEquinixMetalProject creates a new instance of this resource
func createEquinixMetalProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlEquinixMetalProject{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("equinix.metal.project", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlEquinixMetalProject) MqlName() string {
	return "equinix.metal.project"
}

func (c *mqlEquinixMetalProject) MqlID() string {
	return c.__id
}

func (c *mqlEquinixMetalProject) GetId() *plugin.TValue[string] {
	return &c.Id
}

func (c *mqlEquinixMetalProject) GetName() *plugin.TValue[string] {
	return &c.Name
}

func (c *mqlEquinixMetalProject) GetOrganization() *plugin.TValue[*mqlEquinixMetalOrganization] {
	return plugin.GetOrCompute[*mqlEquinixMetalOrganization](&c.Organization, func() (*mqlEquinixMetalOrganization, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("equinix.metal.project", c.__id, "organization")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.(*mqlEquinixMetalOrganization), nil
			}
		}

		return c.organization()
	})
}

func (c *mqlEquinixMetalProject) GetCreatedAt() *plugin.TValue[*time.Time] {
	return &c.CreatedAt
}

func (c *mqlEquinixMetalProject) GetUpdatedAt() *plugin.TValue[*time.Time] {
	return &c.UpdatedAt
}

func (c *mqlEquinixMetalProject) GetUrl() *plugin.TValue[string] {
	return &c.Url
}

func (c *mqlEquinixMetalProject) GetSshKeys() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.SshKeys, func() ([]interface{}, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("equinix.metal.project", c.__id, "sshKeys")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.([]interface{}), nil
			}
		}

		return c.sshKeys()
	})
}

func (c *mqlEquinixMetalProject) GetDevices() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.Devices, func() ([]interface{}, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("equinix.metal.project", c.__id, "devices")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.([]interface{}), nil
			}
		}

		return c.devices()
	})
}

// mqlEquinixMetalOrganization for the equinix.metal.organization resource
type mqlEquinixMetalOrganization struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlEquinixMetalOrganizationInternal it will be used here
	Id plugin.TValue[string]
	Name plugin.TValue[string]
	Description plugin.TValue[string]
	Website plugin.TValue[string]
	Twitter plugin.TValue[string]
	CreatedAt plugin.TValue[*time.Time]
	UpdatedAt plugin.TValue[*time.Time]
	Address plugin.TValue[interface{}]
	TaxId plugin.TValue[string]
	MainPhone plugin.TValue[string]
	BillingPhone plugin.TValue[string]
	CreditAmount plugin.TValue[float64]
	Url plugin.TValue[string]
	Users plugin.TValue[[]interface{}]
}

// createEquinixMetalOrganization creates a new instance of this resource
func createEquinixMetalOrganization(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlEquinixMetalOrganization{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("equinix.metal.organization", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlEquinixMetalOrganization) MqlName() string {
	return "equinix.metal.organization"
}

func (c *mqlEquinixMetalOrganization) MqlID() string {
	return c.__id
}

func (c *mqlEquinixMetalOrganization) GetId() *plugin.TValue[string] {
	return &c.Id
}

func (c *mqlEquinixMetalOrganization) GetName() *plugin.TValue[string] {
	return &c.Name
}

func (c *mqlEquinixMetalOrganization) GetDescription() *plugin.TValue[string] {
	return &c.Description
}

func (c *mqlEquinixMetalOrganization) GetWebsite() *plugin.TValue[string] {
	return &c.Website
}

func (c *mqlEquinixMetalOrganization) GetTwitter() *plugin.TValue[string] {
	return &c.Twitter
}

func (c *mqlEquinixMetalOrganization) GetCreatedAt() *plugin.TValue[*time.Time] {
	return &c.CreatedAt
}

func (c *mqlEquinixMetalOrganization) GetUpdatedAt() *plugin.TValue[*time.Time] {
	return &c.UpdatedAt
}

func (c *mqlEquinixMetalOrganization) GetAddress() *plugin.TValue[interface{}] {
	return &c.Address
}

func (c *mqlEquinixMetalOrganization) GetTaxId() *plugin.TValue[string] {
	return &c.TaxId
}

func (c *mqlEquinixMetalOrganization) GetMainPhone() *plugin.TValue[string] {
	return &c.MainPhone
}

func (c *mqlEquinixMetalOrganization) GetBillingPhone() *plugin.TValue[string] {
	return &c.BillingPhone
}

func (c *mqlEquinixMetalOrganization) GetCreditAmount() *plugin.TValue[float64] {
	return &c.CreditAmount
}

func (c *mqlEquinixMetalOrganization) GetUrl() *plugin.TValue[string] {
	return &c.Url
}

func (c *mqlEquinixMetalOrganization) GetUsers() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.Users, func() ([]interface{}, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("equinix.metal.organization", c.__id, "users")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.([]interface{}), nil
			}
		}

		return c.users()
	})
}

// mqlEquinixMetalUser for the equinix.metal.user resource
type mqlEquinixMetalUser struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlEquinixMetalUserInternal it will be used here
	Id plugin.TValue[string]
	FirstName plugin.TValue[string]
	LastName plugin.TValue[string]
	FullName plugin.TValue[string]
	Email plugin.TValue[string]
	TwoFactorAuth plugin.TValue[string]
	AvatarUrl plugin.TValue[string]
	Twitter plugin.TValue[string]
	Facebook plugin.TValue[string]
	Linkedin plugin.TValue[string]
	CreatedAt plugin.TValue[*time.Time]
	UpdatedAt plugin.TValue[*time.Time]
	Timezone plugin.TValue[string]
	PhoneNumber plugin.TValue[string]
	Url plugin.TValue[string]
}

// createEquinixMetalUser creates a new instance of this resource
func createEquinixMetalUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlEquinixMetalUser{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("equinix.metal.user", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlEquinixMetalUser) MqlName() string {
	return "equinix.metal.user"
}

func (c *mqlEquinixMetalUser) MqlID() string {
	return c.__id
}

func (c *mqlEquinixMetalUser) GetId() *plugin.TValue[string] {
	return &c.Id
}

func (c *mqlEquinixMetalUser) GetFirstName() *plugin.TValue[string] {
	return &c.FirstName
}

func (c *mqlEquinixMetalUser) GetLastName() *plugin.TValue[string] {
	return &c.LastName
}

func (c *mqlEquinixMetalUser) GetFullName() *plugin.TValue[string] {
	return &c.FullName
}

func (c *mqlEquinixMetalUser) GetEmail() *plugin.TValue[string] {
	return &c.Email
}

func (c *mqlEquinixMetalUser) GetTwoFactorAuth() *plugin.TValue[string] {
	return &c.TwoFactorAuth
}

func (c *mqlEquinixMetalUser) GetAvatarUrl() *plugin.TValue[string] {
	return &c.AvatarUrl
}

func (c *mqlEquinixMetalUser) GetTwitter() *plugin.TValue[string] {
	return &c.Twitter
}

func (c *mqlEquinixMetalUser) GetFacebook() *plugin.TValue[string] {
	return &c.Facebook
}

func (c *mqlEquinixMetalUser) GetLinkedin() *plugin.TValue[string] {
	return &c.Linkedin
}

func (c *mqlEquinixMetalUser) GetCreatedAt() *plugin.TValue[*time.Time] {
	return &c.CreatedAt
}

func (c *mqlEquinixMetalUser) GetUpdatedAt() *plugin.TValue[*time.Time] {
	return &c.UpdatedAt
}

func (c *mqlEquinixMetalUser) GetTimezone() *plugin.TValue[string] {
	return &c.Timezone
}

func (c *mqlEquinixMetalUser) GetPhoneNumber() *plugin.TValue[string] {
	return &c.PhoneNumber
}

func (c *mqlEquinixMetalUser) GetUrl() *plugin.TValue[string] {
	return &c.Url
}

// mqlEquinixMetalSshkey for the equinix.metal.sshkey resource
type mqlEquinixMetalSshkey struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlEquinixMetalSshkeyInternal it will be used here
	Id plugin.TValue[string]
	Label plugin.TValue[string]
	Key plugin.TValue[string]
	FingerPrint plugin.TValue[string]
	CreatedAt plugin.TValue[*time.Time]
	UpdatedAt plugin.TValue[*time.Time]
	Url plugin.TValue[string]
}

// createEquinixMetalSshkey creates a new instance of this resource
func createEquinixMetalSshkey(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlEquinixMetalSshkey{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("equinix.metal.sshkey", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlEquinixMetalSshkey) MqlName() string {
	return "equinix.metal.sshkey"
}

func (c *mqlEquinixMetalSshkey) MqlID() string {
	return c.__id
}

func (c *mqlEquinixMetalSshkey) GetId() *plugin.TValue[string] {
	return &c.Id
}

func (c *mqlEquinixMetalSshkey) GetLabel() *plugin.TValue[string] {
	return &c.Label
}

func (c *mqlEquinixMetalSshkey) GetKey() *plugin.TValue[string] {
	return &c.Key
}

func (c *mqlEquinixMetalSshkey) GetFingerPrint() *plugin.TValue[string] {
	return &c.FingerPrint
}

func (c *mqlEquinixMetalSshkey) GetCreatedAt() *plugin.TValue[*time.Time] {
	return &c.CreatedAt
}

func (c *mqlEquinixMetalSshkey) GetUpdatedAt() *plugin.TValue[*time.Time] {
	return &c.UpdatedAt
}

func (c *mqlEquinixMetalSshkey) GetUrl() *plugin.TValue[string] {
	return &c.Url
}

// mqlEquinixMetalDevice for the equinix.metal.device resource
type mqlEquinixMetalDevice struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlEquinixMetalDeviceInternal it will be used here
	Id plugin.TValue[string]
	ShortID plugin.TValue[string]
	Url plugin.TValue[string]
	Hostname plugin.TValue[string]
	Description plugin.TValue[string]
	State plugin.TValue[string]
	CreatedAt plugin.TValue[*time.Time]
	UpdatedAt plugin.TValue[*time.Time]
	Locked plugin.TValue[bool]
	BillingCycle plugin.TValue[string]
	SpotInstance plugin.TValue[bool]
	Os plugin.TValue[interface{}]
}

// createEquinixMetalDevice creates a new instance of this resource
func createEquinixMetalDevice(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlEquinixMetalDevice{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("equinix.metal.device", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlEquinixMetalDevice) MqlName() string {
	return "equinix.metal.device"
}

func (c *mqlEquinixMetalDevice) MqlID() string {
	return c.__id
}

func (c *mqlEquinixMetalDevice) GetId() *plugin.TValue[string] {
	return &c.Id
}

func (c *mqlEquinixMetalDevice) GetShortID() *plugin.TValue[string] {
	return &c.ShortID
}

func (c *mqlEquinixMetalDevice) GetUrl() *plugin.TValue[string] {
	return &c.Url
}

func (c *mqlEquinixMetalDevice) GetHostname() *plugin.TValue[string] {
	return &c.Hostname
}

func (c *mqlEquinixMetalDevice) GetDescription() *plugin.TValue[string] {
	return &c.Description
}

func (c *mqlEquinixMetalDevice) GetState() *plugin.TValue[string] {
	return &c.State
}

func (c *mqlEquinixMetalDevice) GetCreatedAt() *plugin.TValue[*time.Time] {
	return &c.CreatedAt
}

func (c *mqlEquinixMetalDevice) GetUpdatedAt() *plugin.TValue[*time.Time] {
	return &c.UpdatedAt
}

func (c *mqlEquinixMetalDevice) GetLocked() *plugin.TValue[bool] {
	return &c.Locked
}

func (c *mqlEquinixMetalDevice) GetBillingCycle() *plugin.TValue[string] {
	return &c.BillingCycle
}

func (c *mqlEquinixMetalDevice) GetSpotInstance() *plugin.TValue[bool] {
	return &c.SpotInstance
}

func (c *mqlEquinixMetalDevice) GetOs() *plugin.TValue[interface{}] {
	return &c.Os
}

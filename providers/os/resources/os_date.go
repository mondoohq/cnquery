// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/date"
)

type mqlOsDateInternal struct {
	lock    sync.Mutex
	fetched bool
	result  *date.Result
}

func (p *mqlOs) date() (*mqlOsDate, error) {
	o, err := CreateResource(p.MqlRuntime, "os.date", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return o.(*mqlOsDate), nil
}

func (d *mqlOsDate) id() (string, error) {
	return "os.date", nil
}

func (d *mqlOsDate) fetch() (*date.Result, error) {
	if d.fetched {
		return d.result, nil
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.fetched {
		return d.result, nil
	}

	conn := d.MqlRuntime.Connection.(shared.Connection)
	dt, err := date.New(conn)
	if err != nil {
		return nil, err
	}

	res, err := dt.Get()
	if err != nil {
		return nil, err
	}

	d.fetched = true
	d.result = res
	return res, nil
}

func (d *mqlOsDate) time() (*time.Time, error) {
	res, err := d.fetch()
	if err != nil {
		return nil, err
	}
	if res.Time == nil {
		d.Time.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return res.Time, nil
}

func (d *mqlOsDate) timezone() (string, error) {
	res, err := d.fetch()
	if err != nil {
		return "", err
	}
	return res.Timezone, nil
}

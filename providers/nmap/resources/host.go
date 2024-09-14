package resources

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Ullaakut/nmap/v3"
	"github.com/google/uuid"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/nmap/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func initNmapHost(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["target"]; !ok {
		// try to get the ip from the connection
		conn := runtime.Connection.(*connection.NmapConnection)
		if conn.Conf.Options != nil && conn.Conf.Options["search"] == "host" {
			args["target"] = llx.StringData(conn.Conf.Host)
		}
	}

	if _, ok := args["target"]; !ok {
		return nil, nil, errors.New("missing required argument 'host'")
	}

	return args, nil, nil
}

func newMqlNmapHost(runtime *plugin.Runtime, host nmap.Host) (*mqlNmapHost, error) {
	distance, _ := convert.JsonToDict(host.Distance)
	os, _ := convert.JsonToDict(host.OS)
	trace, _ := convert.JsonToDict(host.Trace)
	addresses, _ := convert.JsonToDictSlice(host.Addresses)
	hostnames, _ := convert.JsonToDictSlice(host.Hostnames)

	// TODO: consider using the host IP from nmap since it is more reliable
	id := uuid.New().String()

	ports := make([]interface{}, 0)
	for _, port := range host.Ports {
		r, err := newMqlNmapPort(runtime, id, port)
		if err != nil {
			return nil, err
		}
		ports = append(ports, r)
	}

	name := ""
	if len(host.Addresses) == 1 {
		name = host.Addresses[0].Addr
	} else {
		entries := []string{}
		for _, addr := range host.Addresses {
			if addr.Addr != "" {
				entries = append(entries, addr.Addr)
			}
		}
		name = strings.Join(entries, ", ")
	}

	mqlNmapHostResource, err := CreateResource(runtime, "nmap.host", map[string]*llx.RawData{
		"__id":      llx.StringData("nmap.host/" + id),
		"name":      llx.StringData(name),
		"distance":  llx.DictData(distance),
		"os":        llx.DictData(os),
		"endTime":   llx.TimeData(time.Time(host.EndTime)),
		"comment":   llx.StringData(host.Comment),
		"trace":     llx.DictData(trace),
		"addresses": llx.ArrayData(addresses, types.Dict),
		"hostnames": llx.ArrayData(hostnames, types.Dict),
		"ports":     llx.ArrayData(ports, types.Resource("nmap.port")),
		"state":     llx.StringData(host.Status.State),
	})
	return mqlNmapHostResource.(*mqlNmapHost), err
}

func newMqlNmapPort(runtime *plugin.Runtime, id string, port nmap.Port) (*mqlNmapPort, error) {
	mqlPort, err := CreateResource(runtime, "nmap.port", map[string]*llx.RawData{
		"__id":     llx.StringData("nmap.port/" + id + "/" + strconv.Itoa(int(port.ID))),
		"port":     llx.IntData(int64(port.ID)),
		"service":  llx.StringData(port.Service.Name),
		"method":   llx.StringData(port.Service.Method),
		"protocol": llx.StringData(port.Protocol),
		"product":  llx.StringData(port.Service.Product),
		"version":  llx.StringData(port.Service.Version),
		"state":    llx.StringData(port.State.State),
	})
	return mqlPort.(*mqlNmapPort), err
}

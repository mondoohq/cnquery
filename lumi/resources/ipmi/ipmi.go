package ipmi

// Specifications
// https://www.intel.com/content/www/us/en/servers/ipmi/ipmi-technical-resources.html
//
// Commonly used commands
// https://www.thomas-krenn.com/de/tkmag/wp-content/uploads/2017/08/ipmitool-cheat-sheet-v1.0.pdf
// https://community.pivotal.io/s/article/How-to-work-on-IPMI-and-IPMITOOL?language=en_US

import (
	"errors"
	"fmt"

	ipmiTransport "github.com/vmware/goipmi"
)

// Connection properties for a Client
type Connection struct {
	Path      string
	Hostname  string
	Port      int
	Username  string
	Password  string
	Interface string
}

type IpmiClient struct {
	*Connection
	*ipmiTransport.Client
}

// NewIpmiClient is a high-level api to acces the data of a Ipmi instance
func NewIpmiClient(c *Connection) (*IpmiClient, error) {
	if c == nil {
		return nil, errors.New("no connection details provided")
	}

	tc := &ipmiTransport.Connection{
		Hostname:  c.Hostname,
		Path:      c.Path,
		Port:      c.Port,
		Username:  c.Username,
		Password:  c.Password,
		Interface: c.Interface,
	}

	t, err := ipmiTransport.NewClient(tc)
	if err != nil {
		return nil, err
	}

	return &IpmiClient{
		Connection: c,
		Client:     t,
	}, nil

}

type deviceIDReq struct{}

// 20.1Get Device IDCommand
type deviceIDResp struct {
	ipmiTransport.CompletionCode
	DeviceID                      uint8
	DeviceRevision                uint8
	FirmwareRevision1             uint8 // Major Firmware Revision, binary encoded
	FirmwareRevision2             uint8 // Minor Firmware Revision. BCD encoded.
	IPMIVersion                   uint8
	AdditionalDeviceSupport       uint8
	ManufacturerID1               uint16 // the number is 20 bit
	ManufacturerID2               uint8
	ProductID                     uint16
	AuxiliaryFirmwareInformation1 uint8 // optional Auxiliary Firmware Revision Information
	AuxiliaryFirmwareInformation2 uint8
	AuxiliaryFirmwareInformation3 uint8
	AuxiliaryFirmwareInformation4 uint8
}

type DeviceID struct {
	DeviceID                int64                   `json:"deviceID"`
	DeviceRevision          int64                   `json:"deviceRevision"`
	ProvidesDeviceSDRs      bool                    `json:"providesDeviceSDRs"`
	DeviceAvailable         bool                    `json:"deviceAvailable"`
	FirmwareRevision        string                  `json:"firmwareRevision"`
	IpmiVersion             int64                   `json:"ipmiVersion"`
	ManufacturerID          int64                   `json:"manufacturerID"`
	ManufacturerName        string                  `json:"manufacturerName"`
	ProductID               int64                   `json:"productID"`
	ProductName             string                  `json:"productName"`
	AdditionalDeviceSupport AdditionalDeviceSupport `json:"additionalDeviceSupport"`
}

type AdditionalDeviceSupport struct {
	SensorDevice        bool `json:"sensorDevice"`
	SDRRepositoryDevice bool `json:"sdrRepositoryDevice"`
	SELDevice           bool `json:"selDevice"`
	FRUInventoryDevice  bool `json:"fruInventoryDevice"`
	IPMBEventReceiver   bool `json:"ipmbEventReciever"`
	IPMBEventGenerator  bool `json:"ipmbEventGenerator"`
	Bridge              bool `json:"bridge"`
	ChassisDevice       bool `json:"chassisDevice"`
}

const (
	SensorDevice        uint8 = 0x1
	SDRRepositoryDevice uint8 = 0x2
	SELDevice           uint8 = 0x4
	FRUInventoryDevice  uint8 = 0x8
	IPMBEventReceiver   uint8 = 0x10
	IPMBEventGenerator  uint8 = 0x20
	Bridge              uint8 = 0x40
	ChassisDevice       uint8 = 0x80
)

// DeviceID get the Device ID of the BMC
func (c *IpmiClient) DeviceID() (*DeviceID, error) {
	req := &ipmiTransport.Request{
		ipmiTransport.NetworkFunctionApp,
		ipmiTransport.CommandGetDeviceID,
		&deviceIDReq{},
	}
	res := &deviceIDResp{}
	err := c.Client.Send(req, res)
	if err != nil {
		return nil, err
	}

	// 20 bits of are used for the vendor id
	manufactor := OemVendorID(uint32(res.ManufacturerID1) + uint32(res.ManufacturerID2&0x3<<8))
	product := OemProductID(res.ProductID)

	return &DeviceID{
		DeviceID:           int64(res.DeviceID),
		DeviceRevision:     int64(res.DeviceRevision & 0x07), // only the last 3 bits 00000111
		ProvidesDeviceSDRs: res.DeviceRevision&0x80 != 0,
		DeviceAvailable:    res.FirmwareRevision1&0x80 == 0, // 0 - normal operation, 1 - firmware
		FirmwareRevision:   fmt.Sprintf("%d.%02d", res.FirmwareRevision1&0x7F, res.FirmwareRevision2),
		IpmiVersion:        int64(res.IPMIVersion),
		ManufacturerID:     int64(manufactor),
		ManufacturerName:   manufactor.String(),
		ProductID:          int64(product),
		ProductName:        product.String(),
		AdditionalDeviceSupport: AdditionalDeviceSupport{
			SensorDevice:        res.AdditionalDeviceSupport&SensorDevice != 0,
			SDRRepositoryDevice: res.AdditionalDeviceSupport&SDRRepositoryDevice != 0,
			SELDevice:           res.AdditionalDeviceSupport&SELDevice != 0,
			FRUInventoryDevice:  res.AdditionalDeviceSupport&FRUInventoryDevice != 0,
			IPMBEventReceiver:   res.AdditionalDeviceSupport&IPMBEventReceiver != 0,
			IPMBEventGenerator:  res.AdditionalDeviceSupport&IPMBEventGenerator != 0,
			Bridge:              res.AdditionalDeviceSupport&Bridge != 0,
			ChassisDevice:       res.AdditionalDeviceSupport&ChassisDevice != 0,
		},
	}, nil
}

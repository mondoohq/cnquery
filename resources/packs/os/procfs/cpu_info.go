package procfs

import (
	"bufio"
	"io"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type CpuInfo struct {
	Processors []Processor `json:"processors"`
}

type Processor struct {
	Id              uint     `json:"id"`
	VendorID        string   `json:"vendorID"`
	CPUFamily       string   `json:"cpuFamily"`
	Model           string   `json:"model"`
	ModelName       string   `json:"modelName"`
	Stepping        string   `json:"stepping"`
	Microcode       string   `json:"microcode"`
	CPUMHz          float64  `json:"cpuMhz"`
	CacheSize       int64    `json:"cacheSize"`
	PhysicalID      uint     `json:"physicalID"`
	Siblings        uint     `json:"siblings"`
	CoreID          uint     `json:"coreId"`
	CPUCores        uint     `json:"cpuCores"`
	ApicID          string   `json:"apicID"`
	InitialApicID   string   `json:"initialApicID"`
	FPU             string   `json:"fpu"`
	FPUException    string   `json:"fpuException"`
	CpuIDLevel      uint     `json:"cpuIdLevel"`
	WP              string   `json:"wp"`
	Flags           []string `json:"flags"`
	Bugs            []string `json:"bugs"`
	BogoMips        float64  `json:"bogoMips"`
	CLFlushSize     uint     `json:"clflushSize"`
	CacheAlignment  uint     `json:"cacheAlignment"`
	AddressSizes    string   `json:"addressSizes"`
	PowerManagement string   `json:"powerManagement"`
}

// different cpu platforms return a different list, problem is also that if you use an arm container on qemu,
// proc fs will still return the information from x64. To make those cases work, we use the same parser for all
// different cpu architectures
func ParseCpuInfo(r io.Reader) (*CpuInfo, error) {
	scanner := bufio.NewScanner(r)

	cpuinfo := CpuInfo{
		Processors: []Processor{},
	}
	i := -1 // first processor will start with index 0 then

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		if !strings.Contains(line, ":") {
			log.Warn().Str("line", line).Msg("invalid cpuinfo line")
			continue
		}

		// split the line to get the key value pair
		entry := strings.SplitN(line, ": ", 2)

		switch strings.TrimSpace(entry[0]) {
		case "processor":
			// start a next processor and append the new info to the full list
			cpuinfo.Processors = append(cpuinfo.Processors, Processor{})
			i++
			v, err := strconv.ParseUint(entry[1], 0, 32)
			if err != nil {
				return nil, err
			}
			cpuinfo.Processors[i].Id = uint(v)
		case "vendor", "vendor_id":
			cpuinfo.Processors[i].VendorID = entry[1]
		case "cpu family":
			cpuinfo.Processors[i].CPUFamily = entry[1]
		case "model":
			cpuinfo.Processors[i].Model = entry[1]
		case "model name":
			cpuinfo.Processors[i].ModelName = entry[1]
		case "stepping":
			cpuinfo.Processors[i].Stepping = entry[1]
		case "microcode":
			cpuinfo.Processors[i].Microcode = entry[1]
		case "cpu MHz":
			v, err := strconv.ParseFloat(entry[1], 64)
			if err != nil {
				return nil, err
			}
			cpuinfo.Processors[i].CPUMHz = v
		case "cache size":
			raw := strings.TrimSpace(entry[1])
			value := strings.Split(raw, " ")

			v, err := strconv.ParseInt(value[0], 10, 64)
			if err != nil {
				return nil, err
			}
			// we expect cache size to be in kb
			if strings.HasSuffix(strings.ToLower(value[1]), "mb") {
				v = v * 1024
			}
			cpuinfo.Processors[i].CacheSize = v
		case "physical id":
			v, err := strconv.ParseUint(entry[1], 0, 32)
			if err != nil {
				return nil, err
			}
			cpuinfo.Processors[i].PhysicalID = uint(v)
		case "siblings":
			v, err := strconv.ParseUint(entry[1], 0, 32)
			if err != nil {
				return nil, err
			}
			cpuinfo.Processors[i].Siblings = uint(v)
		case "core id":
			v, err := strconv.ParseUint(entry[1], 0, 32)
			if err != nil {
				return nil, err
			}
			cpuinfo.Processors[i].CoreID = uint(v)
		case "cpu cores":
			v, err := strconv.ParseUint(entry[1], 0, 32)
			if err != nil {
				return nil, err
			}
			cpuinfo.Processors[i].CPUCores = uint(v)
		case "apicid":
			cpuinfo.Processors[i].ApicID = entry[1]
		case "initial apicid":
			cpuinfo.Processors[i].InitialApicID = entry[1]
		case "fpu":
			cpuinfo.Processors[i].FPU = entry[1]
		case "fpu_exception":
			cpuinfo.Processors[i].FPUException = entry[1]
		case "cpuid level":
			v, err := strconv.ParseUint(entry[1], 0, 32)
			if err != nil {
				return nil, err
			}
			cpuinfo.Processors[i].CpuIDLevel = uint(v)
		case "wp":
			cpuinfo.Processors[i].WP = entry[1]
		case "flags", "Features": // arm flags
			cpuinfo.Processors[i].Flags = strings.Fields(entry[1])
		case "bugs":
			cpuinfo.Processors[i].Bugs = strings.Fields(entry[1])
		case "bogomips", "BogoMIPS": // x64 & arm
			v, err := strconv.ParseFloat(entry[1], 64)
			if err != nil {
				return nil, err
			}
			cpuinfo.Processors[i].BogoMips = v
		case "clflush size":
			v, err := strconv.ParseUint(entry[1], 0, 32)
			if err != nil {
				return nil, err
			}
			cpuinfo.Processors[i].CLFlushSize = uint(v)
		case "cache_alignment":
			v, err := strconv.ParseUint(entry[1], 0, 32)
			if err != nil {
				return nil, err
			}
			cpuinfo.Processors[i].CacheAlignment = uint(v)
		case "address sizes":
			cpuinfo.Processors[i].AddressSizes = entry[1]
		case "power management":
			cpuinfo.Processors[i].PowerManagement = entry[1]
		}
	}

	return &cpuinfo, nil
}

package plugin

type Provider struct {
	Name       string
	Connectors []Connector
}

type Connector struct {
	Name    string
	Use     string   `json:",omitempty"`
	Short   string   `json:",omitempty"`
	Long    string   `json:",omitempty"`
	MinArgs uint     `json:",omitempty"`
	MaxArgs uint     `json:",omitempty"`
	Flags   []Flag   `json:",omitempty"`
	Aliases []string `json:",omitempty"`
}

type FlagType byte

const (
	FlagType_Bool FlagType = 1 + iota
	FlagType_Int
	FlagType_String
	FlagType_List
	FlagType_KeyValue
)

type FlagOption byte

const (
	FlagOption_Hidden FlagOption = 0x1 << iota
	FlagOption_Deprecated
	FlagOption_Required
	FlagOption_Password
	// max: 8 options!
)

type Flag struct {
	Long    string      `json:",omitempty"`
	Short   string      `json:",omitempty"`
	Default interface{} `json:",omitempty"`
	Desc    string      `json:",omitempty"`
	Type    FlagType    `json:",omitempty"`
	Option  FlagOption  `json:",omitempty"`
}

func Start(args []string) {
	if len(args) != 0 {
		switch args[0] {
		case "generate":
		}
	}
}

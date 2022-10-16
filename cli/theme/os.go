package theme

const logo = "  ___ _ __   __ _ _   _  ___ _ __ _   _ \n" +
	" / __| '_ \\ / _` | | | |/ _ \\ '__| | | |\n" +
	"| (__| | | | (_| | |_| |  __/ |  | |_| |\n" +
	" \\___|_| |_|\\__, |\\__,_|\\___|_|   \\__, |\n" +
	"  mondoo™      |_|                |___/ "

var DefaultTheme = OperatingSystemTheme

func init() {
	DefaultTheme.PolicyPrinter.Error = DefaultTheme.Error
	DefaultTheme.PolicyPrinter.Primary = DefaultTheme.Primary
	DefaultTheme.PolicyPrinter.Secondary = DefaultTheme.Secondary
}

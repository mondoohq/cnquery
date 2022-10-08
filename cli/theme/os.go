package theme

const logo = " .--. ,-.,-. .---..-..-. .--. .--. .-..-.™\n" +
	"'  ..': ,. :' .; :: :; :' '_.': ..': :; :\n" +
	"`.__.':_;:_;`._. ;`.__.'`.__.':_;  `._. ;\n" +
	"   mondoo™     : :                  .-. :\n" +
	"               :_:                  `._.'"

var DefaultTheme = OperatingSystemTheme

func init() {
	DefaultTheme.PolicyPrinter.Error = DefaultTheme.Error
	DefaultTheme.PolicyPrinter.Primary = DefaultTheme.Primary
	DefaultTheme.PolicyPrinter.Secondary = DefaultTheme.Secondary
}

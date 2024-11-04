# `components` package

This Go package has interactive helpers used by `cnquery` and `cnspec`.

We use a powerful little TUI framework called [bubbletea](https://github.com/charmbracelet/bubbletea).

## `Select` component

Select is an interactive prompt that displays the provided message and displays a
list of items to be selected.

e.g.
```go
type CustomString string

func (s CustomString) Display() string {
	return string(s)
}

func main() {
	customStrings := []CustomString{"first", "second", "third"}
	selected := components.Select("Choose a string", customStrings)
	fmt.Printf("You chose the %s string.\n", customStrings[selected])
}
```

To execute this example:
```
go run cli/components/_examples/selector/main.go
```

## `List` component

List is a non-interactive function that lists items to the user.

e.g.
```go
type CustomString string

func (s CustomString) PrintableKeys() []string {
	return []string{"string"}
}
func (s CustomString) PrintableValue(_ int) string {
	return string(s)
}

func main() {
	customStrings := []CustomString{"first", "second", "third"}
	list := components.List(theme.OperatingSystemTheme, customStrings)
	fmt.Printf(list)
}
```

To execute this example:
```
go run cli/components/_examples/list/main.go
```


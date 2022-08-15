package arista

import (
	"bufio"
	"io"
	"strings"
)

func CountLeadingSpace(line string) int {
	i := 0
	for _, runeValue := range line {
		if runeValue != ' ' {
			break
		}
		i++
	}
	return i
}

func ParseConfig(in io.Reader) map[string]interface{} {
	stack := []map[string]interface{}{}
	keyStack := []string{}

	scanner := bufio.NewScanner(in)

	// add root to stack
	stack = append(stack, map[string]interface{}{})
	keyStack = append(keyStack, "root")

	lastDepth := 0
	lastKey := ""
	for scanner.Scan() {
		line := scanner.Text()
		key := strings.TrimSpace(line)

		if strings.HasPrefix(key, "!") || key == "end" {
			continue
		}

		indent := CountLeadingSpace(line)
		level := 0
		if indent > 0 {
			level = indent / 3
		}

		if level > lastDepth {
			// add level to stack
			entry := map[string]interface{}{}
			stack = append(stack, entry)
			keyStack = append(keyStack, lastKey)
		}

		if level < lastDepth {
			stackKey := keyStack[lastDepth]

			// store stack with proper parent key
			stack[level][stackKey] = stack[level+1]

			levelDiff := lastDepth - level

			// delete old entry from stack
			stack = stack[:len(stack)-levelDiff]
			keyStack = keyStack[:len(keyStack)-levelDiff]
		}

		lastDepth = level
		lastKey = key

		// TODO: only temporary until we can check for key existence in MQL
		stack[level][key] = true
	}

	return stack[0]
}

func GetSection(in io.Reader, section string) string {
	keyStack := []string{}
	keyStack = append(keyStack, "")

	scanner := bufio.NewScanner(in)

	lastDepth := 0
	lastKey := ""
	recorded := ""
	for scanner.Scan() {
		line := scanner.Text()
		key := strings.TrimSpace(line)

		if strings.HasPrefix(key, "!") || key == "end" {
			continue
		}

		indent := CountLeadingSpace(line)
		level := 0
		if indent > 0 {
			level = indent / 3
		}

		if level > lastDepth {
			// add level to stack
			keyStack = append(keyStack, lastKey)
		}

		if level < lastDepth {
			levelDiff := lastDepth - level

			// delete old entry from stack
			keyStack = keyStack[:len(keyStack)-levelDiff]
		}

		lastDepth = level
		lastKey = key

		if strings.Join(keyStack, " ") == " "+section {
			recorded += key + "\n"
		}
	}

	return recorded
}

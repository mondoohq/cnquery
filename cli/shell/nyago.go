// Copyright (c) Nihad Abbasov
// SPDX-License-Identifier: BSD-2-Clause
//
// Code taken from: https://github.com/NARKOZ/go-nyancat

package shell

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pierrec/lz4/v4"
)

func nyago(width, height int) {
	cdec, err := base64.StdEncoding.DecodeString(c)
	if err != nil {
		return
	}

	reader := lz4.NewReader(bytes.NewReader(cdec))
	all := make([]byte, 50000)
	if _, err := reader.Read(all); err != nil && err != io.EOF {
		return
	}

	framesRaw := strings.Split(string(all), "z")
	frames := make([][]string, len(framesRaw))
	for i := range framesRaw {
		frames[i] = strings.Split(framesRaw[i], "\n")
	}

	fmt.Printf("%+v\n", frames)

	stop := make(chan struct{}, 1)
	captureSIGINTonce(stop)

	colors := map[string]string{
		"'": "0",   // outline
		".": "15",  // white
		",": "234", // bg
		">": "198", // lightred (rainbow 1)
		"&": "211", // lightorange (rainbow 2)
		"+": "222", // lightyellow (rainbow 3)
		"#": "86",  // lightgreen (rainbow 4)
		"=": "45",  // lightblue (rainbow 5)
		";": "32",  // lightpurple (rainbow 6)
		"@": "224", // outer body
		"$": "217", // inner body
		"-": "204", // dots on the cat
		"%": "210", // cheeks
		"*": "248", // grey
	}

	fmt.Print("\033[H\033[2J\033[?25l")
	const outputChar = "  "

	y0 := 0
	y1 := len(frames[0])

	x0 := 0
	x1 := len(frames[0][0])

	if y1 > height {
		y0 = (y1 - height) / 2
		y1 = y0 + height
	}

	if x1 > width {
		x0 = (x1 - width) / 2
		x1 = x0 + width
	}

	ticker := time.NewTicker(90 * time.Millisecond)
	defer func() { ticker.Stop() }()

	for i := 0; i < 3; i++ {
		for _, frame := range frames {
			// Print the next frame
			for _, line := range frame[y0:y1] {
				for _, char := range line[x0:x1] {
					fmt.Printf("\033[48;5;%sm%s", colors[string(char)], outputChar)
				}
				fmt.Println("\033[m")
			}

			// Reset the frame and sleep
			fmt.Print("\033[H")
			time.Sleep(90 * time.Millisecond)

			select {
			case <-stop:
				return
			case <-ticker.C:
			}
		}
	}
}

const c = "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"

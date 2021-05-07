package cmd

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.io/mondoo/lumi/lr"
)

func init() {
	rootCmd.AddCommand(markdownCmd)
}

var markdownCmd = &cobra.Command{
	Use:   "markdown",
	Short: "generates markdown files",
	Long:  `parse an LR file and generates a markdown file`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		raw, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		res, err := lr.Parse(string(raw))
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		builder := &strings.Builder{}
		builder.WriteString("# Mondoo Resource Reference\n\n")

		// to ensure we generate the same markdown, we sort the resources first
		sort.SliceStable(res.Resources, func(i, j int) bool {
			return res.Resources[i].ID < res.Resources[j].ID
		})

		for i := range res.Resources {
			resource := res.Resources[i]
			builder.WriteString("## ")
			builder.WriteString(resource.ID)
			builder.WriteString("\n\n")

			if len(resource.Comments) > 0 {
				builder.WriteString("**Description**\n\n")
				builder.WriteString(strings.Join(sanitizeComments(resource.Comments), "\n"))
				builder.WriteString("\n\n")
			}

			// generate the constructor
			if len(resource.Body.Inits) > 0 {
				builder.WriteString("**Init**\n\n")
				for j := range resource.Body.Inits {
					init := resource.Body.Inits[j]

					for a := range init.Args {
						arg := init.Args[a]
						builder.WriteString(resource.ID + "(" + arg.ID + " " + renderLrType(arg.Type) + ")")
						builder.WriteString("\n")
					}
				}
				builder.WriteString("\n")
			}

			if resource.ListType != nil {
				builder.WriteString("**List**\n\n")
				builder.WriteString("[]" + resource.ListType.Type.Type)
				builder.WriteString("\n\n")
			}

			// generate the fields markdown table
			// NOTE: list types may not have any fields
			if len(resource.Body.Fields) > 0 {
				builder.WriteString("**Fields**\n\n")
				rows := [][]string{}

				for k := range resource.Body.Fields {
					field := resource.Body.Fields[k]
					rows = append(rows, []string{field.ID, renderLrType(field.Type), strings.Join(sanitizeComments(field.Comments), ", ")})
				}

				table := tablewriter.NewWriter(builder)
				table.SetHeader([]string{"ID", "Type", "Description"})
				table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
				table.SetCenterSeparator("|")
				table.SetAutoWrapText(false)
				table.AppendBulk(rows)
				table.Render()
				builder.WriteString("\n")
			}
		}

		fmt.Println(builder.String())
	},
}

func renderLrType(t lr.Type) string {
	switch {
	case t.SimpleType != nil:
		return t.SimpleType.Type
	case t.ListType != nil:
		return "[]" + renderLrType(t.ListType.Type)
	case t.MapType != nil:
		return "map[" + t.MapType.Key.Type + "]" + renderLrType(t.MapType.Value)
	default:
		return "?"
	}
}

func sanitizeComments(c []string) []string {
	for i := range c {
		c[i] = strings.TrimPrefix(c[i], "// ")
	}
	return c
}

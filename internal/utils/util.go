// Copyright 2021 MIMIRO AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"

	"github.com/pterm/pterm"
)

var templateFuncs = template.FuncMap{
	"trim":                    strings.TrimSpace,
	"trimRightSpace":          trimRightSpace,
	"trimTrailingWhitespaces": trimRightSpace,
	"appendIfNotPresent":      appendIfNotPresent,
	"rpad":                    rpad,
	"gt":                      cobra.Gt,
	"eq":                      cobra.Eq,
}

func trimRightSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

// appendIfNotPresent will append stringToAppend to the end of s, but only if it's not yet present in s.
func appendIfNotPresent(s, stringToAppend string) string {
	if strings.Contains(s, stringToAppend) {
		return s
	}
	return s + " " + stringToAppend
}

func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(template, s)
}

func ResolveFormat(cmd *cobra.Command) string {
	format := "term"
	js, _ := cmd.Flags().GetBool("json")
	if js {
		format = "json"
		return format
	}
	pretty, _ := cmd.Flags().GetBool("pretty")
	if pretty {
		format = "pretty"
	}
	return format
}

func HandleError(err error) {
	if err != nil {
		pterm.Error.Println(err.Error())
		pterm.Println()
		os.Exit(1)
	}
}

func Pretty(obj interface{}) {
	themBytes, _ := json.Marshal(obj)
	f := pretty.Pretty(themBytes)
	result := pretty.Color(f, nil)
	fmt.Println(string(result))
}

func AskForConfirmation() bool {
	var response string

	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}

	switch strings.ToLower(response) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		fmt.Println("Type (y)es or (n)o and then press enter:")
		return AskForConfirmation()
	}
}

func Tmpl(w io.Writer, text string, data interface{}) error {
	t := template.New("top")
	t.Funcs(templateFuncs)
	template.Must(t.Parse(text))
	return t.Execute(w, data)
}

func ReadStdIn() ([]byte, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return nil, errors.New("no file provided and no stdin pipe")
	} else {
		reader := bufio.NewReader(os.Stdin)
		var output []byte

		for {
			input, err := reader.ReadByte()

			if err != nil && err == io.EOF {
				break
			}
			output = append(output, input)
		}

		return output, nil
	}
}

// ReadInput attempts to read a file from the location given, or if a location
// is not given, then it will attempt to read from stdin instead.
func ReadInput(file string) ([]byte, error) {
	if file != "" {
		// attempt to read file
		k, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}
		return k, nil
	} else {
		fi, err := os.Stdin.Stat()
		if err != nil {
			return nil, err
		}
		if fi.Mode()&os.ModeNamedPipe == 0 {
			return nil, errors.New("no file provided and no stdin pipe")
		} else {
			reader := bufio.NewReader(os.Stdin)
			var output []byte

			for {
				input, err := reader.ReadByte()

				if err != nil && err == io.EOF {
					break
				}
				output = append(output, input)
			}

			return output, nil
		}
	}
}

func SortOutputList(output [][] string, sortBy string) ([][] string){
	header, out := output[0], output[1:]
	headerPosition := 0
	for hPosition := range header{
		if header[hPosition] == sortBy{
			headerPosition = hPosition
		}
	}

	//sorting the list since it is sorted on ids, but we want it to sort on titles
	sort.Slice(out[:], func(i, j int) bool {
		for _ = range out[i] {
			if out[i][headerPosition] == out[j][headerPosition] {
				continue
			}
			return out[i][headerPosition] < out[j][headerPosition]
		}
		return false
	})

	// add header to output before returning
	out = append([][]string{header}, out...)
	return out
}

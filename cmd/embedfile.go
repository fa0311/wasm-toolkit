/*
	Copyright 2023 Loophole Labs

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

		   http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package main

import (
	"fmt"
	"os"
	"path"

	"github.com/loopholelabs/wasm-toolkit/wasmfile"

	"github.com/spf13/cobra"
)

var (
	cmdEmbedfile = &cobra.Command{
		Use:   "embedfile",
		Short: "Add a file to the wasm",
		Long:  `This will embed a file within the wasm`,
		Run:   runEmbedFile,
	}
)

var em_filename = "embedtest"
var em_content = "Yeah!"

func init() {
	rootCmd.AddCommand(cmdEmbedfile)
	cmdEmbedfile.Flags().StringVar(&em_filename, "filename", "embedtest", "Func name regexp")
	cmdEmbedfile.Flags().StringVar(&em_content, "content", "Hey! This isn't really a file. It's embedded in the wasm.", "Func name regexp")
}

func runEmbedFile(ccmd *cobra.Command, args []string) {
	if Input == "" {
		panic("No input file")
	}

	fmt.Printf("Loading wasm file \"%s\"...\n", Input)
	wfile, err := wasmfile.New(Input)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Parsing custom name section...\n")
	err = wfile.ParseName()
	if err != nil {
		panic(err)
	}

	// Add a payload to the wasm file
	memFunctions, err := wasmfile.NewFromWat(path.Join("wat_code", "memory.wat"))
	if err != nil {
		panic(err)
	}

	// TODO: Wrap file imports so we can do what we want to...

	originalFunctionLength := len(wfile.Code)

	wfile.AddFuncsFrom(memFunctions)

	payload_size := 2

	data_ptr := wfile.Memory[0].LimitMin << 16

	wfile.SetGlobal("$debug_mem_size", wasmfile.ValI32, fmt.Sprintf("i32.const %d", payload_size)) // The size of our addition in 64k pages
	wfile.SetGlobal("$debug_start_mem", wasmfile.ValI32, fmt.Sprintf("i32.const %d", data_ptr))
	wfile.Memory[0].LimitMin = wfile.Memory[0].LimitMin + payload_size

	// Now we can start doing interesting things...

	// Add a payload to the wasm file
	embedFunctions, err := wasmfile.NewFromWatWithData(path.Join("wat_code", "embed.wat"), map[string][]byte{
		"$file_name":    []byte(em_filename),
		"$file_content": []byte(em_content),
	})
	if err != nil {
		panic(err)
	}

	wfile.AddDataFrom(int32(data_ptr), embedFunctions)
	wfile.AddFuncsFrom(embedFunctions) // NB: This may mean inserting an import which changes all func numbers.

	// Redirect some imports...
	import_redirect_map := map[string]string{
		"wasi_snapshot_preview1:fd_prestat_get": "$wrap_fd_prestat_get",
		"wasi_snapshot_preview1:path_open":      "$wrap_path_open",
		"wasi_snapshot_preview1:fd_read":        "$wrap_fd_read",
	}

	for from, to := range import_redirect_map {
		fromId := wfile.LookupImport(from)
		toId := wfile.LookupFunctionID(to)

		fmt.Printf("Redirecting code from %d to %d\n", fromId, toId)

		for idx, c := range wfile.Code {
			if idx < originalFunctionLength {
				c.ModifyAllCalls(map[int]int{fromId: toId})
			}
		}

	}

	// Adjust any memory.size / memory.grow calls
	for idx, c := range wfile.Code {
		if idx < originalFunctionLength {
			err = c.ReplaceInstr(wfile, "memory.grow", "call $debug_memory_grow")
			if err != nil {
				panic(err)
			}
			err = c.ReplaceInstr(wfile, "memory.size", "call $debug_memory_size")
			if err != nil {
				panic(err)
			}
		} else {
			// Do any relocation adjustments...
			err = c.InsertAfterRelocating(wfile, `global.get $debug_start_mem
																						i32.add`)
			if err != nil {
				panic(err)
			}
		}
	}

	fmt.Printf("Writing wasm out to %s...\n", Output)
	f, err := os.Create(Output)
	if err != nil {
		panic(err)
	}

	err = wfile.EncodeBinary(f)
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	/*
	   fmt.Printf("Writing debug.wat\n")
	   f2, err := os.Create("debug.wat")

	   	if err != nil {
	   		panic(err)
	   	}

	   err = wfile.EncodeWat(f2)

	   	if err != nil {
	   		panic(err)
	   	}

	   err = f2.Close()

	   	if err != nil {
	   		panic(err)
	   	}
	*/
}

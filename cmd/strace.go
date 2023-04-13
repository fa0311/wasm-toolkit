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
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"regexp"

	"github.com/loopholelabs/wasm-toolkit/wasmfile"

	"github.com/spf13/cobra"
)

var (
	cmdStrace = &cobra.Command{
		Use:   "strace",
		Short: "Use strace to add tracing output to as wasm file",
		Long:  `This will output debug info to STDERR`,
		Run:   runStrace,
	}
)

var include_imports = false
var include_timings = false
var include_line_numbers = false
var include_func_signatures = false
var include_param_names = false
var include_all = false
var func_regex = ".*"
var cfg_color = false

func init() {
	rootCmd.AddCommand(cmdStrace)
	cmdStrace.Flags().StringVarP(&func_regex, "func", "f", ".*", "Func name regexp")
	cmdStrace.Flags().BoolVar(&include_line_numbers, "linenumbers", false, "Include line number info")
	cmdStrace.Flags().BoolVar(&include_func_signatures, "funcsignatures", false, "Include function signatures")
	cmdStrace.Flags().BoolVar(&include_param_names, "paramnames", false, "Include param names")
	cmdStrace.Flags().BoolVar(&include_timings, "timing", false, "Include timing summary")
	cmdStrace.Flags().BoolVar(&include_imports, "imports", false, "Include imports")
	cmdStrace.Flags().BoolVar(&include_all, "all", false, "Include everything")

	cmdStrace.Flags().BoolVar(&cfg_color, "color", false, "Output ANSI color in the log")
}

func runStrace(ccmd *cobra.Command, args []string) {
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

	fmt.Printf("Parsing custom dwarf debug sections...\n")
	err = wfile.ParseDwarf()
	if err != nil {
		panic(err)
	}

	// Add a payload to the wasm file
	memFunctions, err := wasmfile.NewFromWat(path.Join("wat_code", "memory.wat"))
	if err != nil {
		panic(err)
	}

	// Keep track of wasi import wrappers
	wasi_functions := make(map[int]string)

	// Wrap all imports if we need to...
	// Then they will get included in normal debug logging and or timing
	if include_all || include_imports {
		for idx, i := range wfile.Import {

			newidx := len(wfile.Import) + len(wfile.Code)

			// First we create a func wrapper, then adjust all calls
			f := &wasmfile.FunctionEntry{
				TypeIndex: i.Index,
			}

			t := wfile.Type[i.Index]

			// Load the params...
			expr := make([]*wasmfile.Expression, 0)
			for idx, _ := range t.Param {
				expr = append(expr, &wasmfile.Expression{
					Opcode:     wasmfile.InstrToOpcode["local.get"],
					LocalIndex: idx,
				})
			}

			expr = append(expr, &wasmfile.Expression{
				Opcode:    wasmfile.InstrToOpcode["call"],
				FuncIndex: idx,
			})

			c := &wasmfile.CodeEntry{
				Locals:     make([]wasmfile.ValType, 0),
				Expression: expr,
			}

			// Fixup any calls
			for _, c := range wfile.Code {
				c.ModifyAllCalls(map[int]int{idx: newidx})
			}

			// If they're wasi calls. Add function signatures etc
			if i.Module == "wasi_snapshot_preview1" {
				wasi_functions[newidx] = i.Name
				de, ok := wasmfile.Debug_wasi_snapshot_preview1[i.Name]
				if ok {
					wfile.SetFunctionSignature(newidx, de)
				}
			}

			wfile.FunctionNames[newidx] = fmt.Sprintf("$IMPORT_%s_%s", i.Module, i.Name) //wfile.GetFunctionIdentifier(idx, false))

			wfile.Function = append(wfile.Function, f)
			wfile.Code = append(wfile.Code, c)
		}
	}

	originalFunctionLength := len(wfile.Code)

	wfile.AddFuncsFrom(memFunctions)

	data_ptr := wfile.Memory[0].LimitMin << 16

	wfile.SetGlobal("$debug_start_mem", wasmfile.ValI32, fmt.Sprintf("i32.const %d", data_ptr))

	// Now we can start doing interesting things...

	datamap := make(map[string][]byte, 0)

	we_data := make([]byte, 0)
	er_data := make([]byte, 0)

	errors_by_id := make([]string, 77)
	for m, v := range wasmfile.Wasi_errors {
		errors_by_id[v] = m
	}

	for _, m := range errors_by_id {
		we_data = binary.LittleEndian.AppendUint32(we_data, uint32(len(er_data)))
		we_data = binary.LittleEndian.AppendUint32(we_data, uint32(len([]byte(m))))
		er_data = append(er_data, []byte(m)...)
	}

	datamap["$wasi_errors"] = we_data
	datamap["$wasi_error_messages"] = er_data

	//Wasi_errors

	// Add a payload to the wasm file
	debugFunctions, err := wasmfile.NewFromWatWithData(path.Join("wat_code", "strace.wat"), datamap)
	if err != nil {
		panic(err)
	}

	wfile.AddDataFrom(int32(data_ptr), debugFunctions)
	wfile.AddFuncsFrom(debugFunctions) // NB: This may mean inserting an import which changes all func numbers.

	// Parse the dwarf stuff *here* incase the above messed up function IDs
	fmt.Printf("Parsing dwarf line numbers...\n")
	err = wfile.ParseDwarfLineNumbers()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Parsing dwarf local variables...\n")
	err = wfile.ParseDwarfVariables()
	if err != nil {
		panic(err)
	}

	// Pass some config into wasm
	if include_timings {
		wfile.SetGlobal("$debug_do_timings", wasmfile.ValI32, fmt.Sprintf("i32.const 1"))
	}

	if cfg_color {
		wfile.SetGlobal("$debug_color", wasmfile.ValI32, fmt.Sprintf("i32.const 1"))
	}

	fmt.Printf("Patching functions matching regexp \"%s\"\n", func_regex)

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

			functionIndex := idx + len(wfile.Import)
			fidentifier := wfile.GetFunctionIdentifier(functionIndex, false)

			match, err := regexp.MatchString(func_regex, fidentifier)
			if err != nil {
				panic(err)
			}

			if match {
				fmt.Printf("Patching function[%d] %s\n", idx, fidentifier)
				blockInstr := "block"
				f := wfile.Function[idx]
				t := wfile.Type[f.TypeIndex]
				if len(t.Result) > 0 {
					blockInstr = fmt.Sprintf("block (result %s)", wasmfile.ByteToValType[t.Result[0]])
				}

				wfile.AddData(fmt.Sprintf("$function_name_%d", functionIndex), []byte(fidentifier))

				startCode := fmt.Sprintf(`%s
			i32.const %d
			i32.const offset($function_name_%d)
			i32.const length($function_name_%d)
			call $debug_enter_func
			`, blockInstr, functionIndex, functionIndex, functionIndex)

				// Do parameters...
				for paramIndex, pt := range t.Param {
					if paramIndex > 0 {
						startCode = fmt.Sprintf(`%s
					call $debug_param_separator
					`, startCode)
					}

					// NB This assumes CodeSectionPtr to be correct...
					if include_all || include_param_names {
						if c.PCValid {
							vname := wfile.GetLocalVarName(c.CodeSectionPtr, paramIndex)
							if vname != "" {
								wfile.AddData(fmt.Sprintf("$dd_param_name_%d_%d", functionIndex, paramIndex), []byte(vname))
								startCode = fmt.Sprintf(`%s
					i32.const offset($dd_param_name_%d_%d)
					i32.const length($dd_param_name_%d_%d)
					call $debug_param_name
					`, startCode, functionIndex, paramIndex, functionIndex, paramIndex)
							}
						}
					}
					startCode = fmt.Sprintf(`%s
					i32.const %d
					i32.const %d
					local.get %d
					call $debug_enter_%s
					`, startCode, functionIndex, paramIndex, paramIndex, wasmfile.ByteToValType[pt])
				}

				startCode = fmt.Sprintf(`%s
					i32.const %d
					call $debug_enter_end
					`, startCode, functionIndex)

				// Now add a bit of debug....
				funcSig := wfile.GetFunctionSignature(functionIndex)
				if funcSig != "" && (include_all || include_func_signatures) {
					wfile.AddData(fmt.Sprintf("$dd_function_debug_sig_%d", functionIndex), []byte(funcSig))
					startCode = fmt.Sprintf(`%s
					i32.const offset($dd_function_debug_sig_%d)
					i32.const length($dd_function_debug_sig_%d)
					call $debug_func_context`, startCode, functionIndex, functionIndex)
				}

				lineRange := wfile.GetLineNumberRange(functionIndex, c)
				if lineRange != "" && (include_all || include_line_numbers) {
					wfile.AddData(fmt.Sprintf("$dd_function_debug_lines_%d", functionIndex), []byte(lineRange))
					startCode = fmt.Sprintf(`%s
					i32.const offset($dd_function_debug_lines_%d)
					i32.const length($dd_function_debug_lines_%d)
					call $debug_func_context
					`, startCode, functionIndex, functionIndex)
				}

				// If it's a wasi call, then output some detail here...
				wasi_name, is_wasi := wasi_functions[functionIndex]

				// Add some code to show function parameter values...
				startCode = fmt.Sprintf(`%s
					%s`, startCode, wasmfile.GetWasiParamCodeEnter(wasi_name))

				err = c.InsertFuncStart(wfile, startCode)
				if err != nil {
					panic(err)
				}

				rt := wasmfile.ValNone
				if len(t.Result) == 1 {
					rt = t.Result[0]
				}

				endCode := fmt.Sprintf(`i32.const %d
			i32.const offset($function_name_%d)
			i32.const length($function_name_%d)
			call $debug_exit_func`, functionIndex, functionIndex, functionIndex)

				if is_wasi && rt == wasmfile.ValI32 {
					// We also want to output the error message
					endCode = fmt.Sprintf(`%s
					call $debug_exit_func_wasi
					%s`, endCode, wasmfile.GetWasiParamCodeExit(wasi_name))

				} else {
					endCode = fmt.Sprintf(`%s
					call $debug_exit_func_%s`, endCode, wasmfile.ByteToValType[rt])
				}

				err = c.ReplaceInstr(wfile, "return", endCode+"\nreturn")
				if err != nil {
					panic(err)
				}

				err = c.InsertFuncEnd(wfile, "end\n"+endCode)
				if err != nil {
					panic(err)
				}

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

	// Find out how much data we need for the payload
	total_payload_data := data_ptr
	if len(wfile.Data) > 0 {
		last_data := wfile.Data[len(wfile.Data)-1]
		total_payload_data = int(last_data.Offset[0].I32Value) + len(last_data.Data) - data_ptr
	}

	payload_size := (total_payload_data + 65535) >> 16
	fmt.Printf("Payload data of %d (%d pages)\n", total_payload_data, payload_size)

	wfile.SetGlobal("$debug_mem_size", wasmfile.ValI32, fmt.Sprintf("i32.const %d", payload_size)) // The size of our addition in 64k pages
	wfile.Memory[0].LimitMin = wfile.Memory[0].LimitMin + payload_size

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

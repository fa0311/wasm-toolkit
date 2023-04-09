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

package wasmfile

import (
	"debug/dwarf"
	"encoding/binary"
	"fmt"
	"io"
)

func (wf *WasmFile) ParseDwarf() error {
	debug_abbrev := wf.GetCustomSectionData(".debug_abbrev")
	debug_aranges := wf.GetCustomSectionData(".debug_aranges")
	debug_info := wf.GetCustomSectionData(".debug_info")
	debug_line := wf.GetCustomSectionData(".debug_line")
	debug_pubnames := wf.GetCustomSectionData(".debug_pubnames")
	debug_ranges := wf.GetCustomSectionData(".debug_ranges")
	debug_str := wf.GetCustomSectionData(".debug_str")

	debug_loc := wf.GetCustomSectionData(".debug_loc")
	wf.dwarfLoc = debug_loc

	debug_frame := make([]byte, 0) // call frame info

	dd, err := dwarf.New(debug_abbrev, debug_aranges, debug_frame, debug_info, debug_line, debug_pubnames, debug_ranges, debug_str)
	if err != nil {
		return err
	}

	wf.dwarfData = dd
	return nil
}

func (wf *WasmFile) ParseDwarfLineNumbers() error {
	wf.lineNumbers = make(map[uint64]LineInfo)

	entryReader := wf.dwarfData.Reader()

	for {
		// Read all entries in sequence
		entry, err := entryReader.Next()
		if entry == nil || err == io.EOF {
			// We've reached the end of DWARF entries
			break
		}

		if entry.Tag == dwarf.TagCompileUnit {
			liner, err := wf.dwarfData.LineReader(entry)

			if err != nil {
				return err
			}
			if liner != nil {
				ent := dwarf.LineEntry{}
				for {
					err = liner.Next(&ent)
					if err == io.EOF {
						break
					}

					wf.lineNumbers[ent.Address] = LineInfo{
						Filename:   ent.File.Name,
						Linenumber: ent.Line,
						Column:     ent.Column,
					}
				}
			}
		}
	}
	return nil
}

func (wf *WasmFile) GetLocalVarName(pc uint64, index int) string {
	for _, lnd := range wf.localNames {
		if lnd.Index == index && (pc >= lnd.StartPC && pc <= lnd.EndPC) {
			return lnd.VarName
		}
	}
	return ""
}

func (wf *WasmFile) GetLineNumberInfo(pc uint64) string {
	// See if we have any line info...
	lineInfo := ""
	li, ok := wf.lineNumbers[pc]
	if ok {
		lineInfo = fmt.Sprintf("%s:%d.%d", li.Filename, li.Linenumber, li.Column)
	}
	return lineInfo
}

func (wf *WasmFile) GetFunctionDebug(fid int) string {
	de, ok := wf.functionDebug[fid]
	if ok {
		return de
	}
	return ""
}

type LocalNameData struct {
	StartPC uint64
	EndPC   uint64
	Index   int
	VarName string
}

func (wf *WasmFile) ParseDwarfVariables() error {
	wf.functionDebug = make(map[int]string)
	wf.localNames = make([]*LocalNameData, 0)
	entryReader := wf.dwarfData.Reader()

	for {
		// Read all entries in sequence
		entry, err := entryReader.Next()
		if entry == nil || err == io.EOF {
			// We've reached the end of DWARF entries
			break
		}

		if entry.Tag == dwarf.TagSubprogram {
			spname := "<unknown>"
			sploc := uint64(0)
			for _, field := range entry.Field {
				//				log.Printf("Field %v\n", field)
				if field.Attr == dwarf.AttrName {
					spname = field.Val.(string)
				} else if field.Attr == dwarf.AttrLowpc {
					switch field.Val.(type) {
					case uint64:
						sploc = field.Val.(uint64)
					}
				}
			}

			params := ""
			locals := ""
			if entry.Children {
				// Read the children...
				for {
					entry, err := entryReader.Next()
					if err != nil {
						return err
					}
					if entry.Tag == 0 {
						break
					}

					vname := "<unknown>"
					vtype := ""
					vloc := int64(-1)
					for _, field := range entry.Field {
						if field.Attr == dwarf.AttrName {
							vname = field.Val.(string)
						} else if field.Attr == dwarf.AttrType {
							t := field.Val.(dwarf.Offset)
							ty, err := wf.dwarfData.Type(t)
							if err == nil {
								vtype = ty.String()
							}
						} else if field.Attr == dwarf.AttrLocation {
							switch field.Val.(type) {
							case int64:
								vloc = field.Val.(int64)
							}
						}
					}

					if entry.Tag == dwarf.TagFormalParameter {
						if vloc != -1 {
							locdata := wf.ReadLocation(uint64(vloc))
							for _, ld := range locdata {
								// We have code ptr range here...

								locs := extractWasmDwarfExpression(ld.expression)
								for _, l := range locs {
									if l.isLocal {
										// Store in the locals lookup...
										wf.localNames = append(wf.localNames, &LocalNameData{
											StartPC: uint64(ld.startAddress),
											EndPC:   uint64(ld.endAddress),
											Index:   int(l.index),
											VarName: vname,
										})

										//										fmt.Printf("LocationLocal %s %s %d-%d  local %d\n", spname, vname, ld.startAddress, ld.endAddress, l.index)
									}
								}
							}
						}
						if len(params) > 0 {
							params = params + ", "
						}
						params = fmt.Sprintf("%s%s(%s)", params, vname, vtype)
					} else if entry.Tag == dwarf.TagVariable {

						//						fmt.Printf("TagVariable %v\n", entry)

						if vloc != -1 {
							locdata := wf.ReadLocation(uint64(vloc))
							for _, ld := range locdata {
								// We have code ptr range here...

								//								fmt.Printf("Var Data %s is %d %d %x\n", vname, ld.startAddress, ld.endAddress, ld.expression)

								locs := extractWasmDwarfExpression(ld.expression)
								for _, l := range locs {
									if l.isLocal {
										// Store in the locals lookup...
										wf.localNames = append(wf.localNames, &LocalNameData{
											StartPC: uint64(ld.startAddress),
											EndPC:   uint64(ld.endAddress),
											Index:   int(l.index),
											VarName: vname,
										})

										//										fmt.Printf("LocationLocal %s %s %d-%d  local %d\n", spname, vname, ld.startAddress, ld.endAddress, l.index)
									}
								}
							}
						}
						locals = fmt.Sprintf("%s;; local %s %s\n", locals, vname, vtype)
					}

				}
			}
			function_debug := fmt.Sprintf(";; %s(%s)\n%s", spname, params, locals)

			fid := wf.FindFunction(sploc)

			if fid != -1 {
				wf.functionDebug[fid] = function_debug
			}
		}
	}
	return nil
}

type LocationData struct {
	startAddress uint32
	endAddress   uint32
	expression   []byte
}

const DW_OP_WASM_location = 0xed
const DW_Location_Local = 0
const DW_Location_Global = 1
const DW_Location_Stack = 2 // 0 = bottom of the stack
const DW_Location_Global_i32 = 3

const DW_OP_stack_value = 0x9f
const DW_OP_piece = 0x93

type WasmLocation struct {
	isLocal  bool
	isGlobal bool
	isStack  bool
	index    uint64
}

func extractWasmDwarfExpression(data []byte) []*WasmLocation {
	locs := make([]*WasmLocation, 0)
	for {
		if len(data) == 0 {
			break
		}
		opcode := data[0]
		data = data[1:]
		if opcode == DW_OP_stack_value {
			// Fine...
		} else if opcode == DW_OP_piece {
			_, l := binary.Uvarint(data)
			data = data[l:]
		} else if opcode == DW_OP_WASM_location {
			t := data[0]
			data = data[1:]
			var index uint64
			if t == 3 {
				index = uint64(binary.LittleEndian.Uint32(data))
				data = data[4:]
			} else {
				var l int
				index, l = binary.Uvarint(data)
				data = data[l:]
			}
			locs = append(locs, &WasmLocation{
				isLocal:  t == DW_Location_Local,
				isGlobal: t == DW_Location_Global || t == DW_Location_Global_i32,
				isStack:  t == DW_Location_Stack,
				index:    index,
			})

		} else {
			// FIXME: Deal with other dwarf opcodes
			//			fmt.Printf("WARN: Unknown dwarf expression opcode %d %x\n", opcode, orgdata)
			return locs
		}
	}
	return locs
}

func (wf *WasmFile) ReadLocation(p uint64) []*LocationData {
	baseAddress := uint32(0)
	ld := make([]*LocationData, 0)

	ptr := p
	for {
		low := binary.LittleEndian.Uint32(wf.dwarfLoc[ptr:])
		ptr += 4
		high := binary.LittleEndian.Uint32(wf.dwarfLoc[ptr:])
		ptr += 4
		if low == 0 && high == 0 {
			break
		}
		if low == 0xffffffff {
			baseAddress = high
		} else {
			// Read expr len
			explen := binary.LittleEndian.Uint16(wf.dwarfLoc[ptr:])
			ptr += 2
			expr := wf.dwarfLoc[ptr : ptr+uint64(explen)]
			ptr += uint64(explen)
			ld = append(ld, &LocationData{
				startAddress: baseAddress + low,
				endAddress:   baseAddress + high,
				expression:   expr,
			})
		}
	}
	return ld
}

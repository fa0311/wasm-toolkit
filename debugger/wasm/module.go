package wasm

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

type WasmModule struct {
	filename string

	Types   []*Type
	Imports []*Import
	Funcs   []*Func
	Tables  []*Table
	Memorys []*Memory
	Globals []*Global
	Exports []*Export
	Elems   []*Elem
	Datas   []*Data
}

func NewModule(file string) *WasmModule {
	return &WasmModule{
		filename: file,

		Types:   make([]*Type, 0),
		Imports: make([]*Import, 0),
		Funcs:   make([]*Func, 0),
		Tables:  make([]*Table, 0),
		Memorys: make([]*Memory, 0),
		Globals: make([]*Global, 0),
		Exports: make([]*Export, 0),
		Elems:   make([]*Elem, 0),
		Datas:   make([]*Data, 0),
	}
}

func (wm *WasmModule) ReplaceConst(s string, i int) {
	news := fmt.Sprintf("%d", i)
	for _, f := range wm.Funcs {
		for index, i := range f.Instructions {
			f.Instructions[index] = strings.Replace(i, s, news, -1)
		}
	}
}

func (wm *WasmModule) ReplaceInstruction(s string, t string) {
	for _, f := range wm.Funcs {
		for index, i := range f.Instructions {
			if strings.Trim(i, Whitespace) == s {
				f.Instructions[index] = t
			}
		}
	}
}

// Wrap an import into an internal function
func (wm *WasmModule) WrapImport(i *Import) (*Func, string) {
	// Usually just (type 0)
	sig := i.GetFuncSignature() // This could be (type 0) or (param...) (result) etc
	typeParams := ""

	fun_name1 := i.Identifier1
	if strings.HasPrefix(fun_name1, "\"") {
		fun_name1 = fun_name1[1 : len(fun_name1)-1]
	}
	fun_name2 := i.Identifier2
	if strings.HasPrefix(fun_name2, "\"") {
		fun_name2 = fun_name2[1 : len(fun_name2)-1]
	}

	funName := fmt.Sprintf("$import.%s.%s", fun_name1, fun_name2)

	code := make([]string, 0)

	pindex := 0

	var e string
	for {
		sig = strings.Trim(sig, Whitespace) // Skip to next bit
		if len(sig) == 0 {
			break
		}
		e, sig = ReadElement(sig)
		if e[0] != '(' {
			panic("Not element")
		}
		eType, _ := ReadToken(e[1:])
		if eType == "type" {
			v := e[6 : len(e)-1]
			i, err := strconv.Atoi(v)
			if err != nil {
				panic("Invalid type val")
			}
			ty := wm.Types[i]
			// Strip the func
			if strings.HasPrefix(ty.Type, "(func ") {
				params := ty.Type[6 : len(ty.Type)-1]
				typeParams = params
				// Now parse it and move it over to the function...

				for {
					params = strings.Trim(params, Whitespace) // Skip to next bit
					if len(params) == 0 {
						break
					}
					var pa string
					pa, params = ReadElement(params)
					if strings.HasPrefix(pa, "(param ") {
						bits := pa[7 : len(pa)-1]
						words := strings.Fields(bits)

						for _, w := range words {
							if w == "i32" {
								code = append(code, fmt.Sprintf("local.get %d", pindex))
								pindex++
							} else if w == "i64" {
								code = append(code, fmt.Sprintf("local.get %d", pindex))
								pindex++
							} else if w == "f32" {
								code = append(code, fmt.Sprintf("local.get %d", pindex))
								pindex++
							} else if w == "f64" {
								code = append(code, fmt.Sprintf("local.get %d", pindex))
								pindex++
							}
						}
					}
				}
			}
		} else if eType == "param" {
			fmt.Printf("DEBUG param %s", e)
			panic("Can't do yet")
		} else if eType == "result" {
			fmt.Printf("DEBUG result %s", e)
			panic("Can't do yet")
		}
	}

	code = append(code, fmt.Sprintf("call %s", i.GetFuncName()))
	f := NewFunc(fmt.Sprintf("(func %s %s %s\n)", funName, sig, typeParams))
	f.Instructions = code
	return f, funName
}

func (wm *WasmModule) Parse() {
	data, err := ioutil.ReadFile(wm.filename)
	if err != nil {
		log.Fatal(err)
	}

	text := string(data)

	// Read the module
	moduleText, _ := ReadElement(text)

	moduleType, _ := ReadToken(moduleText[1:])

	//fmt.Printf("Module %s is %d bytes\n", wm.filename, len(moduleText))

	// Now read all the individual elements from within the module...

	text = text[len(moduleType)+1:]

	for {
		text = strings.TrimLeft(text, " \t\r\n") // Skip to next bit
		// End of the module?
		if text[0] == ')' {
			break
		}

		// Skip any single line comments
		for {
			if strings.HasPrefix(text, ";;") {
				// Skip to end of line
				p := strings.Index(text, "\n")
				if p == -1 {
					panic("TODO: Comment without newline")
				}
				text = text[p+1:]
				text = strings.TrimLeft(text, " \t\r\n") // Skip to next bit
			} else {
				break
			}
		}

		e, _ := ReadElement(text)
		eType, _ := ReadToken(e[1:])

		if eType == "data" {
			d := NewData(e)
			wm.Datas = append(wm.Datas, d)
		} else if eType == "elem" {
			el := NewElem(e)
			wm.Elems = append(wm.Elems, el)
		} else if eType == "export" {
			ex := NewExport(e)
			wm.Exports = append(wm.Exports, ex)
		} else if eType == "func" {
			f := NewFunc(e)
			wm.Funcs = append(wm.Funcs, f)
		} else if eType == "global" {
			g := NewGlobal(e)
			wm.Globals = append(wm.Globals, g)
		} else if eType == "import" {
			i := NewImport(e)
			wm.Imports = append(wm.Imports, i)
		} else if eType == "memory" {
			mem := NewMemory(e)
			wm.Memorys = append(wm.Memorys, mem)
		} else if eType == "table" {
			t := NewTable(e)
			wm.Tables = append(wm.Tables, t)
		} else if eType == "type" {
			t := NewType(e)
			wm.Types = append(wm.Types, t)
		} else {
			panic(fmt.Sprintf("Unknown element \"%s\"", eType))
		}

		// Skip over this element
		text = text[len(e):]
	}
	/*
	   fmt.Printf("Parsed wat file. %d Data, %d Elem, %d Export, %d Func, %d Global, %d Import, %d Memory, %d Table, %d type\n",

	   	len(wm.Datas),
	   	len(wm.Elems),
	   	len(wm.Exports),
	   	len(wm.Funcs),
	   	len(wm.Globals),
	   	len(wm.Imports),
	   	len(wm.Memorys),
	   	len(wm.Tables),
	   	len(wm.Types),

	   )
	*/
}

func (m *WasmModule) Write() string {
	d := "(module\n"

	// Write out the various things...

	for _, t := range m.Types {
		d = d + t.Write() + "\n"
	}

	for _, i := range m.Imports {
		d = d + i.Write() + "\n"
	}

	for _, f := range m.Funcs {
		d = d + f.Write() + "\n"
	}

	for _, t := range m.Tables {
		d = d + t.Write() + "\n"
	}

	for _, m := range m.Memorys {
		d = d + m.Write() + "\n"
	}

	for _, g := range m.Globals {
		d = d + g.Write() + "\n"
	}

	for _, e := range m.Exports {
		d = d + e.Write() + "\n"
	}

	for _, e := range m.Elems {
		d = d + e.Write() + "\n"
	}

	for _, da := range m.Datas {
		d = d + da.Write() + "\n"
	}

	return d + ")"
}

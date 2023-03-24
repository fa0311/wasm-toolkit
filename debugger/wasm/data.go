package wasm

import (
	"fmt"
	"strconv"
	"strings"
)

/**
 * Example:
 * (data $.data (i32.const 66160) "x\9c\19\f6\dc\02\01\00\00\00\00\00\9c\03\01\00\c1\82\01\00\00\00\00\00\04\00\00\00\0c\00\00\00\01\00\00\00\00\00\00\00\01\00\00\00\00\00\00\00\02\00\00\00\a8\02\01\00\98\01\00\00\01\00\00\00\ff\01\01\00\0b\00\00\00\00\00\00\00 \01\01\00\13\00\00\003\01\01\00\13"))
 *
 *
 */

type Data struct {
	Source     string
	Identifier string
	Location   string
	Data       string
}

func NewData(src string) *Data {
	// Skip the "(data", and remove the trailing ")"
	s := strings.TrimLeft(src[5:len(src)-1], Whitespace)

	s = SkipComment(s)

	var id string
	if s[0] == '$' {
		id, s = ReadToken(s)
	}

	loc, s := ReadElement(s)
	data, s := ReadString(s)

	return &Data{
		Source:     src,
		Identifier: id,
		Location:   loc,
		Data:       data,
	}
}

func (d *Data) Write() string {
	if d.Identifier == "" {
		return fmt.Sprintf("(data %s %s)", d.Location, d.Data)
	}
	return fmt.Sprintf("(data %s %s %s)", d.Identifier, d.Location, d.Data)
}

func (d *Data) AdjustLocation(adj int) {
	if strings.HasPrefix(d.Location, "(i32.const") {
		loc, err := strconv.Atoi(d.Location[11 : len(d.Location)-1])
		if err != nil {
			panic("Wasm.data: Error parsing location")
		}
		d.Location = fmt.Sprintf("(i32.const %d)", loc+adj)
	} else {
		panic("Wasm.data: Can only have i32.const location")
	}
}

func MergeDatas(prefix1 string, data1 []*Data, offset2 int, prefix2 string, data2 []*Data) []*Data {
	datas := make([]*Data, 0)

	// mod1 can be added unchanged.
	for _, i1 := range data1 {
		if i1.Identifier != "" {
			i1.Identifier = "$" + prefix1 + i1.Identifier[1:]
		}
		datas = append(datas, i1)
	}

	// mod2 needs offset added.
	for _, i1 := range data2 {
		if i1.Identifier != "" {
			i1.Identifier = "$" + prefix2 + i1.Identifier[1:]
		}
		i1.AdjustLocation(offset2)
		datas = append(datas, i1)
	}

	return datas
}

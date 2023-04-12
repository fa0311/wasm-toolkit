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
	"bufio"
	"fmt"
	"io"
)

func (e *Expression) EncodeWat(w io.Writer, prefix string, wf *WasmFile) error {
	comment := "" //fmt.Sprintf("    ;; PC=%d", e.PC) // TODO From line numbers, vars etc

	lineNumberData := wf.GetLineNumberInfo(e.PC)
	if lineNumberData != "" {
		comment = fmt.Sprintf(" ;; Src = %s", lineNumberData)
	}

	wr := bufio.NewWriter(w)

	defer func() {
		wr.Flush()
	}()

	// First deal with simple opcodes (No args)
	if e.Opcode == InstrToOpcode["unreachable"] ||
		e.Opcode == InstrToOpcode["nop"] ||
		e.Opcode == InstrToOpcode["return"] ||
		e.Opcode == InstrToOpcode["drop"] ||
		e.Opcode == InstrToOpcode["select"] ||
		e.Opcode == InstrToOpcode["end"] ||
		e.Opcode == InstrToOpcode["else"] ||
		e.Opcode == InstrToOpcode["i32.eqz"] ||
		e.Opcode == InstrToOpcode["i32.eq"] ||
		e.Opcode == InstrToOpcode["i32.ne"] ||
		e.Opcode == InstrToOpcode["i32.lt_s"] ||
		e.Opcode == InstrToOpcode["i32.lt_u"] ||
		e.Opcode == InstrToOpcode["i32.gt_s"] ||
		e.Opcode == InstrToOpcode["i32.gt_u"] ||
		e.Opcode == InstrToOpcode["i32.le_s"] ||
		e.Opcode == InstrToOpcode["i32.le_u"] ||
		e.Opcode == InstrToOpcode["i32.ge_s"] ||
		e.Opcode == InstrToOpcode["i32.ge_u"] ||
		e.Opcode == InstrToOpcode["i64.eqz"] ||
		e.Opcode == InstrToOpcode["i64.eq"] ||
		e.Opcode == InstrToOpcode["i64.ne"] ||
		e.Opcode == InstrToOpcode["i64.lt_s"] ||
		e.Opcode == InstrToOpcode["i64.lt_u"] ||
		e.Opcode == InstrToOpcode["i64.gt_s"] ||
		e.Opcode == InstrToOpcode["i64.gt_u"] ||
		e.Opcode == InstrToOpcode["i64.le_s"] ||
		e.Opcode == InstrToOpcode["i64.le_u"] ||
		e.Opcode == InstrToOpcode["i64.ge_s"] ||
		e.Opcode == InstrToOpcode["i64.ge_u"] ||
		e.Opcode == InstrToOpcode["f32.eq"] ||
		e.Opcode == InstrToOpcode["f32.ne"] ||
		e.Opcode == InstrToOpcode["f32.lt"] ||
		e.Opcode == InstrToOpcode["f32.gt"] ||
		e.Opcode == InstrToOpcode["f32.le"] ||
		e.Opcode == InstrToOpcode["f32.ge"] ||
		e.Opcode == InstrToOpcode["f64.eq"] ||
		e.Opcode == InstrToOpcode["f64.ne"] ||
		e.Opcode == InstrToOpcode["f64.lt"] ||
		e.Opcode == InstrToOpcode["f64.gt"] ||
		e.Opcode == InstrToOpcode["f64.le"] ||
		e.Opcode == InstrToOpcode["f64.ge"] ||

		e.Opcode == InstrToOpcode["i32.clz"] ||
		e.Opcode == InstrToOpcode["i32.ctz"] ||
		e.Opcode == InstrToOpcode["i32.popcnt"] ||
		e.Opcode == InstrToOpcode["i32.add"] ||
		e.Opcode == InstrToOpcode["i32.sub"] ||
		e.Opcode == InstrToOpcode["i32.mul"] ||
		e.Opcode == InstrToOpcode["i32.div_s"] ||
		e.Opcode == InstrToOpcode["i32.div_u"] ||
		e.Opcode == InstrToOpcode["i32.rem_s"] ||
		e.Opcode == InstrToOpcode["i32.rem_u"] ||
		e.Opcode == InstrToOpcode["i32.and"] ||
		e.Opcode == InstrToOpcode["i32.or"] ||
		e.Opcode == InstrToOpcode["i32.xor"] ||
		e.Opcode == InstrToOpcode["i32.shl"] ||
		e.Opcode == InstrToOpcode["i32.shr_s"] ||
		e.Opcode == InstrToOpcode["i32.shr_u"] ||
		e.Opcode == InstrToOpcode["i32.rotl_s"] ||
		e.Opcode == InstrToOpcode["i32.rotr_u"] ||

		e.Opcode == InstrToOpcode["i64.clz"] ||
		e.Opcode == InstrToOpcode["i64.ctz"] ||
		e.Opcode == InstrToOpcode["i64.popcnt"] ||
		e.Opcode == InstrToOpcode["i64.add"] ||
		e.Opcode == InstrToOpcode["i64.sub"] ||
		e.Opcode == InstrToOpcode["i64.mul"] ||
		e.Opcode == InstrToOpcode["i64.div_s"] ||
		e.Opcode == InstrToOpcode["i64.div_u"] ||
		e.Opcode == InstrToOpcode["i64.rem_s"] ||
		e.Opcode == InstrToOpcode["i64.rem_u"] ||
		e.Opcode == InstrToOpcode["i64.and"] ||
		e.Opcode == InstrToOpcode["i64.or"] ||
		e.Opcode == InstrToOpcode["i64.xor"] ||
		e.Opcode == InstrToOpcode["i64.shl"] ||
		e.Opcode == InstrToOpcode["i64.shr_s"] ||
		e.Opcode == InstrToOpcode["i64.shr_u"] ||
		e.Opcode == InstrToOpcode["i64.rotl_s"] ||
		e.Opcode == InstrToOpcode["i64.rotr_u"] ||

		e.Opcode == InstrToOpcode["f32.abs"] ||
		e.Opcode == InstrToOpcode["f32.neg"] ||
		e.Opcode == InstrToOpcode["f32.ceil"] ||
		e.Opcode == InstrToOpcode["f32.floor"] ||
		e.Opcode == InstrToOpcode["f32.trunc"] ||
		e.Opcode == InstrToOpcode["f32.nearest"] ||
		e.Opcode == InstrToOpcode["f32.sqrt"] ||
		e.Opcode == InstrToOpcode["f32.add"] ||
		e.Opcode == InstrToOpcode["f32.sub"] ||
		e.Opcode == InstrToOpcode["f32.mul"] ||
		e.Opcode == InstrToOpcode["f32.div"] ||
		e.Opcode == InstrToOpcode["f32.min"] ||
		e.Opcode == InstrToOpcode["f32.max"] ||
		e.Opcode == InstrToOpcode["f32.copysign"] ||

		e.Opcode == InstrToOpcode["f64.abs"] ||
		e.Opcode == InstrToOpcode["f64.neg"] ||
		e.Opcode == InstrToOpcode["f64.ceil"] ||
		e.Opcode == InstrToOpcode["f64.floor"] ||
		e.Opcode == InstrToOpcode["f64.trunc"] ||
		e.Opcode == InstrToOpcode["f64.nearest"] ||
		e.Opcode == InstrToOpcode["f64.sqrt"] ||
		e.Opcode == InstrToOpcode["f64.add"] ||
		e.Opcode == InstrToOpcode["f64.sub"] ||
		e.Opcode == InstrToOpcode["f64.mul"] ||
		e.Opcode == InstrToOpcode["f64.div"] ||
		e.Opcode == InstrToOpcode["f64.min"] ||
		e.Opcode == InstrToOpcode["f64.max"] ||
		e.Opcode == InstrToOpcode["f64.copysign"] ||

		e.Opcode == InstrToOpcode["i32.wrap_i64"] ||
		e.Opcode == InstrToOpcode["i32.trunc_f32_s"] ||
		e.Opcode == InstrToOpcode["i32.trunc_f32_u"] ||
		e.Opcode == InstrToOpcode["i32.trunc_f64_s"] ||
		e.Opcode == InstrToOpcode["i32.trunc_f64_u"] ||
		e.Opcode == InstrToOpcode["i64.extend_i32_s"] ||
		e.Opcode == InstrToOpcode["i64.extend_i32_u"] ||
		e.Opcode == InstrToOpcode["i64.trunc_f32_s"] ||
		e.Opcode == InstrToOpcode["i64.trunc_f32_u"] ||
		e.Opcode == InstrToOpcode["i64.trunc_f64_s"] ||
		e.Opcode == InstrToOpcode["i64.trunc_f64_u"] ||
		e.Opcode == InstrToOpcode["f32.convert_i32_s"] ||
		e.Opcode == InstrToOpcode["f32.convert_i32_u"] ||
		e.Opcode == InstrToOpcode["f32.convert_i64_s"] ||
		e.Opcode == InstrToOpcode["f32.convert_i64_u"] ||
		e.Opcode == InstrToOpcode["f32.demote_f64"] ||
		e.Opcode == InstrToOpcode["f64.convert_i32_s"] ||
		e.Opcode == InstrToOpcode["f64.convert_i32_u"] ||
		e.Opcode == InstrToOpcode["f64.convert_i64_s"] ||
		e.Opcode == InstrToOpcode["f64.convert_i64_u"] ||
		e.Opcode == InstrToOpcode["f64.promote_f32"] ||
		e.Opcode == InstrToOpcode["i32.reinterpret_f32"] ||
		e.Opcode == InstrToOpcode["i64.reinterpret_f64"] ||
		e.Opcode == InstrToOpcode["f32.reinterpret_i32"] ||
		e.Opcode == InstrToOpcode["f64.reinterpret_i64"] ||

		e.Opcode == InstrToOpcode["i32.extend8_s"] ||
		e.Opcode == InstrToOpcode["i32.extend16_s"] ||
		e.Opcode == InstrToOpcode["i64.extend8_s"] ||
		e.Opcode == InstrToOpcode["i64.extend16_s"] ||
		e.Opcode == InstrToOpcode["i64.extend32_s"] {

		_, err := wr.WriteString(fmt.Sprintf("%s%s%s\n", prefix, opcodeToInstr[e.Opcode], comment))
		return err
	} else if e.Opcode == InstrToOpcode["br_table"] {
		targets := ""
		for _, l := range e.Labels {
			targets = fmt.Sprintf("%s %d", targets, l)
		}
		defaultTarget := fmt.Sprintf(" %d", e.LabelIndex)
		_, err := wr.WriteString(fmt.Sprintf("%s%s%s%s%s\n", prefix, opcodeToInstr[e.Opcode], targets, defaultTarget, comment))
		return err
	} else if e.Opcode == InstrToOpcode["br"] ||
		e.Opcode == InstrToOpcode["br_if"] {
		target := fmt.Sprintf(" %d", e.LabelIndex)
		_, err := wr.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, opcodeToInstr[e.Opcode], target, comment))
		return err
	} else if e.Opcode == InstrToOpcode["i32.load"] ||
		e.Opcode == InstrToOpcode["i64.load"] ||
		e.Opcode == InstrToOpcode["f32.load"] ||
		e.Opcode == InstrToOpcode["f64.load"] ||
		e.Opcode == InstrToOpcode["i32.load8_s"] ||
		e.Opcode == InstrToOpcode["i32.load8_u"] ||
		e.Opcode == InstrToOpcode["i32.load16_s"] ||
		e.Opcode == InstrToOpcode["i32.load16_u"] ||
		e.Opcode == InstrToOpcode["i64.load8_s"] ||
		e.Opcode == InstrToOpcode["i64.load8_u"] ||
		e.Opcode == InstrToOpcode["i64.load16_s"] ||
		e.Opcode == InstrToOpcode["i64.load16_u"] ||
		e.Opcode == InstrToOpcode["i64.load32_s"] ||
		e.Opcode == InstrToOpcode["i64.load32_u"] ||
		e.Opcode == InstrToOpcode["i32.store"] ||
		e.Opcode == InstrToOpcode["i64.store"] ||
		e.Opcode == InstrToOpcode["f32.store"] ||
		e.Opcode == InstrToOpcode["f64.store"] ||
		e.Opcode == InstrToOpcode["i32.store8"] ||
		e.Opcode == InstrToOpcode["i32.store16"] ||
		e.Opcode == InstrToOpcode["i64.store8"] ||
		e.Opcode == InstrToOpcode["i64.store16"] ||
		e.Opcode == InstrToOpcode["i64.store32"] {
		modAlign := fmt.Sprintf(" align=%d", 1<<e.MemAlign)
		modOffset := fmt.Sprintf(" offset=%d", e.MemOffset)
		if e.MemOffset == 0 {
			modOffset = ""
		}
		// TODO: Default align?
		/*
			if e.MemAlign == 0 {
				modAlign = ""
			}
		*/
		_, err := wr.WriteString(fmt.Sprintf("%s%s%s%s%s\n", prefix, opcodeToInstr[e.Opcode], modOffset, modAlign, comment))
		return err
	} else if e.Opcode == InstrToOpcode["memory.size"] ||
		e.Opcode == InstrToOpcode["memory.grow"] {
		_, err := wr.WriteString(fmt.Sprintf("%s%s%s\n", prefix, opcodeToInstr[e.Opcode], comment))
		return err
	} else if e.Opcode == InstrToOpcode["block"] ||
		e.Opcode == InstrToOpcode["if"] ||
		e.Opcode == InstrToOpcode["loop"] {

		result := ""
		if e.Result != ValNone {
			result = fmt.Sprintf(" (result %s)", ByteToValType[e.Result])
		}

		_, err := wr.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, opcodeToInstr[e.Opcode], result, comment))

		return err
	} else if e.Opcode == InstrToOpcode["i32.const"] {
		value := fmt.Sprintf(" %d", e.I32Value)
		_, err := wr.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, opcodeToInstr[e.Opcode], value, comment))
		return err
	} else if e.Opcode == InstrToOpcode["i64.const"] {
		value := fmt.Sprintf(" %d", e.I64Value)
		_, err := wr.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, opcodeToInstr[e.Opcode], value, comment))
		return err
	} else if e.Opcode == InstrToOpcode["f32.const"] {
		value := fmt.Sprintf(" %f", e.F32Value)
		_, err := wr.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, opcodeToInstr[e.Opcode], value, comment))
		return err
	} else if e.Opcode == InstrToOpcode["f64.const"] {
		value := fmt.Sprintf(" %f", e.F64Value)
		_, err := wr.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, opcodeToInstr[e.Opcode], value, comment))
		return err
	} else if e.Opcode == InstrToOpcode["local.get"] ||
		e.Opcode == InstrToOpcode["local.set"] ||
		e.Opcode == InstrToOpcode["local.tee"] {
		tname := wf.GetLocalVarName(e.PC, e.LocalIndex)
		if tname != "" {
			comment = comment + " ;; Variable " + tname
		}
		localTarget := fmt.Sprintf(" %d", e.LocalIndex)
		_, err := wr.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, opcodeToInstr[e.Opcode], localTarget, comment))
		return err
	} else if e.Opcode == InstrToOpcode["global.get"] ||
		e.Opcode == InstrToOpcode["global.set"] {
		g := wf.GetGlobalIdentifier(e.GlobalIndex)
		globalTarget := fmt.Sprintf(" %s", g)
		_, err := wr.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, opcodeToInstr[e.Opcode], globalTarget, comment))
		return err
	} else if e.Opcode == InstrToOpcode["call"] {
		f := wf.GetFunctionIdentifier(e.FuncIndex, false)
		callTarget := fmt.Sprintf(" %s", f)
		_, err := wr.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, opcodeToInstr[e.Opcode], callTarget, comment))
		return err
	} else if e.Opcode == InstrToOpcode["call_indirect"] {
		typeIndex := fmt.Sprintf(" (type %d)", e.TypeIndex)
		_, err := wr.WriteString(fmt.Sprintf("%s%s%s%s\n", prefix, opcodeToInstr[e.Opcode], typeIndex, comment))
		return err
	} else if e.Opcode == ExtendedOpcodeFC {
		// Now deal with opcode2...
		if e.OpcodeExt == instrToOpcodeFC["memory.copy"] {
			_, err := wr.WriteString(fmt.Sprintf("%s%s%s\n", prefix, opcodeToInstrFC[e.OpcodeExt], comment))
			return err
		} else if e.OpcodeExt == instrToOpcodeFC["memory.fill"] {
			_, err := wr.WriteString(fmt.Sprintf("%s%s%s\n", prefix, opcodeToInstrFC[e.OpcodeExt], comment))
			return err
		} else if e.OpcodeExt == instrToOpcodeFC["i32.trunc_sat_f32_s"] ||
			e.OpcodeExt == instrToOpcodeFC["i32.trunc_sat_f32_u"] ||
			e.OpcodeExt == instrToOpcodeFC["i32.trunc_sat_f64_s"] ||
			e.OpcodeExt == instrToOpcodeFC["i32.trunc_sat_f64_u"] ||
			e.OpcodeExt == instrToOpcodeFC["i64.trunc_sat_f32_s"] ||
			e.OpcodeExt == instrToOpcodeFC["i64.trunc_sat_f32_u"] ||
			e.OpcodeExt == instrToOpcodeFC["i64.trunc_sat_f64_s"] ||
			e.OpcodeExt == instrToOpcodeFC["i64.trunc_sat_f64_u"] {
			_, err := wr.WriteString(fmt.Sprintf("%s%s%s\n", prefix, opcodeToInstrFC[e.OpcodeExt], comment))
			return err
		} else {
			return fmt.Errorf("Unsupported opcode 0xfc %d", e.OpcodeExt)
		}
	} else {
		return fmt.Errorf("Unsupported opcode %d", e.Opcode)
	}

}

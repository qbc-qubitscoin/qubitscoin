package vm

// CounterContractWASM is a hand-crafted WASM binary implementing a simple counter.
//
// WAT equivalent:
//
//	(module
//	  (import "env" "qbc_get" (func $get (param i32) (result i64)))
//	  (import "env" "qbc_set" (func $set (param i32 i64)))
//	  (func (export "increment")
//	    ;; qbc_set(0, qbc_get(0) + 1)
//	    i32.const 0
//	    i32.const 0
//	    call $get        ;; stack: [i32(0), i64(count)]
//	    i64.const 1
//	    i64.add          ;; stack: [i32(0), i64(count+1)]
//	    call $set
//	  )
//	  (func (export "get_count") (result i64)
//	    i32.const 0
//	    call $get
//	  )
//	)
//
// Section sizes are computed manually below.
var CounterContractWASM = buildCounterWASM()

func buildCounterWASM() []byte {
	// ── Type section (id=0x01) ────────────────────────────────────────────
	// 4 types:
	//  [0] (i32) -> (i64): qbc_get
	//  [1] (i32,i64) -> () : qbc_set
	//  [2] () -> () : increment
	//  [3] () -> (i64): get_count
	typeContent := []byte{
		0x04,                         // count=4
		0x60, 0x01, 0x7f, 0x01, 0x7e, // type 0: (i32)->(i64)
		0x60, 0x02, 0x7f, 0x7e, 0x00, // type 1: (i32,i64)->()
		0x60, 0x00, 0x00, // type 2: ()->()
		0x60, 0x00, 0x01, 0x7e, // type 3: ()->(i64)
	}

	// ── Import section (id=0x02) ─────────────────────────────────────────
	// 2 imports: "env"."qbc_get" (type 0), "env"."qbc_set" (type 1)
	importContent := []byte{
		0x02,                   // count=2
		0x03, 0x65, 0x6e, 0x76, // "env"  (len=3)
		0x07, 0x71, 0x62, 0x63, 0x5f, 0x67, 0x65, 0x74, // "qbc_get" (len=7)
		0x00, 0x00, // kind=func, type idx 0
		0x03, 0x65, 0x6e, 0x76, // "env"  (len=3)
		0x07, 0x71, 0x62, 0x63, 0x5f, 0x73, 0x65, 0x74, // "qbc_set" (len=7)
		0x00, 0x01, // kind=func, type idx 1
	}

	// ── Function section (id=0x03) ───────────────────────────────────────
	// 2 locally defined functions: type 2 (increment), type 3 (get_count)
	funcContent := []byte{
		0x02,       // count=2
		0x02, 0x03, // type indices
	}

	// ── Export section (id=0x07) ─────────────────────────────────────────
	// func indices: import 0 = qbc_get (idx 0), import 1 = qbc_set (idx 1)
	// local func: increment (idx 2), get_count (idx 3)
	exportContent := []byte{
		0x02,                                                                   // count=2
		0x09, 0x69, 0x6e, 0x63, 0x72, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x00, 0x02, // "increment"->func 2
		0x09, 0x67, 0x65, 0x74, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x00, 0x03, // "get_count"->func 3
	}

	// ── Code section (id=0x0a) ────────────────────────────────────────────
	// Body 0: increment()
	//   locals: 0
	//   i32.const 0; slot arg for qbc_set
	//   i32.const 0; slot arg for qbc_get
	//   call 0       ; qbc_get(0) → i64 on stack
	//   i64.const 1
	//   i64.add
	//   call 1       ; qbc_set(0, count+1)
	//   end
	body0 := []byte{
		0x00,       // 0 locals
		0x41, 0x00, // i32.const 0
		0x41, 0x00, // i32.const 0
		0x10, 0x00, // call 0 (qbc_get)
		0x42, 0x01, // i64.const 1
		0x7c,       // i64.add
		0x10, 0x01, // call 1 (qbc_set)
		0x0b, // end
	} // 13 bytes

	// Body 1: get_count() -> i64
	//   i32.const 0
	//   calls 0
	//   end
	body1 := []byte{
		0x00,       // 0 locals
		0x41, 0x00, // i32.const 0
		0x10, 0x00, // call 0 (qbc_get)
		0x0b, // end
	} // 6 bytes

	codeContent := []byte{0x02} // count=2
	codeContent = append(codeContent, byte(len(body0)))
	codeContent = append(codeContent, body0...)
	codeContent = append(codeContent, byte(len(body1)))
	codeContent = append(codeContent, body1...)

	// ── Assemble module ───────────────────────────────────────────────────
	module := []byte{
		0x00, 0x61, 0x73, 0x6d, // magic
		0x01, 0x00, 0x00, 0x00, // version
	}
	module = appendSection(module, 0x01, typeContent)
	module = appendSection(module, 0x02, importContent)
	module = appendSection(module, 0x03, funcContent)
	module = appendSection(module, 0x07, exportContent)
	module = appendSection(module, 0x0a, codeContent)
	return module
}

// appendSection writes [sectionID][leb128(len(content))][content] to buf.
func appendSection(buf []byte, id byte, content []byte) []byte {
	buf = append(buf, id)
	buf = appendULEB128(buf, uint32(len(content)))
	buf = append(buf, content...)
	return buf
}

// appendULEB128 encodes v as an unsigned LEB128 integer into buf.
func appendULEB128(buf []byte, v uint32) []byte {
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		buf = append(buf, b)
		if v == 0 {
			break
		}
	}
	return buf
}

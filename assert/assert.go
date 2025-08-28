package assert

func Assert(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}

func AssertNotEmpty(s string) {
	if s == "" {
		panic("expected non-empty string")
	}
}

func AssertNotNil(a any) {
	if a == nil {
		panic("expect non-nil value")
	}
}

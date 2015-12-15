package main

type TestErr struct {
	Msg string
}

func (t *TestErr) Error() string {
	return "test err"
}

func b() (error, error) {
//	var a *TestErr = nil
//	var b *TestErr = &TestErr{}
//	a = &TestErr{}
//	a = nil
	var z *TestErr
//	z = nil
	if zz := z; true {
		if true {
//			b = nil
//			a = &TestErr{}
//			b = &TestErr{}
//			b = nil
//			c := zz
			zz = &TestErr{}
			return nil, nil
		} else {
//			a = &TestErr{}
			return zz, nil
		}
//	} else if true {
//		a = &TestErr{}
//	} else {
//		a = &TestErr{}
	}
//	for {
//		b = nil
//		if true {
//			b = &TestErr{}
//		}
//	}
//	b = &TestErr{}
//	var c *TestErr
//	if b = c; b != nil {
//
//	}
//	d := &TestErr{}
//	d = nil
	return nil, nil
}

func main() {
	e1, _ := b()
	if e1 == nil {
		panic("b == nil")
	} else {
		panic("b != nil")
	}
}

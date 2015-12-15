package main

type TestErr struct {
	Msg string
}

func (t *TestErr) Error() string {
	return "test err"
}

func b() (error, error) {
	var a *TestErr = nil
	var b *TestErr = &TestErr{}
	a = &TestErr{}
	a = nil
	if true {
		if true {
			b = nil
			a = &TestErr{}
			b = &TestErr{}
			b = nil
		} else {
//			a = &TestErr{}
		}
	} else if true {
		a = &TestErr{}
	} else {
		a = &TestErr{}
	}
	return a, b
}

func main() {
	e1, _ := b()
	if e1 == nil {
		panic("b == nil")
	} else {
		panic("b != nil")
	}
}

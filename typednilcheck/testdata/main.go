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
			var c *TestErr = nil
			return a, c
		} else {
			a = &TestErr{}
		}
	} else if true {
		a = &TestErr{}
	} else {
		a = &TestErr{}
	}
	for {
		b = nil
		if true {
			b = &TestErr{}
		}
	}
	b = &TestErr{}
	if b = nil; b != nil {

	}
	return nil, b
}

func main() {
	e1, _ := b()
	if e1 == nil {
		panic("b == nil")
	} else {
		panic("b != nil")
	}
}

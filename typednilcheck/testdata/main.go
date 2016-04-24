package main

//import "fmt"

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
	//	if zz := b; true {
	//		return zz, nil
	//		if zz = z; true {
	//			b = nil
	//			a = &TestErr{}
	//			b = &TestErr{}
	//			b = nil
	//			c := zz
	//			zz = &TestErr{}
	//			fmt.Println(zz)
	//			return nil, nil
	//		} else if zz := b; true {
	//			if zz = z; true {
	//				return zz, nil
	//			}
	//			a = &TestErr{}
	//			return zz, nil
	//			zz = &TestErr{}
	//		} else {
	//			zz = &TestErr{}
	//		}
	//		return zz, nil
	//	} else if true {
	//		a = &TestErr{}
	//	} else {
	//		a = &TestErr{}
	//	}
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
	//	var z *TestErr = nil
	//	b := &TestErr{}
	//	a := "aaa"
	//	switch a {
	//	case "aaa":
	//		o := 1
	//		fmt.Println(o)
	//		z = &TestErr{}
	//		fallthrough
	//	case "bb":
	//		fmt.Println("!212")
	//		z = &TestErr{}
	//	}
	//	return z, b
	x := takeEx()
	z := takeEx
	z()
	return x, nil
}

func takeEx() *TestErr {
	return nil
}

func main() {
	e1, _ := b()
	if e1 == nil {
		panic("b == nil")
	} else {
		panic("b != nil")
	}
}

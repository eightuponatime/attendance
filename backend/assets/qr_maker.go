package main

import (
	"fmt"

	"github.com/skip2/go-qrcode"
)

func MakeQR(link string, name string) error {
	return qrcode.WriteFile(
		link,
		qrcode.High,
		800,
		"images/"+fmt.Sprintf("%s.png", name),
	)
}

func main() {
	err := MakeQR("https://office.kzgcj.kz:33235", "employee")
	if err != nil {
		panic(err)
	}

	err = MakeQR("https://kzgcj.kz:33237/admin", "admin")
	if err != nil {
		panic(err)
	}
}

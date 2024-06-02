package main

import "fmt"

// 定义接口
type MyInterface interface {
	MustImplement()
	CanHaveDefault()
}

// 创建默认实现结构体
type DefaultImpl struct{}

func (di DefaultImpl) CanHaveDefault() {
	fmt.Println("This is the default implementation.")
}

// 创建实现接口的结构体
type MyStruct struct {
	DefaultImpl
	x string
}

func (m MyStruct) MustImplement() {
	fmt.Println("This method must be implemented by MyStruct.")
}

// 定义接受接口参数的函数
func abc(x MyInterface) {
	x.MustImplement()
	x.CanHaveDefault()
}

// 主函数
func mains() {
	m := MyStruct{x: ""}
	abc(m) // 传递MyStruct的实例到函数abc
}

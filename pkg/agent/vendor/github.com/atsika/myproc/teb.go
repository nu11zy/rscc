package myproc

import (
	"unsafe"
)

// GetTEBptr returns the address of the start of TEB
func GetTEBptr() uintptr

// GetTEB returns a pointer to the TEB structure
func GetTEB() *TEB {
	return (*TEB)(unsafe.Pointer(GetTEBptr()))
}

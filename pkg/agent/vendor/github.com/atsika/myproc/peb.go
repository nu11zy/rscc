package myproc

import (
	"unsafe"
)

// GetPEBptr returns the address of the start of PEB
func GetPEBptr() uintptr

// GetPEB returns a pointer to the PEB structure
func GetPEB() *PEB {
	return (*PEB)(unsafe.Pointer(GetPEBptr()))
}

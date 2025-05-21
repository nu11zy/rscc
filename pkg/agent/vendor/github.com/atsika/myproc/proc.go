package myproc

import (
	"reflect"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// NewProc is the reimplementation of GetProcAddress using binary search (3x faster) with a linear search fallback.
// It implements search by name, ordinal or hash.
func NewProc[T ~string | ~uint16 | ~uint32](dll *windows.DLL, procedure T) *windows.Proc {

	if dll == nil || dll.Handle == 0 {
		return nil
	}

	// In case this is a forwarded function
	var parent string

	proc := new(windows.Proc)
	proc.Dll = dll

	switch reflect.TypeOf(procedure).Kind() {
	case reflect.String:
		proc.Name = any(procedure).(string)
	}

	dataDir := GetDataDirectory(dll.Handle, IMAGE_DIRECTORY_ENTRY_EXPORT)
	exportDir := GetExportDirectory(dll.Handle)

	procAddr := ResolveFunctionAddr(dll, procedure)

	// Check if this is a forwarded function
	for IsForwardedFunction(procAddr, exportDir, dataDir.Size) {
		forwardName := windows.BytePtrToString((*byte)(unsafe.Pointer(procAddr)))
		forward := strings.Split(forwardName, ".")

		// Check if it starts with api or ext
		if strings.HasPrefix(forward[0], "api-") || strings.HasPrefix(forward[0], "ext-") {
			host := ResolveApiSet(forward[0], parent)
			// Forwarded again ?
			if !strings.HasPrefix(host, "api-") && !strings.HasPrefix(host, "ext-") {
				parent = host
			}
			// Set new dll
			proc.Dll = NewDLL(host)
			exportDir = GetExportDirectory(proc.Dll.Handle)
			// Resolve again
			procAddr = ResolveFunctionAddr(proc.Dll, forward[1])
		} else {
			procAddr = ResolveFunctionAddr(NewDLL(forward[0]+".dll"), forward[1])
		}
	}

	// Still not found ? return nil
	if procAddr != 0 {
		// trick to set unexported addr field in windows.Proc structure
		addr := reflect.ValueOf(proc).Elem().FieldByName("addr")
		reflect.NewAt(addr.Type(), unsafe.Pointer(addr.UnsafeAddr())).Elem().Set(reflect.ValueOf(procAddr))
		return proc
	}

	return nil
}

// ResolveFunctionAddr returns the address of a function. Search by name, ordinal or hash.
func ResolveFunctionAddr[T ~string | ~uint16 | ~uint32](dll *windows.DLL, procedure T) uintptr {
	if dll == nil || dll.Handle == 0 {
		return 0
	}

	var procName string
	var procOrdinal uint16
	var procHash uint32
	var procAddr uintptr

	switch reflect.TypeOf(procedure).Kind() {
	case reflect.String:
		procName = any(procedure).(string)
	case reflect.Uint16:
		procOrdinal = any(procedure).(uint16)
	case reflect.Uint32:
		procHash = any(procedure).(uint32)
	}

	if procHash == 0 && procName != "" {
		procHash = Hash(procName)
	}

	module := unsafe.Pointer(dll.Handle)
	exportDir := GetExportDirectory(dll.Handle)

	addrOfFunctions := unsafe.Add(module, exportDir.AddressOfFunctions)
	addrOfNames := unsafe.Add(module, exportDir.AddressOfNames)
	addrOfNameOrdinals := unsafe.Add(module, exportDir.AddressOfNameOrdinals)

	sliceOfFunctions := unsafe.Slice((*uint32)(addrOfFunctions), exportDir.NumberOfFunctions)
	sliceOfNames := unsafe.Slice((*uint32)(addrOfNames), exportDir.NumberOfNames)
	sliceOfAddrOfNameOrdinals := unsafe.Slice((*uint16)(addrOfNameOrdinals), exportDir.NumberOfNames)

	// ordinal search
	if procOrdinal != 0 {
		procOrdinal -= uint16(exportDir.Base)
		procAddr = uintptr(unsafe.Add(module, sliceOfFunctions[procOrdinal]))
		return procAddr
	}

	// binary search
	if procName != "" {

		left := uintptr(0)
		right := uintptr(exportDir.NumberOfNames - 1)

		for left != right {

			middle := left + ((right - left) >> 1)
			currentName := windows.BytePtrToString((*byte)(unsafe.Add(module, sliceOfNames[middle])))

			if Hash(currentName) == procHash {
				index := sliceOfAddrOfNameOrdinals[middle]
				procAddr = uintptr(unsafe.Add(module, sliceOfFunctions[index]))
				return procAddr
			} else if currentName < procName {
				left = middle + 1
			} else {
				right = middle - 1
			}
		}
	}

	// linear search
	if procAddr == 0 {

		for i := uintptr(0); i < uintptr(exportDir.NumberOfNames); i++ {
			currentName := windows.BytePtrToString((*byte)(unsafe.Add(module, sliceOfNames[i])))
			if Hash(currentName) == procHash {
				index := sliceOfAddrOfNameOrdinals[i]
				procAddr = uintptr(unsafe.Add(module, sliceOfFunctions[index]))
				return procAddr
			}
		}
	}

	return 0
}

// IsForwardedFunction checks if the proc is valid, if not it's a forwarded function
func IsForwardedFunction(procAddr uintptr, exportDir *IMAGE_EXPORT_DIRECTORY, exportDirSize uint32) bool {
	if procAddr >= uintptr(unsafe.Pointer(exportDir)) && procAddr < uintptr(unsafe.Pointer(exportDir))+uintptr(exportDirSize) {
		return true
	}
	return false
}

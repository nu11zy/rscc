package myproc

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// GetDosHeader returns the IMAGE_DOS_HEADER of a module
func GetDosHeader[T ~uintptr | ~unsafe.Pointer](module T) *IMAGE_DOS_HEADER {
	return (*IMAGE_DOS_HEADER)(unsafe.Pointer(module))
}

// GetNtHeaders returns the IMAGE_NT_HEADERS of a module
func GetNtHeaders[T ~uintptr | ~unsafe.Pointer](module T) *IMAGE_NT_HEADERS {
	dosHdrs := GetDosHeader(module)
	return (*IMAGE_NT_HEADERS)(unsafe.Add(unsafe.Pointer(module), dosHdrs.E_lfanew))
}

// GetDataDirectory returns the IMAGE_DATA_DIRECTORY of a module specified by the entry index
func GetDataDirectory[T ~uintptr | ~unsafe.Pointer](module T, entry int) *IMAGE_DATA_DIRECTORY {
	ntHdrs := GetNtHeaders(module)
	return (*IMAGE_DATA_DIRECTORY)(unsafe.Pointer(&ntHdrs.OptionalHeader.DataDirectory[entry]))
}

func GetExportDirectory[T ~uintptr | ~unsafe.Pointer](module T) *IMAGE_EXPORT_DIRECTORY {
	dataDir := GetDataDirectory(module, IMAGE_DIRECTORY_ENTRY_EXPORT)
	return (*IMAGE_EXPORT_DIRECTORY)(unsafe.Add(unsafe.Pointer(module), dataDir.VirtualAddress))
}

// ImageFirstSection returns the first section by parsing NT headers
func ImageFirstSection(ntHdrs *IMAGE_NT_HEADERS) *IMAGE_SECTION_HEADER {
	sectionOffset := unsafe.Offsetof(ntHdrs.OptionalHeader) + uintptr(ntHdrs.FileHeader.SizeOfOptionalHeader)
	return (*IMAGE_SECTION_HEADER)(unsafe.Add(unsafe.Pointer(ntHdrs), sectionOffset))
}

// GetSectionHeader returns the IMAGE_SECTION_HEADER specified by the section name for the given module or nil if the section is not found
func GetSectionHeader[T ~uintptr | ~unsafe.Pointer](module T, sectionName string) *IMAGE_SECTION_HEADER {
	ntHdrs := GetNtHeaders(module)
	section := ImageFirstSection(ntHdrs)
	sections := unsafe.Slice(section, ntHdrs.FileHeader.NumberOfSections)

	for i := uint16(0); i < ntHdrs.FileHeader.NumberOfSections; i++ {
		name := windows.BytePtrToString(&sections[i].Name[0])
		if name == sectionName {
			return section
		}
	}

	return nil
}

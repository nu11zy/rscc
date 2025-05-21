package myproc

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Resolve API Set V6
// Thanks to these references:
// https://lucasg.github.io/2017/10/15/Api-set-resolution/
// https://github.com/ajkhoury/ApiSet

// ResolveApiSet returns the name of the real function host.
func ResolveApiSet(apiSet string, parentName string) string {
	ns := GetApiSetNamespace()

	// api-ms-win-core-apiquery-l1-1-0.dll -> api-ms-win-core-apiquery-l1-1
	apiToResolve := apiSet[:strings.LastIndex(apiSet, "-")]

	entry := ns.SearchForApiSet(apiToResolve)
	if entry == nil {
		return ""
	}

	hostLibEntry := new(API_SET_VALUE_ENTRY)

	if entry.ValueCount > 1 && parentName != "" {
		hostLibEntry = ns.SearchForApiSetHost(entry, parentName)
	} else if entry.ValueCount > 0 {
		hostLibEntry = entry.GetValueEntry(0)
	}

	return hostLibEntry.Value()
}

// GetApiSetNamespace returns an API_SET_NAMESPACE structure from PEB
func GetApiSetNamespace() *API_SET_NAMESPACE {
	return GetPEB().ApiSetMap
}

// GetApiSetNamespacePtr returns a pointer to the API_SET_NAMESPACE structure from PEB
func GetApiSetNamespacePtr() unsafe.Pointer {
	return unsafe.Pointer(GetApiSetNamespace())
}

// SearchForApiSet searches for an API set in the API set namespace
func (ns *API_SET_NAMESPACE) SearchForApiSet(apiToResolve string) *API_SET_NAMESPACE_ENTRY {
	lower := strings.ToLower(apiToResolve)
	hashKey := uint32(0)

	for i := 0; i < len(lower); i++ {
		hashKey = hashKey*ns.HashFactor + uint32(lower[i])
	}

	// binary search
	low := uint32(0)
	middle := uint32(0)
	high := ns.Count - 1

	hashEntry := new(API_SET_HASH_ENTRY)
	for high >= low {
		middle = (high + low) >> 1
		hashEntry = ns.GetHashEntry(middle)

		if hashKey < hashEntry.Hash {
			high = middle - 1
		} else if hashKey > hashEntry.Hash {
			low = middle + 1
		} else {
			break
		}
	}

	// not found
	if high < low {
		return nil
	}

	foundEntry := ns.GetNsEntry(hashEntry.Index)
	name := foundEntry.Name()

	// equivalent to truncate after last hyphen
	if strings.HasPrefix(lower, strings.ToLower(name)) {
		return nil
	}

	return foundEntry
}

// SearchForApiSetHost searches for an API set host in the API set namespace
func (apins *API_SET_NAMESPACE) SearchForApiSetHost(entry *API_SET_NAMESPACE_ENTRY, apiToResolve string) *API_SET_VALUE_ENTRY {

	foundEntry := entry.GetValueEntry(0)

	high := entry.ValueCount - 1
	if high == 0 {
		return foundEntry
	}

	host := new(API_SET_VALUE_ENTRY)
	for low := uint32(1); low <= high; {
		middle := (low + high) >> 1
		host = entry.GetValueEntry(middle)
		switch name := host.Name(); {
		case apiToResolve == name:
			return host
		case apiToResolve < name:
			high = middle - 1
		case apiToResolve > name:
			low = middle + 1
		}
	}

	return nil
}

// GetHashEntry returns the API_SET_HASH_ENTRY at the specified index
func (ns *API_SET_NAMESPACE) GetHashEntry(i uint32) *API_SET_HASH_ENTRY {
	return (*API_SET_HASH_ENTRY)(unsafe.Add(unsafe.Pointer(ns), ns.HashOffset+(i*uint32(unsafe.Sizeof(API_SET_HASH_ENTRY{})))))
}

// GetNsEntry returns the API_SET_NAMESPACE_ENTRY at the specified index
func (ns *API_SET_NAMESPACE) GetNsEntry(i uint32) *API_SET_NAMESPACE_ENTRY {
	return (*API_SET_NAMESPACE_ENTRY)(unsafe.Add(unsafe.Pointer(ns), ns.EntryOffset+(i*uint32(unsafe.Sizeof(API_SET_NAMESPACE_ENTRY{})))))
}

// Name returns the name of the given API_SET_VALUE_ENTRY
func (entry *API_SET_NAMESPACE_ENTRY) Name() string {
	name := (*uint16)(unsafe.Add(GetApiSetNamespacePtr(), entry.NameOffset))
	return windows.UTF16PtrToString(name)
}

// GetNsValueEntry returns the API_SET_VALUE_ENTRY at the specified index
func (entry *API_SET_NAMESPACE_ENTRY) GetValueEntry(i uint32) *API_SET_VALUE_ENTRY {
	return (*API_SET_VALUE_ENTRY)(unsafe.Add(GetApiSetNamespacePtr(), entry.ValueOffset+(i*uint32(unsafe.Sizeof(API_SET_VALUE_ENTRY{})))))
}

// Value returns the value of the given API_SET_VALUE_ENTRY
func (entry *API_SET_VALUE_ENTRY) Value() string {
	value := (*uint16)(unsafe.Add(GetApiSetNamespacePtr(), entry.ValueOffset))
	name := unsafe.Slice(value, entry.ValueLength/uint32(unsafe.Sizeof(uint16(0))))
	return windows.UTF16ToString(name)
}

// Name returns the name of the given API_SET_VALUE_ENTRY
func (entry *API_SET_VALUE_ENTRY) Name() string {
	value := (*uint16)(unsafe.Add(GetApiSetNamespacePtr(), entry.NameOffset))
	name := unsafe.Slice(value, entry.NameLength/uint32(unsafe.Sizeof(uint16(0))))
	return windows.UTF16ToString(name)
}

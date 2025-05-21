package executeassembly

//go:generate go run golang.org/x/sys/windows/mkwinsyscall@latest -output zsyscalls_windows.go syscalls_windows.go

//sys VirtualAllocEx(hProcess windows.Handle, lpAddress uintptr, dwSize uintptr, flAllocationType uint32, flProtect uint32) (addr uintptr, err error) = kernel32.VirtualAllocEx
//sys WriteProcessMemory(hProcess windows.Handle, lpBaseAddress uintptr, lpBuffer *byte, nSize uintptr, lpNumberOfBytesWritten *uintptr) (err error) = kernel32.WriteProcessMemory
//sys VirtualProtectEx(hProcess windows.Handle, lpAddress uintptr, dwSize uintptr, flNewProtect uint32, lpflOldProtect *uint32) (err error) = kernel32.VirtualProtectEx
//sys CreateRemoteThread(hProcess windows.Handle, lpThreadAttributes *windows.SecurityAttributes, dwStackSize uint32, lpStartAddress uintptr, lpParameter uintptr, dwCreationFlags uint32, lpThreadId *uint32)(threadHandle windows.Handle, err error) = kernel32.CreateRemoteThread
//sys GetExitCodeThread(hThread windows.Handle, lpExitCode *uint32) (err error) = kernel32.GetExitCodeThread

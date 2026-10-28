// Package elevate provides Windows UAC elevation helpers.
package elevate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	swShowNormal      = 1
	seErrAccessDenied = 5
	errorCancelled    = 1223
)

// ErrCancelled is returned when the user dismisses the UAC prompt.
var ErrCancelled = errors.New("elevation cancelled by user")

var procShellExecuteW = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW")

// IsElevated reports whether the current process is running with an
// administrator token.
func IsElevated() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	member, err := windows.GetCurrentProcessToken().IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

// RelaunchElevated starts a new copy of the current executable through the
// UAC "runas" verb. The caller should exit after a nil return.
func RelaunchElevated() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	args := make([]string, 0, len(os.Args)-1)
	for _, arg := range os.Args[1:] {
		args = append(args, syscall.EscapeArg(arg))
	}
	params := strings.Join(args, " ")

	cwd, err := os.Getwd()
	if err != nil {
		cwd = filepath.Dir(exe)
	}

	verbPtr, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return fmt.Errorf("encode verb: %w", err)
	}
	exePtr, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return fmt.Errorf("encode executable: %w", err)
	}

	var paramsPtr *uint16
	if params != "" {
		paramsPtr, err = windows.UTF16PtrFromString(params)
		if err != nil {
			return fmt.Errorf("encode arguments: %w", err)
		}
	}

	var cwdPtr *uint16
	if cwd != "" {
		cwdPtr, err = windows.UTF16PtrFromString(cwd)
		if err != nil {
			return fmt.Errorf("encode working directory: %w", err)
		}
	}

	ret, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(exePtr)),
		uintptr(unsafe.Pointer(paramsPtr)),
		uintptr(unsafe.Pointer(cwdPtr)),
		uintptr(swShowNormal),
	)
	if ret <= 32 {
		if ret == seErrAccessDenied || ret == errorCancelled {
			return ErrCancelled
		}
		return fmt.Errorf("ShellExecuteW failed (code %d): %v", ret, callErr)
	}

	return nil
}

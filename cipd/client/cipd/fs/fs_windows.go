// Copyright 2015 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build windows

package fs

import (
	"os"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

// See https://msdn.microsoft.com/en-us/library/windows/desktop/aa365240(v=vs.85).aspx

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procMoveFileExW = kernel32.NewProc("MoveFileExW")
)

const (
	moveFileReplaceExisting = 1
	moveFileWriteThrough    = 8
)

// longFileName converts a non-UNC path to a \?\\ style long path.
func longFileName(path string) string {
	const MAGIC = `\\?\`
	if !strings.HasPrefix(path, MAGIC) {
		path = MAGIC + path
	}
	return path
}

// openFile on Windows ensures that the FILE_SHARE_DELETE sharing permission is
// applied to the file when it is opened. This is required in order for
// atomicRename to be able to rename a file while it is being read.
func openFile(path string) (*os.File, error) {
	lpFileName, err := syscall.UTF16PtrFromString(longFileName(path))
	if err != nil {
		return nil, err
	}
	// Read-only, full shared access, no descriptor inheritance.
	handle, err := syscall.CreateFile(
		lpFileName,
		uint32(syscall.GENERIC_READ),
		uint32(syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE),
		nil,
		uint32(syscall.OPEN_EXISTING),
		syscall.FILE_ATTRIBUTE_NORMAL,
		0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), path), nil
}

func moveFileEx(source, target *uint16, flags uint32) error {
	ret, _, err := procMoveFileExW.Call(uintptr(unsafe.Pointer(source)), uintptr(unsafe.Pointer(target)), uintptr(flags))
	runtime.KeepAlive(source)
	runtime.KeepAlive(target)
	if ret == 0 {
		if err != nil {
			return err
		}
		return syscall.EINVAL
	}
	return nil
}

func mostlyAtomicRename(source, target string) error {
	source, target = longFileName(source), longFileName(target)

	lpTarget, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	lpSource, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return err
	}

	// MoveFileEx is unable to replace read-only files. Do best effort to remove
	// the read-only flag (in most cases where mostlyAtomicRename is used in CIPD
	// it is set, since CIPD uses it to update files it itself installed and they
	// are usually read-only). No big deal if this fails or if MoveFileEx below
	// fails. The caller will eventually delete `target` one way or another, so
	// leaving it with read-only bit removed is fine.
	if attrs, err := syscall.GetFileAttributes(lpTarget); err == nil && (attrs&syscall.FILE_ATTRIBUTE_READONLY) != 0 {
		syscall.SetFileAttributes(lpTarget, attrs&^syscall.FILE_ATTRIBUTE_READONLY)
	}

	return moveFileEx(lpSource, lpTarget, moveFileReplaceExisting|moveFileWriteThrough)
}

// For errors codes see
// https://docs.microsoft.com/en-us/windows/desktop/debug/system-error-codes--0-499-

const (
	// "Access is denied."
	ERROR_ACCESS_DENIED syscall.Errno = 5
	// "The directory is not empty."
	ERROR_DIR_NOT_EMPTY syscall.Errno = 145
	// "The directory name is invalid".
	ERROR_DIRECTORY syscall.Errno = 267
)

func errnoNotEmpty(err error) bool     { return err == ERROR_DIR_NOT_EMPTY }
func errnoNotDir(err error) bool       { return err == ERROR_DIRECTORY }
func errnoAccessDenied(err error) bool { return err == ERROR_ACCESS_DENIED }

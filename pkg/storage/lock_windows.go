//go:build windows

package storage

import (
	"errors"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func platformLockFile(f *os.File, shared bool) error {
	flags := uint32(0)
	if !shared {
		flags = windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	var overlapped windows.Overlapped
	return windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, &overlapped)
}

func platformUnlockFile(f *os.File) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &overlapped)
}

func platformOpenDirectory(path string) (*os.File, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		pathPtr,
		windows.FILE_LIST_DIRECTORY|windows.FILE_TRAVERSE|windows.FILE_READ_ATTRIBUTES|windows.SYNCHRONIZE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return nil, platformNormalizeOpenError(err)
	}
	file := os.NewFile(uintptr(handle), path)
	if err := rejectWindowsReparseHandle(file, true); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func platformOpenDirectoryAt(parent *os.File, name string, create bool) (*os.File, error) {
	disposition := uint32(windows.FILE_OPEN)
	if create {
		disposition = windows.FILE_OPEN_IF
	}
	file, err := windowsOpenRelative(
		parent,
		name,
		windows.FILE_LIST_DIRECTORY|windows.FILE_TRAVERSE|windows.FILE_READ_ATTRIBUTES|windows.SYNCHRONIZE,
		disposition,
		windows.FILE_ATTRIBUTE_DIRECTORY,
		windows.FILE_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT,
	)
	if err != nil {
		return nil, err
	}
	if err := rejectWindowsReparseHandle(file, true); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func platformOpenRegularFileAt(parent *os.File, name string, flags int, perm os.FileMode) (*os.File, error) {
	_ = perm // Windows creates files with attributes rather than POSIX mode bits.
	access := uint32(windows.FILE_READ_ATTRIBUTES | windows.SYNCHRONIZE)
	switch flags & (os.O_WRONLY | os.O_RDWR) {
	case os.O_WRONLY:
		access |= windows.FILE_GENERIC_WRITE
	case os.O_RDWR:
		access |= windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE
	default:
		access |= windows.FILE_GENERIC_READ
	}
	if flags&os.O_APPEND != 0 {
		access |= windows.FILE_APPEND_DATA
	}
	disposition := uint32(windows.FILE_OPEN)
	if flags&os.O_CREATE != 0 {
		if flags&os.O_EXCL != 0 {
			disposition = windows.FILE_CREATE
		} else {
			disposition = windows.FILE_OPEN_IF
		}
	}
	file, err := windowsOpenRelative(
		parent,
		name,
		access,
		disposition,
		windows.FILE_ATTRIBUTE_NORMAL,
		windows.FILE_NON_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT,
	)
	if err != nil {
		return nil, err
	}
	if err := rejectWindowsReparseHandle(file, false); err != nil {
		_ = file.Close()
		return nil, err
	}
	if flags&os.O_TRUNC != 0 {
		if err := file.Truncate(0); err != nil {
			_ = file.Close()
			return nil, err
		}
	}
	return file, nil
}

func windowsOpenRelative(parent *os.File, name string, access, disposition, attributes, options uint32) (*os.File, error) {
	objectName, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return nil, err
	}
	oa := &windows.OBJECT_ATTRIBUTES{
		RootDirectory: windows.Handle(parent.Fd()),
		ObjectName:    objectName,
		Attributes:    windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE,
	}
	oa.Length = uint32(unsafe.Sizeof(*oa))
	var handle windows.Handle
	var status windows.IO_STATUS_BLOCK
	err = windows.NtCreateFile(
		&handle,
		access,
		oa,
		&status,
		nil,
		attributes,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		disposition,
		options,
		0,
		0,
	)
	if err != nil {
		return nil, platformNormalizeOpenError(windowsNTStatusErr(err))
	}
	return os.NewFile(uintptr(handle), name), nil
}

// platformNormalizeOpenError gives CreateFile and NtCreateFile the same
// portable absence identity exposed by os package file operations. Reparse,
// directory, permission, and every other error remain unchanged and fail closed.
func platformNormalizeOpenError(err error) error {
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) || errors.Is(err, os.ErrNotExist) {
		return os.ErrNotExist
	}
	return err
}

func rejectWindowsReparseHandle(file *os.File, wantDirectory bool) error {
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &info); err != nil {
		return err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return windows.ERROR_CANT_ACCESS_FILE
	}
	isDirectory := info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0
	if isDirectory != wantDirectory {
		return windows.ERROR_DIRECTORY
	}
	return nil
}

func windowsNTStatusErr(err error) error {
	var status windows.NTStatus
	if errors.As(err, &status) {
		return status.Errno()
	}
	return err
}

type repositoryFileRenameInformation struct {
	ReplaceIfExists uint32
	RootDirectory   windows.Handle
	FileNameLength  uint32
	FileName        [1]uint16
}

func platformRenameAt(fromDir *os.File, from string, toDir *os.File, to string) error {
	source, err := windowsOpenRelative(
		fromDir,
		from,
		windows.DELETE|windows.FILE_READ_ATTRIBUTES|windows.SYNCHRONIZE,
		windows.FILE_OPEN,
		0,
		windows.FILE_NON_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT,
	)
	if err != nil {
		return err
	}
	defer source.Close()
	if err := rejectWindowsReparseHandle(source, false); err != nil {
		return err
	}
	nameUTF16, err := windows.UTF16FromString(to)
	if err != nil {
		return err
	}
	nameBytes := (len(nameUTF16) - 1) * 2
	var placeholder repositoryFileRenameInformation
	bufferSize := int(unsafe.Offsetof(placeholder.FileName)) + nameBytes
	buffer := make([]byte, bufferSize)
	info := (*repositoryFileRenameInformation)(unsafe.Pointer(&buffer[0]))
	info.ReplaceIfExists = 1
	info.RootDirectory = windows.Handle(toDir.Fd())
	info.FileNameLength = uint32(nameBytes)
	copy((*[windows.MAX_LONG_PATH]uint16)(unsafe.Pointer(&info.FileName[0]))[:nameBytes/2:nameBytes/2], nameUTF16[:len(nameUTF16)-1])
	var status windows.IO_STATUS_BLOCK
	return windows.NtSetInformationFile(
		windows.Handle(source.Fd()),
		&status,
		&buffer[0],
		uint32(len(buffer)),
		windows.FileRenameInformation,
	)
}

func platformUnlinkAt(parent *os.File, name string) error {
	file, err := windowsOpenRelative(
		parent,
		name,
		windows.DELETE|windows.FILE_READ_ATTRIBUTES|windows.SYNCHRONIZE,
		windows.FILE_OPEN,
		0,
		windows.FILE_NON_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT,
	)
	if err != nil {
		return err
	}
	defer file.Close()
	deleteFile := byte(1)
	var status windows.IO_STATUS_BLOCK
	return windows.NtSetInformationFile(
		windows.Handle(file.Fd()),
		&status,
		&deleteFile,
		1,
		windows.FileDispositionInformation,
	)
}

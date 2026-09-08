//go:build !windows

package storage

import (
	"os"

	"golang.org/x/sys/unix"
)

func platformLockFile(f *os.File, shared bool) error {
	mode := unix.LOCK_EX
	if shared {
		mode = unix.LOCK_SH
	}
	return unix.Flock(int(f.Fd()), mode)
}

func platformUnlockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}

func platformOpenDirectory(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

func platformOpenDirectoryAt(parent *os.File, name string, create bool) (*os.File, error) {
	flags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC
	fd, err := unix.Openat(int(parent.Fd()), name, flags, 0)
	if err == nil {
		return os.NewFile(uintptr(fd), name), nil
	}
	if !create || err != unix.ENOENT {
		return nil, err
	}
	if mkdirErr := unix.Mkdirat(int(parent.Fd()), name, 0755); mkdirErr != nil && mkdirErr != unix.EEXIST {
		return nil, mkdirErr
	}
	fd, err = unix.Openat(int(parent.Fd()), name, flags, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func platformOpenRegularFileAt(parent *os.File, name string, flags int, perm os.FileMode) (*os.File, error) {
	fd, err := unix.Openat(int(parent.Fd()), name, flags|unix.O_NOFOLLOW|unix.O_CLOEXEC, uint32(perm.Perm()))
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func platformRenameAt(fromDir *os.File, from string, toDir *os.File, to string) error {
	return unix.Renameat(int(fromDir.Fd()), from, int(toDir.Fd()), to)
}

func platformUnlinkAt(parent *os.File, name string) error {
	return unix.Unlinkat(int(parent.Fd()), name, 0)
}

// Package netlink listens to kernel events, filters power events and reports them back.
package netlink

import (
	"bytes"
	"context"
	"encoding/binary"
	"sync"

	"golang.org/x/sys/unix"
)

// KOBJECT_UEVENT_GROUP is the bitmask for listening to kernel uevents.
const KOBJECT_UEVENT_GROUP = 1

var (
	powerEventFilters = [...][]byte{
		[]byte("SUBSYSTEM=power_supply"),
		[]byte("ACTION=change"),
	}
)

// Listen subscribes to the kernel events and executes a callback on power events only.
func Listen(ctx context.Context, onEvent func(data []byte)) error {
	fd, err := unix.Socket(
		unix.AF_NETLINK,
		unix.SOCK_RAW|unix.SOCK_CLOEXEC|unix.SOCK_NONBLOCK,
		unix.NETLINK_KOBJECT_UEVENT,
	)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	addr := &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
		Groups: KOBJECT_UEVENT_GROUP,
	}
	if err := unix.Bind(fd, addr); err != nil {
		return err
	}

	efd, err := unix.Eventfd(0, unix.EFD_CLOEXEC|unix.EFD_NONBLOCK)
	if err != nil {
		return err
	}
	defer unix.Close(efd)

	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return err
	}
	defer unix.Close(epfd)

	// Register event fd for context cancellation
	if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, efd, &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(efd),
	}); err != nil {
		return err
	}

	// Register netlink fd for read events
	if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(fd),
	}); err != nil {
		return err
	}

	// When ctx is done, write to eventfd to wake epoll
	var once sync.Once
	stop := func() {
		once.Do(func() {
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], 1)
			unix.Write(efd, b[:])
		})
	}

	killer := make(chan struct{})
	defer close(killer)

	go func(killer chan struct{}) {
		select {
		case <-killer:
			return
		case <-ctx.Done():
			stop()
		}
	}(killer)

	buf := make([]byte, 4096)
	events := make([]unix.EpollEvent, 2)
	for {
		n, err := unix.EpollWait(epfd, events, -1)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return err
		}
		for i := 0; i < n; i++ {
			switch int(events[i].Fd) {
			case fd:
				nread, _, err := unix.Recvfrom(fd, buf, 0)
				if err != nil {
					continue
				}
				if isPowerEvent(buf[:nread]) {
					eventData := make([]byte, nread)
					copy(eventData, buf[:nread])
					onEvent(eventData)
				}
			case efd:
				// Context cancelled
				// read the data before bail out
				var tmp [8]byte
				unix.Read(efd, tmp[:])
				return ctx.Err()
			}
		}
	}
}

// isPowerEvent reports if the kernel event is a power event.
func isPowerEvent(event []byte) bool {
	for _, filter := range powerEventFilters {
		if !bytes.Contains(event, filter) {
			return false
		}
	}
	return true
}

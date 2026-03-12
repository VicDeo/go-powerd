// netlink package listens to kernel events, filters power events and reports them back.
package netlink

import (
	"bytes"
	"context"
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

// Listen subscribes to the kernel events and executes a callback on all power events.
func Listen(ctx context.Context, onEvent func(data []byte)) error {
	fd, err := unix.Socket(
		unix.AF_NETLINK,
		unix.SOCK_RAW,
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

	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return err
	}
	defer unix.Close(epfd)

	var pipeFds [2]int
	if err := unix.Pipe(pipeFds[:]); err != nil {
		return err
	}
	r, w := pipeFds[0], pipeFds[1]
	defer unix.Close(r)
	defer unix.Close(w)

	// Register netlink fd for read events
	if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(fd),
	}); err != nil {
		return err
	}
	// Register pipe read end for context cancellation
	if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, r, &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(r),
	}); err != nil {
		return err
	}

	// When ctx is done, write to pipe to wake epoll
	var once sync.Once
	stop := func() { once.Do(func() { unix.Write(w, []byte{1}) }) }
	go func() {
		<-ctx.Done()
		stop()
	}()

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
					onEvent(buf[:nread])
				}
			case r:
				// Context cancelled or pipe readable
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

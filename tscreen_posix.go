// +build solaris

// Copyright 2017 The TCell Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tcell

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

type termiosPrivate struct {
	tio *unix.Termios
}

const (
	// These are for missing CBAUDEXT and CIBAUDEXT.
	// The values are fixed for Solaris and illumos, and cannot ever
	// change without breaking applications.
	cBaudExt  = 0x200000
	ciBaudExt = 0x400000
)

// getbaud is sort of cfgetospeed, but in Go.
func getbaud(tios *unix.Termios) int {
	// First we mask off the rate by looking at the Cflag.
	bval := tios.Cflag & unix.CBAUD
	if (tios.Cflag & cBaudExt) != 0 {
		bval += unix.CBAUD + 1
	}

	// This gives us the appropriate BXXX value, so convert
	switch bval {
	case unix.B0:
		return 0
	case unix.B50:
		return 50
	case unix.B75:
		return 75
	case unix.B110:
		return 110
	case unix.B134:
		return 134
	case unix.B150:
		return 150
	case unix.B200:
		return 200
	case unix.B300:
		return 300
	case unix.B600:
		return 600
	case unix.B1200:
		return 1200
	case unix.B1800:
		return 1800
	case unix.B2400:
		return 2400
	case unix.B4800:
		return 4800
	case unix.B9600:
		return 9600
	case unix.B19200:
		return 19200
	case unix.B38400:
		return 38400
	case unix.B57600:
		return 57600
	case unix.B76800:
		return 76800
	case unix.B115200:
		return 115200
	case unix.B153600:
		return 153600
	case unix.B230400:
		return 230400
	case unix.B307200:
		return 307200
	case unix.B460800:
		return 460800
	case unix.B921600:
		return 921600
	}
	return 0
}

func (t *tScreen) termioInit() error {
	var e error
	var raw *unix.Termios
	var tio *unix.Termios

	if t.in, e = os.OpenFile("/dev/tty", os.O_RDONLY, 0); e != nil {
		goto failed
	}
	if t.out, e = os.OpenFile("/dev/tty", os.O_WRONLY, 0); e != nil {
		goto failed
	}

	t.tiosp = &termiosPrivate{}

	tio, e = unix.IoctlGetTermios(int(t.out.Fd()), unix.TCGETS)
	if e != nil {
		goto failed
	}

	t.tiosp.tio = tio
	t.baud = getbaud(tio)

	// make a local copy, to make it raw
	raw = &unix.Termios{
		Cflag: tio.Cflag,
		Oflag: tio.Oflag,
		Iflag: tio.Iflag,
		Lflag: tio.Lflag,
		Cc:    tio.Cc,
	}

	raw.Iflag &^= (unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.INLCR |
		unix.IGNCR | unix.ICRNL | unix.IXON)
	raw.Oflag &^= unix.OPOST
	raw.Lflag &^= (unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN)
	raw.Cflag &^= (unix.CSIZE | unix.PARENB)
	raw.Cflag |= unix.CS8

	// This is setup for blocking reads.  In the past we attempted to
	// use non-blocking reads, but now a separate input loop and timer
	// copes with the problems we had on some systems (BSD/Darwin)
	// where close hung forever.
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	e = unix.IoctlSetTermios(int(t.out.Fd()), unix.TCSETS, raw)
	if e != nil {
		goto failed
	}

	signal.Notify(t.sigwinch, syscall.SIGWINCH)

	if w, h, e := t.getWinSize(); e == nil && w != 0 && h != 0 {
		t.cells.Resize(w, h)
	}

	return nil

failed:
	if t.in != nil {
		t.in.Close()
	}
	if t.out != nil {
		t.out.Close()
	}
	return e
}

func (t *tScreen) termioFini() {

	signal.Stop(t.sigwinch)

	<-t.indoneq

	if t.out != nil && t.tiosp != nil {
		unix.IoctlSetTermios(int(t.out.Fd()), unix.TCSETSF, t.tiosp.tio)
		t.out.Close()
	}
	if t.in != nil {
		t.in.Close()
	}
}

func (t *tScreen) getWinSize() (int, int, error) {
	wsz, err := unix.IoctlGetWinsize(int(t.out.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return -1, -1, err
	}
	return int(wsz.Col), int(wsz.Row), nil
}

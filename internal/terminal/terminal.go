package terminal

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gofiber/contrib/websocket"
	"github.com/rs/zerolog/log"
)

// Message types for WebSocket communication
const (
	MsgTypeInput  = "input"
	MsgTypeOutput = "output"
	MsgTypeResize = "resize"
)

// WsMessage represents a WebSocket message
type WsMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

// setWinsize sets the terminal window size
func setWinsize(f *os.File, cols, rows int) error {
	ws := struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}{
		Row: uint16(rows),
		Col: uint16(cols),
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		syscall.TIOCSWINSZ,
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// HandleWebSocket handles the WebSocket connection for the terminal
func HandleWebSocket(c *websocket.Conn) {
	log.Info().Str("remote", c.RemoteAddr().String()).Msg("Terminal WebSocket connected")

	// Start a shell - use /bin/sh for Alpine
	cmd := exec.Command("/bin/sh")
	cmd.Dir = "/tmp" // Start in /tmp, not /app (safer)
	cmd.Env = []string{
		"TERM=xterm-256color",
		"HOME=/tmp",
		"PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
		"PS1=gagos$ ",
	}

	// Create pseudo-terminal
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start pty")
		c.WriteJSON(WsMessage{Type: MsgTypeOutput, Data: "Error: Failed to start terminal\r\n"})
		return
	}
	defer func() {
		ptmx.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// Set initial size
	setWinsize(ptmx, 80, 24)

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Read from pty and send to WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			select {
			case <-done:
				return
			default:
				n, err := ptmx.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Debug().Err(err).Msg("PTY read error")
					}
					return
				}
				if n > 0 {
					if err := c.WriteJSON(WsMessage{
						Type: MsgTypeOutput,
						Data: string(buf[:n]),
					}); err != nil {
						log.Debug().Err(err).Msg("WebSocket write error")
						return
					}
				}
			}
		}
	}()

	// Read from WebSocket and write to pty
	for {
		var msg WsMessage
		if err := c.ReadJSON(&msg); err != nil {
			log.Debug().Err(err).Msg("WebSocket read error")
			break
		}

		switch msg.Type {
		case MsgTypeInput:
			if _, err := ptmx.Write([]byte(msg.Data)); err != nil {
				log.Debug().Err(err).Msg("PTY write error")
				break
			}
		case MsgTypeResize:
			if msg.Cols > 0 && msg.Rows > 0 {
				setWinsize(ptmx, msg.Cols, msg.Rows)
			}
		}
	}

	close(done)
	wg.Wait()
	log.Info().Str("remote", c.RemoteAddr().String()).Msg("Terminal WebSocket disconnected")
}

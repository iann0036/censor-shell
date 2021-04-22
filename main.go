package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/inancgumus/screen"
	"github.com/mitchellh/go-homedir"
	"github.com/riywo/loginshell"
	"golang.org/x/term"
	"gopkg.in/ini.v1"
)

var iniCfg *ini.File

// CapturingFilterWriter is a writer that remembers
// data written to it and passes it to w
type CapturingFilterWriter struct {
	buf  bytes.Buffer
	w    io.Writer
	last time.Time
}

// NewCapturingFilterWriter creates new CapturingFilterWriter
func NewCapturingFilterWriter(w io.Writer) *CapturingFilterWriter {
	cfw := &CapturingFilterWriter{
		w:    w,
		last: time.Now(),
	}

	go func() {
		for {
			time.Sleep(time.Duration(80+rand.Int63n(20)) * time.Millisecond) // Slight randomness as primitive defense to timing attacks
			if time.Since(cfw.last) < 10*time.Millisecond {                  // TODO: Partials edge case here
				time.Sleep(10 * time.Millisecond)
			}
			cfw.Flush()
		}
	}()

	return cfw
}

// Write writes data to the buffer, returns number of bytes written and an error
func (w *CapturingFilterWriter) Write(d []byte) (int, error) {
	w.last = time.Now()

	return w.buf.Write(d)
}

// Bytes retrieves the byte buffer and flushes
func (w *CapturingFilterWriter) Flush() (int, error) {
	b := w.buf.Bytes()
	w.buf.Reset()

	// TODO: optimize performance
	repl := string(b)
	for _, section := range iniCfg.Sections() {
		if section.HasKey("pattern") && section.HasKey("replacement") {
			pattern := regexp.MustCompile(section.Key("pattern").String())
			repl = pattern.ReplaceAllString(repl, section.Key("replacement").String())
		}
	}

	return w.w.Write([]byte(repl))
}

func main() {
	// Read config
	var err error
	home, _ := homedir.Expand("~/.censor-shell")
	iniCfg, err = ini.Load(home)
	if err != nil {
		fmt.Printf("No config found at %s, exiting...\n", home)
		os.Exit(0)
	}

	// Clear terminal
	screen.Clear()
	screen.MoveTopLeft()

	// Determine login shell and use
	shell, err := loginshell.Shell()
	if err != nil {
		shell = "sh"
	}
	c := exec.Command(shell)

	// Start the command with a pty
	ptmx, err := pty.Start(c)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = ptmx.Close() }() // Best effort

	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH                        // Initial resize
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done

	// Set stdin in raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort

	// Copy stdin to the pty
	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()

	// Send the pty to the capturing filter writer
	cfw := NewCapturingFilterWriter(os.Stdout)
	_, _ = io.Copy(cfw, ptmx)

	fmt.Printf("\n[%s is terminating]\n", os.Args[0])
}

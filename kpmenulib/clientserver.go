package kpmenulib

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Packet is the data sent by the client to the server listener
type Packet struct {
	CliArguments []string
}
type PacketResp struct {
	Output string
}

// StartClient sends a packet to the server listener
func StartClient() error {
	port, err := getPort()
	if err != nil {
		return err
	}

	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Send the packet
	enc := gob.NewEncoder(conn)
	enc.Encode(Packet{CliArguments: os.Args[1:]})
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	dec := gob.NewDecoder(conn)
	out := PacketResp{}
	err = dec.Decode(&out)
	fmt.Fprintf(os.Stdout, "%s", out.Output)
	return err
}

// StartServer starts to listen for client packets
func StartServer(m *Menu) (err error) {
	if m.Configuration.Flags.Daemon {
		log.Printf("Executing as daemon")
	}

	if m.Configuration.General.NoCache && !m.Configuration.Flags.Daemon {
		// Directly execute kpmenu
		var out PacketResp
		fatal := m.Execute(&out)
		if fatal {
			os.Exit(1) // Set exit code to 1 and exit
		}
		fmt.Fprintf(os.Stdout, "%s", out.Output)
	} else {
		// Handle packet request
		handlePacket := func(packet Packet, out *PacketResp) (fatal bool) {
			log.Printf("received a client call with args \"%v\"", packet.CliArguments)
			m.Configuration.Flags.Autotype = false
			m.CliArguments = packet.CliArguments
			cc := InitializeFlags(packet.CliArguments)
			clientConfig := NewConfiguration()
			if err := LoadConfig(cc, clientConfig); err != nil {
				log.Fatalf("loading client config: %s", err)
				return false
			}
			m.Configuration = clientConfig
			defer m.ReloadConfig()

			return m.Show(out)
		}

		// Execute kpmenu for the first time, if not a daemon
		exit := false
		var out PacketResp
		if !m.Configuration.Flags.Daemon {
			exit = m.Execute(&out)
			if out.Output != "" {
				fmt.Fprintf(os.Stdout, "%s", out.Output)
			}

		}

		// If exit is false (cache on) listen for client calls
		if !exit {
			err = setupListener(m, handlePacket)
		}
	}
	return
}

func setupListener(m *Menu, handlePacket func(Packet, *PacketResp) bool) error {
	// Listen for client calls
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}
	tcpListener := listener.(*net.TCPListener)
	defer tcpListener.Close()

	// Get used port
	_, port, _ := net.SplitHostPort(listener.Addr().String())

	// Save port
	if err := savePort(port); err != nil {
		return err
	}

	exit := false
	for !exit {
		if !m.Configuration.Flags.Daemon {
			// If not a daemon prepare cache time
			remainingCacheTime := m.Configuration.General.CacheTimeout - time.Since(m.CacheStart)
			tcpListener.SetDeadline(time.Now().Add(remainingCacheTime))
		}

		// Listen to calls
		conn, err := listener.Accept()
		if err != nil {
			netErr := err.(*net.OpError)
			if netErr.Timeout() {
				log.Print("cache timed out")
				return nil
			}
			return err
		}
		defer conn.Close()

		// Go routine to handle input
		ch := make(chan Packet)
		errCh := make(chan error)
		go func(ch chan Packet, errCh chan error) {
			dec := gob.NewDecoder(conn)
			var packet Packet
			err := dec.Decode(&packet)
			if err != nil {
				if err != io.EOF {
					errCh <- err
				} else {
					return
				}
			}
			ch <- packet
		}(ch, errCh)

		// Handle received input
		timeout := time.Tick(3 * time.Second) // Timeout of 3 seconds - to avoid problems
		var output PacketResp
		select {
		case packet := <-ch:
			// Received the data
			fatal := handlePacket(packet, &output)
			enc := gob.NewEncoder(conn)
			enc.Encode(output)
			exit = (fatal && !m.Configuration.Flags.Daemon)
		case err := <-errCh:
			// Received an error
			enc := gob.NewEncoder(conn)
			enc.Encode(output)
			return err
		case <-timeout:
			enc := gob.NewEncoder(conn)
			enc.Encode(output)
			// Timed out
			log.Printf("received request is timed out")
		}
	}

	return nil
}

func makeCacheFolder() error {
	if err := os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".cache/kpmenu/"), 0755); err != nil {
		return fmt.Errorf("failed to make cache folder: %v", err)
	}
	return nil
}

func savePort(port string) (err error) {
	if err = makeCacheFolder(); err == nil {
		if err = os.WriteFile(
			filepath.Join(os.Getenv("HOME"), ".cache/kpmenu/server.port"),
			[]byte(port),
			0644,
		); err != nil {
			return fmt.Errorf("failed to make server port cache file: %v", err)
		}
	}
	return err
}

func getPort() (string, error) {
	data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".cache/kpmenu/server.port"))
	return string(data), err
}

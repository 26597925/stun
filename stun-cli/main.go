package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/cydev/stun"
)

const (
	version = "0.1"
)

func wrapLogrus(f func(c *cli.Context) error) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		err := f(c)
		if err != nil {
			logrus.Errorln("discover error:", err)
		}
		return err
	}
}

func discover(c *cli.Context) error {
	conn, err := net.Dial("udp", stun.Normalize(c.String("server")))
	if err != nil {
		return err
	}
	m := stun.AcquireMessage()
	m.Type = stun.MessageType{
		Method: stun.MethodBinding,
		Class:  stun.ClassRequest,
	}
	m.TransactionID = stun.NewTransactionID()
	m.AddSoftware("cydev/stun alpha")
	m = stun.AcquireFields(stun.Message{
		TransactionID: stun.NewTransactionID(),
		Type: stun.MessageType{
			Method: stun.MethodBinding,
			Class: stun.ClassRequest,
		},
	})
	m.WriteHeader()
	timeout := 100 * time.Millisecond
	for i := 0; i < 9; i++ {
		_, err := m.WriteTo(conn)
		if err != nil {
			return err
		}
		if err = conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			return err
		}
		if timeout < 1600*time.Millisecond {
			timeout *= 2
		}
		var (
			ip   net.IP
			port int
		)
		if err == nil {
			mRec := stun.AcquireMessage()
			if _, err = mRec.ReadFrom(conn); err != nil {
				return err
			}
			if mRec.TransactionID != m.TransactionID {
				return errors.New("TransactionID missmatch")
			}
			ip, port, err = mRec.GetXORMappedAddress()
			if err != nil {
				return err
			}
			fmt.Println(ip, port)
			stun.ReleaseMessage(mRec)
			break
		} else {
			if !err.(net.Error).Timeout() {
				return err
			}
		}
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "stun"
	app.Usage = "command line client for STUN"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "server",
			Value:       "ci.cydev.ru",
			Usage:       "STUN server address",
		},
	}
	app.Action = wrapLogrus(discover)
	app.Version = version
	app.Run(os.Args)
}